package tools

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/julienlegoux/kern-link/ai"

	"github.com/julienlegoux/external-reviewer/internal/confine"
)

// errNotUTF8 marks a file whose content failed the UTF-8 check partway
// through scanning. It never leaves this file: readOneFile turns it into the
// inline explanatory message the model sees.
var errNotUTF8 = errors.New("not valid utf-8")

// DefaultReadLimit and MaxReadLimit are the window's default and ceiling, in
// lines. The default is generous enough that most files are read whole in one
// call; the ceiling exists so a single pathological request cannot pull an
// unbounded amount of one file into a turn (specs 09 — Read-only tool
// contract, "read_file 5000 lines per file").
const (
	DefaultReadLimit = 2000
	MaxReadLimit     = 5000
)

// scanBufferSize and maxScanTokenSize bound the line scanner's buffer: large
// enough that an ordinary long line (a minified asset, a generated file)
// never trips bufio.Scanner's default 64 KiB token limit, without reading a
// whole file into memory to find out — the cap is on the token, not the file.
const (
	scanBufferSize   = 64 * 1024
	maxScanTokenSize = 10 * 1024 * 1024
)

// readFileDescription is the model's only instruction on how this tool
// behaves: the caller owns the system prompt, so it stands alone (specs 09,
// specs 08). It states explicitly that a batch failure is partial, since that
// is the property the tool exists to offer over reading one path at a time.
const readFileDescription = `Read the contents of one or more files in the repository under review, each under its own header naming the path and the line range returned.

paths is a list of repo-relative, /-separated paths — pass several in one call rather than calling this tool once per file. A batch failure is partial, not total: a bad path (missing, outside what this run was granted, a directory, or not valid UTF-8 text) is reported inline under that path's own header with an explanation, while every other path in the batch still returns its content, and the call itself is never reported as an error.

offset (1-based line number, default 1) and limit (lines, default 2000, capped at 5000) select the window returned for every path in the batch. Each line is prefixed "<line number>\t". A window that stops before the end of a file ends with a line reading "[truncated: lines <start>–<end> of <total>]" naming the real total; a file that fits within the window carries no such line.`

// readFileParameters is the wire schema, snake_case like the tool name and
// its arguments, because it is a contract with the model rather than Go API
// (CONVENTIONS § Naming).
var readFileParameters = ai.JSONSchema(`{
  "type": "object",
  "properties": {
    "paths": {
      "type": "array",
      "items": {"type": "string"},
      "description": "Repo-relative, /-separated paths to read, in the order they should be returned."
    },
    "offset": {
      "type": "integer",
      "description": "1-based line number to start each file's window at. Defaults to 1."
    },
    "limit": {
      "type": "integer",
      "description": "Maximum number of lines to return per file, from offset. Defaults to 2000, capped at 5000."
    }
  },
  "required": ["paths"],
  "additionalProperties": false
}`)

// ReadFile builds the tool over the run's confinement. Every path resolves
// through scope, which is the only way this tool — or any other in this
// package — reaches a file (CONVENTIONS § Code style & formatting; specs 10).
func ReadFile(scope *confine.Scope) Tool {
	return Tool{
		Declaration: ai.Tool{
			Name:        "read_file",
			Description: readFileDescription,
			Parameters:  readFileParameters,
		},
		Handler: func(_ context.Context, arguments map[string]any) Result {
			return readFiles(scope, arguments)
		},
	}
}

// readFiles parses the call-level arguments and renders one section per
// requested path. Only a malformed call — a missing or ill-typed parameter —
// is reported as a tool error; a per-path failure is text under that path's
// own header, and the call itself succeeds (specs 09: "a batch where one path
// was a typo must still return the other seven").
func readFiles(scope *confine.Scope, arguments map[string]any) Result {
	paths, err := pathsArgument(arguments)
	if err != nil {
		return errorResult(err.Error())
	}
	offset, err := wholeNumberArgument(arguments, "offset", 1)
	if err != nil {
		return errorResult(err.Error())
	}
	if offset < 1 {
		return errorResult("the offset parameter must be at least 1")
	}
	limit, err := wholeNumberArgument(arguments, "limit", DefaultReadLimit)
	if err != nil {
		return errorResult(err.Error())
	}
	if limit < 1 {
		return errorResult("the limit parameter must be a positive number of lines")
	}
	if limit > MaxReadLimit {
		limit = MaxReadLimit
	}

	sections := make([]string, 0, len(paths))
	for _, path := range paths {
		sections = append(sections, readOneFile(scope, path, offset, limit))
	}
	return Result{Text: strings.Join(sections, "\n")}
}

// readOneFile renders one path's section: its content window on success, or
// an explanatory message under the same header shape on any of the ways a
// single path can fail — a refusal, an absence, a directory, or content that
// is not valid UTF-8. Nothing here can make the batch call itself an error.
func readOneFile(scope *confine.Scope, wirePath string, offset, limit int) string {
	file, err := scope.Open(wirePath)
	if err != nil {
		return sectionHeader(wirePath) + err.Error() + "\n"
	}
	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil {
		return sectionHeader(wirePath) + fmt.Sprintf("reading %q: %v", wirePath, err) + "\n"
	}
	if info.IsDir() {
		return sectionHeader(wirePath) + fmt.Sprintf("reading %q: the path is a directory, not a file", wirePath) + "\n"
	}

	window, err := scanWindow(file, offset, limit)
	if err != nil {
		if errors.Is(err, errNotUTF8) {
			return sectionHeader(wirePath) + fmt.Sprintf("reading %q: the file is not valid UTF-8 text", wirePath) + "\n"
		}
		return sectionHeader(wirePath) + fmt.Sprintf("reading %q: the file could not be read: %v", wirePath, err) + "\n"
	}

	if window.total == 0 {
		return fmt.Sprintf("=== %s (empty file) ===\n", wirePath)
	}
	if offset > window.total {
		return sectionHeader(wirePath) + fmt.Sprintf("reading %q: offset %d is beyond the file's %d lines", wirePath, offset, window.total) + "\n"
	}
	return renderWindow(wirePath, offset, window)
}

// sectionHeader is the header a failed path gets: its own name and nothing
// else, since no range was ever read.
func sectionHeader(wirePath string) string {
	return fmt.Sprintf("=== %s ===\n", wirePath)
}

// readWindow is the result of scanning one file: the lines inside the
// requested window, kept as text, and the file's real total line count,
// counted by scanning to the end without keeping what falls outside the
// window — the streaming half of the tool's contract (specs 09: "content
// streamed to the cap rather than read whole and trimmed").
type readWindow struct {
	lines []string
	total int
}

// scanWindow reads wirePath's content line by line, collecting only the
// lines between offset and offset+limit-1 while still counting every line in
// the file, so the truncation marker's total is always the real one. It
// returns an error the instant a line is not valid UTF-8 — the whole file is
// then reported as unreadable text rather than as a partially-collected
// window, since there is nothing safe to show past that point.
func scanWindow(file *os.File, offset, limit int) (readWindow, error) {
	end := offset + limit - 1

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, scanBufferSize), maxScanTokenSize)

	window := readWindow{}
	for scanner.Scan() {
		window.total++
		raw := scanner.Bytes()
		if !utf8.Valid(raw) {
			return readWindow{}, errNotUTF8
		}
		if window.total >= offset && window.total <= end {
			window.lines = append(window.lines, string(raw))
		}
	}
	if err := scanner.Err(); err != nil {
		return readWindow{}, err
	}
	return window, nil
}

// renderWindow writes one file's header, its numbered lines, and — only when
// the window stopped short of the file's end — the truncation marker naming
// both the range actually returned and the file's real total.
func renderWindow(wirePath string, offset int, window readWindow) string {
	last := offset + len(window.lines) - 1

	var out strings.Builder
	fmt.Fprintf(&out, "=== %s (lines %d–%d) ===\n", wirePath, offset, last)
	for i, line := range window.lines {
		fmt.Fprintf(&out, "%d\t%s\n", offset+i, line)
	}
	if window.total > last {
		fmt.Fprintf(&out, "[truncated: lines %d–%d of %d]\n", offset, last, window.total)
	}
	return out.String()
}

// pathsArgument reads the required paths parameter: a non-empty array of
// strings. Anything else is the model's mistake to learn from — a malformed
// call, not a per-path failure — so it is refused here rather than reaching
// readOneFile at all.
func pathsArgument(arguments map[string]any) ([]string, error) {
	raw, present := arguments["paths"]
	if !present || raw == nil {
		return nil, errors.New("the paths parameter is required: at least one repo-relative path")
	}
	items, ok := raw.([]any)
	if !ok {
		return nil, errors.New("the paths parameter must be an array of strings")
	}
	if len(items) == 0 {
		return nil, errors.New("the paths parameter must not be empty: at least one repo-relative path")
	}
	paths := make([]string, 0, len(items))
	for _, item := range items {
		value, ok := item.(string)
		if !ok {
			return nil, errors.New("the paths parameter must be an array of strings")
		}
		paths = append(paths, value)
	}
	return paths, nil
}

// wholeNumberArgument reads one integer parameter, accepting both a
// JSON-decoded number (float64, the shape a real tool call arrives in) and a
// Go int (the shape a test builds arguments in directly, without a JSON round
// trip). A value of the wrong type or with a fractional part is the model's
// mistake to learn from, so it comes back as a call-level refusal naming the
// parameter — unlike search's own intArgument, which clamps into a range
// instead of erroring, this tool's offset/limit have no such range to clamp
// into below the cap, so a fraction or an out-of-range value is refused
// rather than silently rounded.
func wholeNumberArgument(arguments map[string]any, name string, fallback int) (int, error) {
	raw, present := arguments[name]
	if !present || raw == nil {
		return fallback, nil
	}
	switch value := raw.(type) {
	case int:
		return value, nil
	case int64:
		return int(value), nil
	case float64:
		if value != float64(int(value)) {
			return 0, fmt.Errorf("the %s parameter must be a whole number", name)
		}
		return int(value), nil
	default:
		return 0, fmt.Errorf("the %s parameter must be a number", name)
	}
}
