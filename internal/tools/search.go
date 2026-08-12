package tools

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/julienlegoux/kern-link/ai"

	"github.com/julienlegoux/external-reviewer/internal/confine"
)

// The four numbers that decide how large one search result can be. They are
// stated constants rather than tuning knobs: the binding limit is the model's
// context window (specs 09), and a result is bounded by all four at once —
// at most MaxSearchMatches matches, each carrying at most MaxSearchContext
// lines either side, each line at most maxSearchLineRunes long.
const (
	// DefaultSearchMatches is what a call that says nothing about size gets.
	DefaultSearchMatches = 100
	// MaxSearchMatches caps one result, and a larger max_results is clamped
	// to it rather than refused — the same 200 `list` caps at, because both
	// tools answer "what is there".
	MaxSearchMatches = 200
	// DefaultSearchContext is the two lines either side that are the reason
	// this tool takes a context parameter at all: a match without its
	// surroundings usually forces a follow-up read_file (specs 09).
	DefaultSearchContext = 2
	// MaxSearchContext caps the window. Without it a single call could ask
	// for 200 matches with a thousand lines around each and fill the context
	// window this cap exists to protect.
	MaxSearchContext = 10
)

const (
	// maxSearchFileBytes skips files too large to be source a reviewer reads
	// (specs 11). It also bounds one line: a file this size cannot contain a
	// longer one, so the scanner below can never meet a line it must refuse.
	maxSearchFileBytes = 2 << 20
	// binarySniffBytes is the window a NUL byte marks a file as binary in
	// (specs 11) — a compiled artefact or an image is not material a reviewer
	// reads, and a match inside one is noise its context lines multiply.
	binarySniffBytes = 8 << 10
	// maxSearchLineRunes clips one rendered line. A minified bundle is a
	// text file by every test above, and one of its lines would otherwise be
	// the whole result.
	maxSearchLineRunes = 500
	// lineClippedMarker ends a line maxSearchLineRunes cut, so a clipped line
	// is never mistaken for a short one — truncation is announced wherever it
	// happens (specs 09).
	lineClippedMarker = "…"
	// groupSeparator stands between two windows that do not touch, so a
	// reviewer never reads two distant regions of a file as one passage. It
	// is grep's own separator, and it appears only when context lines do.
	groupSeparator = "--"
)

// wholeGrantedSurface is the path parameter's default. It is not resolved
// through the Scope the way a real subtree is: with `--allow docs`, "." is
// outside the allow-list as a path, while as a *default* it means "everywhere
// this run may look" — which is exactly the enumeration, already confined.
const wholeGrantedSurface = "."

// searchDescription is the model's only instruction on how to grep this
// repository: the caller owns the system prompt, so this stands alone (specs
// 09, specs 08).
const searchDescription = `Search the repository under review for a regular expression, returning each matching line with the lines around it.

The pattern is RE2 syntax (Go's regexp, the same dialect ripgrep uses by default): character classes, groups, alternation, anchors and repetition all work. Backreferences and lookaround do not exist in RE2 and a pattern using them is rejected — rewrite it without them.

A matching line is returned as path:line:text. A surrounding context line is returned as path-line-text, with dashes instead of colons, so the two are distinguishable at a glance; "--" on its own line separates two regions that are not adjacent. Paths are relative to the repository root and always use / as the separator, whichever operating system this runs on; pass them back to the other tools exactly as they appear here.

Parameters: "pattern" is required. "path" restricts the search to one subtree (default ".", the whole visible repository). "glob" restricts it to matching files, matched against the whole path, where * stays within a path segment and ** crosses them ("**/*.go"). "context" is how many lines are returned either side of each match (default 2, maximum 10; 0 returns matching lines only). "max_results" caps the matches returned (default 100, maximum 200).

What is visible is narrower than the repository: files ignored by .gitignore are never searched, nor is anything outside the subtrees this run was granted, nor files whose names mark them as credentials. Files over 2 MB and files that look binary are skipped, and a very long line is clipped with "…".

When more lines match than are returned, the last line reads "[truncated: showing 200 of N matches]" with the real total — narrow the pattern, the path or the glob to see the rest.`

// searchParameters is the wire schema, snake_case like the tool name, because
// it is a contract with the model rather than Go API (CONVENTIONS § Naming).
var searchParameters = ai.JSONSchema(`{
  "type": "object",
  "properties": {
    "pattern": {
      "type": "string",
      "description": "RE2 regular expression matched against each line. No backreferences or lookaround."
    },
    "path": {
      "type": "string",
      "description": "Repo-relative subtree to search, /-separated. Defaults to \".\", the whole visible repository."
    },
    "glob": {
      "type": "string",
      "description": "Restrict the files searched to those whose whole repo-relative path matches this glob; * stays within a path segment and ** crosses them."
    },
    "max_results": {
      "type": "integer",
      "description": "Maximum matches returned. Defaults to 100, clamped to 200."
    },
    "context": {
      "type": "integer",
      "description": "Lines returned either side of each match. Defaults to 2, clamped to 10; 0 returns matching lines only."
    }
  },
  "required": ["pattern"],
  "additionalProperties": false
}`)

// Search builds the tool over the run's confinement and the same enumeration
// `list` reads from, so what the reviewer can glob and what it can grep never
// disagree (specs 11).
//
// Matching is in-process regexp — RE2, so a model-supplied pattern cannot
// backtrack catastrophically — over content read through the Scope. No
// subprocess matcher: `rg` would read the filesystem on its own authority,
// outside the one *os.Root this product's confinement is (specs 11).
func Search(scope *confine.Scope, repository Enumerator) Tool {
	return Tool{
		Declaration: ai.Tool{
			Name:        "search",
			Description: searchDescription,
			Parameters:  searchParameters,
		},
		Handler: func(ctx context.Context, arguments map[string]any) Result {
			return searchRepository(ctx, scope, repository, arguments)
		},
	}
}

func searchRepository(ctx context.Context, scope *confine.Scope, repository Enumerator, arguments map[string]any) Result {
	query, err := newSearchQuery(scope, arguments)
	if err != nil {
		return errorResult(err.Error())
	}

	files, err := repository.Files(ctx)
	if err != nil {
		return errorResult(fmt.Sprintf("the repository could not be enumerated: %v", err))
	}

	matches := &matchSet{query: query}
	for _, file := range files {
		if ctx.Err() != nil {
			break
		}
		if !query.covers(file) {
			continue
		}
		// Re-resolved even though enumeration already did: a tool never reads
		// a path on the strength of another mechanism having named it, and
		// this is the gate that makes the allow-list and the sensitive-file
		// floor true of search rather than inherited by assumption.
		cleaned, err := scope.Resolve(file)
		if err != nil {
			continue
		}
		matches.scan(scope, cleaned)
	}

	if matches.total == 0 {
		return Result{Text: fmt.Sprintf("no matches for %q\n", query.pattern)}
	}
	return Result{Text: matches.render()}
}

// searchQuery is one call's parameters, already validated: everything that can
// be refused has been refused by the time it exists.
type searchQuery struct {
	pattern    string
	expression *regexp.Regexp
	// subtree is the cleaned path parameter, or "" for the whole granted
	// surface.
	subtree    string
	glob       string
	maxResults int
	context    int
}

func newSearchQuery(scope *confine.Scope, arguments map[string]any) (searchQuery, error) {
	pattern, err := stringArgument(arguments, "pattern", "")
	if err != nil {
		return searchQuery{}, err
	}
	if pattern == "" {
		return searchQuery{}, errors.New("the pattern parameter is required: search matches a regular expression against the repository's lines")
	}
	expression, err := regexp.Compile(pattern)
	if err != nil {
		return searchQuery{}, fmt.Errorf("the pattern %q is not a valid regular expression: %w", pattern, err)
	}

	subtree, err := searchSubtree(scope, arguments)
	if err != nil {
		return searchQuery{}, err
	}

	glob, err := stringArgument(arguments, "glob", "")
	if err != nil {
		return searchQuery{}, err
	}
	if glob != "" {
		if err := validateGlob(glob); err != nil {
			return searchQuery{}, fmt.Errorf("the glob %q is not a valid glob: %w", glob, err)
		}
	}

	maxResults, err := intArgument(arguments, "max_results", DefaultSearchMatches, 1, MaxSearchMatches)
	if err != nil {
		return searchQuery{}, err
	}
	surrounding, err := intArgument(arguments, "context", DefaultSearchContext, 0, MaxSearchContext)
	if err != nil {
		return searchQuery{}, err
	}

	return searchQuery{
		pattern:    pattern,
		expression: expression,
		subtree:    subtree,
		glob:       glob,
		maxResults: maxResults,
		context:    surrounding,
	}, nil
}

// searchSubtree resolves the path parameter through the run's confinement, so
// a subtree the invocation never granted is refused before a single file is
// opened — and refused by name, since "widen --allow" and "that file is a
// credential" are different things for a reviewer to learn (specs 10).
func searchSubtree(scope *confine.Scope, arguments map[string]any) (string, error) {
	raw, err := stringArgument(arguments, "path", wholeGrantedSurface)
	if err != nil {
		return "", err
	}
	if raw == wholeGrantedSurface {
		return "", nil
	}

	cleaned, err := scope.Resolve(raw)
	if err != nil {
		var refusal *confine.PathError
		if errors.As(err, &refusal) {
			return "", fmt.Errorf("the path %q cannot be searched: %w", raw, refusal.Reason)
		}
		return "", fmt.Errorf("the path %q cannot be searched: %w", raw, err)
	}
	return cleaned, nil
}

// covers answers whether one enumerated file is a candidate at all, which is
// the whole of what path and glob do: neither widens the file set, they only
// narrow what enumeration already confined.
func (q searchQuery) covers(file string) bool {
	if q.subtree != "" && file != q.subtree && !strings.HasPrefix(file, q.subtree+"/") {
		return false
	}
	if q.glob == "" {
		return true
	}
	matched, err := matchGlob(q.glob, file)
	return err == nil && matched
}

// textLine is one line of a file, as it will be rendered.
type textLine struct {
	number int
	text   string
}

// matchSet accumulates the rendered result across every file scanned.
//
// It counts every match and renders only the first maxResults of them: the
// truncation marker carries both real numbers, which means the matches beyond
// the cap have to be counted even though nothing is kept for them. Counting is
// a regexp over a line already read; keeping would be the context window this
// cap protects.
type matchSet struct {
	query      searchQuery
	out        []string
	total      int
	shown      int
	lastPath   string
	lastNumber int
}

// scan reads one file line by line, matching as it goes. Nothing is read
// whole: a match's preceding context comes from a ring buffer of the last
// `context` lines, and its following context from a countdown after it.
func (m *matchSet) scan(scope *confine.Scope, wirePath string) {
	file, err := scope.Open(wirePath)
	if err != nil {
		return
	}
	defer func() { _ = file.Close() }()

	reader, ok := readableText(file)
	if !ok {
		return
	}

	scanner := bufio.NewScanner(reader)
	// A line can be no longer than the file, and a file this long was already
	// skipped — so the scanner can never return ErrTooLong and a file is
	// never half-searched without saying so.
	scanner.Buffer(make([]byte, 0, bufio.MaxScanTokenSize), maxSearchFileBytes+1)

	var before []textLine
	after := 0
	for number := 1; scanner.Scan(); number++ {
		line := textLine{number: number, text: strings.TrimSuffix(scanner.Text(), "\r")}
		switch {
		case m.query.expression.MatchString(line.text):
			m.total++
			after = 0
			if m.shown < m.query.maxResults {
				m.emitBefore(wirePath, before)
				m.emit(wirePath, line, true)
				m.shown++
				after = m.query.context
			}
		case after > 0:
			m.emit(wirePath, line, false)
			after--
		}
		before = remember(before, line, m.query.context)
	}
}

// readableText decides whether a file is material a reviewer reads at all: not
// oversized, and not binary by the NUL sniff specs 11 specifies. A file that
// fails either is skipped silently — it is not a refusal the model asked for,
// and a line per skipped artefact would be the noise this filter removes.
func readableText(file *os.File) (*bufio.Reader, bool) {
	info, err := file.Stat()
	if err != nil || info.Size() > maxSearchFileBytes {
		return nil, false
	}
	reader := bufio.NewReaderSize(file, binarySniffBytes)
	head, err := reader.Peek(binarySniffBytes)
	if err != nil && len(head) == 0 && info.Size() > 0 {
		return nil, false
	}
	if bytes.IndexByte(head, 0) >= 0 {
		return nil, false
	}
	return reader, true
}

// emitBefore renders the context lines leading up to a match, skipping the
// ones a previous match already rendered — which is what merges two
// overlapping windows into one passage instead of repeating their shared
// lines.
func (m *matchSet) emitBefore(wirePath string, before []textLine) {
	for _, line := range before {
		if wirePath == m.lastPath && line.number <= m.lastNumber {
			continue
		}
		m.emit(wirePath, line, false)
	}
}

func (m *matchSet) emit(wirePath string, line textLine, isMatch bool) {
	if m.separates(wirePath, line) {
		m.out = append(m.out, groupSeparator)
	}
	m.out = append(m.out, renderLine(wirePath, line, isMatch))
	m.lastPath, m.lastNumber = wirePath, line.number
}

// separates answers whether a group boundary falls before this line. Only a
// result carrying context lines has groups at all: with context 0 every line
// is its own match and a separator between each of them would be half the
// result.
func (m *matchSet) separates(wirePath string, line textLine) bool {
	if m.query.context == 0 || len(m.out) == 0 {
		return false
	}
	return wirePath != m.lastPath || line.number != m.lastNumber+1
}

// remember keeps the last `context` lines, which is all the preceding context
// any match can need.
func remember(before []textLine, line textLine, context int) []textLine {
	if context == 0 {
		return before
	}
	before = append(before, line)
	if len(before) > context {
		before = before[1:]
	}
	return before
}

// renderLine is the result's whole vocabulary: path:line:text for a match,
// path-line-text for a line around one. Every line carries its own path, so a
// result read from any point says where it came from.
func renderLine(wirePath string, line textLine, isMatch bool) string {
	separator := "-"
	if isMatch {
		separator = ":"
	}
	return wirePath + separator + strconv.Itoa(line.number) + separator + clip(line.text)
}

// clip bounds one line so that a single minified line cannot become the whole
// result.
func clip(text string) string {
	if len(text) <= maxSearchLineRunes {
		return text
	}
	runes := []rune(text)
	if len(runes) <= maxSearchLineRunes {
		return text
	}
	return string(runes[:maxSearchLineRunes]) + lineClippedMarker
}

func (m *matchSet) render() string {
	var out strings.Builder
	for _, line := range m.out {
		out.WriteString(line)
		out.WriteByte('\n')
	}
	if m.total > m.shown {
		fmt.Fprintf(&out, "[truncated: showing %d of %d matches]\n", m.shown, m.total)
	}
	return out.String()
}

// intArgument reads one integer parameter out of a decoded tool call. JSON has
// one number type, so a whole number arrives as a float64; a value outside the
// stated range is clamped rather than refused, because a model asking for more
// than the cap wants as much as it can have.
func intArgument(arguments map[string]any, name string, fallback, low, high int) (int, error) {
	raw, present := arguments[name]
	if !present || raw == nil {
		return fallback, nil
	}

	value := 0
	switch number := raw.(type) {
	case float64:
		value = int(number)
	case int:
		value = number
	default:
		return 0, fmt.Errorf("the %s parameter must be a number", name)
	}

	return min(max(value, low), high), nil
}
