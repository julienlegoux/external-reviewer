package tools

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/julienlegoux/kern-link/ai"
)

// MaxListPaths caps one listing. It is a stated constant, not a tuning knob:
// the binding limit is the model's context window, and `search` caps at the
// same 200 because both tools answer "what is there" (specs 09 — Read-only
// tool contract).
const MaxListPaths = 200

// defaultListPattern lists the whole granted surface, since `**` crosses
// directories and `*` matches any single segment.
const defaultListPattern = "**/*"

// Enumerator is the repository enumeration `list` reads from — the same one
// `search` uses, so what the reviewer can glob and what it can grep never
// disagree (specs 11 — Tool implementation strategy).
type Enumerator interface {
	Files(ctx context.Context) ([]string, error)
}

// listDescription is the model's only instruction on how to enumerate this
// repository: the caller owns the system prompt, so this stands alone (specs
// 09, specs 08).
const listDescription = `List the files of the repository under review, one path per line.

Paths are relative to the repository root and always use / as the separator, whichever operating system this runs on; pass them back to the other tools exactly as they appear here.

The pattern is a glob matched against the whole path: * matches within one path segment, ** matches across segments, so "**/*.go" finds Go files at any depth and "docs/*" only the immediate children of docs. The default, "**/*", lists everything visible.

What is visible is narrower than the repository: files ignored by .gitignore are never listed, nor is anything outside the subtrees this run was granted, nor files whose names mark them as credentials. A path you cannot see here cannot be read by any other tool either.

At most 200 paths are returned. When more match, the last line reads "[truncated: showing 200 of N paths]" with the real total — narrow the pattern to see the rest.`

// listParameters is the wire schema, snake_case like the tool name, because
// it is a contract with the model rather than Go API (CONVENTIONS § Naming).
var listParameters = ai.JSONSchema(`{
  "type": "object",
  "properties": {
    "pattern": {
      "type": "string",
      "description": "Glob matched against the whole repo-relative path; * stays within a path segment and ** crosses them. Defaults to \"**/*\"."
    }
  },
  "additionalProperties": false
}`)

// List builds the tool over an enumeration. Everything that decides what a
// listing contains — gitignore, the allow-list, the sensitive-file floor —
// lives in that enumeration, so this tool matches and formats and nothing
// else.
func List(repository Enumerator) Tool {
	return Tool{
		Declaration: ai.Tool{
			Name:        "list",
			Description: listDescription,
			Parameters:  listParameters,
		},
		Handler: func(ctx context.Context, arguments map[string]any) Result {
			return listFiles(ctx, repository, arguments)
		},
	}
}

func listFiles(ctx context.Context, repository Enumerator, arguments map[string]any) Result {
	pattern, err := stringArgument(arguments, "pattern", defaultListPattern)
	if err != nil {
		return errorResult(err.Error())
	}
	if err := validateGlob(pattern); err != nil {
		return errorResult(fmt.Sprintf("the pattern %q is not a valid glob: %v", pattern, err))
	}

	files, err := repository.Files(ctx)
	if err != nil {
		return errorResult(fmt.Sprintf("the repository could not be enumerated: %v", err))
	}

	matched := make([]string, 0, len(files))
	for _, file := range files {
		if ok, err := matchGlob(pattern, file); err == nil && ok {
			matched = append(matched, file)
		}
	}
	if len(matched) == 0 {
		return Result{Text: fmt.Sprintf("no paths match %q\n", pattern)}
	}
	return Result{Text: renderPaths(matched)}
}

// renderPaths writes the paths one per line, capped, and announces the cap
// when it bit. Truncation is always announced, with both real numbers: a
// result that silently showed sixty percent of the material would read exactly
// as confident as one that showed all of it (specs 09).
func renderPaths(matched []string) string {
	shown := matched
	if len(shown) > MaxListPaths {
		shown = shown[:MaxListPaths]
	}

	var out strings.Builder
	for _, file := range shown {
		out.WriteString(file)
		out.WriteByte('\n')
	}
	if len(shown) < len(matched) {
		fmt.Fprintf(&out, "[truncated: showing %d of %d paths]\n", len(shown), len(matched))
	}
	return out.String()
}

// stringArgument reads one string parameter out of a decoded tool call. A
// value of the wrong type is the model's mistake to learn from, so it comes
// back as a refusal naming the parameter rather than as a failure.
func stringArgument(arguments map[string]any, name, fallback string) (string, error) {
	raw, present := arguments[name]
	if !present || raw == nil {
		return fallback, nil
	}
	value, ok := raw.(string)
	if !ok {
		return "", fmt.Errorf("the %s parameter must be a string", name)
	}
	if value == "" {
		return fallback, nil
	}
	return value, nil
}

// validateGlob answers whether a pattern is usable before any file is matched
// against it, so an unmatchable pattern is refused even in an empty
// repository. path.Match only reports a malformed pattern once it reaches the
// malformed part, hence the non-empty probe name.
//
// An absolute or backslash-spelled pattern is refused for the same reason a
// wire path is: a pattern is matched against repo-relative, /-separated paths,
// and one that cannot match any of them is a mistake worth naming rather than
// an empty listing to puzzle over.
func validateGlob(pattern string) error {
	switch {
	case strings.HasPrefix(pattern, "/"):
		return fmt.Errorf("a pattern is repo-relative, and %q is absolute", pattern)
	case strings.ContainsRune(pattern, '\\'):
		return fmt.Errorf("a pattern uses / as its separator, and %q uses a native one", pattern)
	}
	for _, segment := range strings.Split(pattern, "/") {
		if segment == "**" {
			continue
		}
		if _, err := path.Match(segment, "probe"); err != nil {
			return err
		}
	}
	return nil
}

// matchGlob matches a wire path against a glob, segment by segment, with `**`
// standing for any number of segments — the one piece of glob syntax
// path.Match does not have, and the one a reviewer reaches for first
// ("**/*.go"). Every other segment is path.Match's own syntax, so `*`, `?` and
// character classes mean what they mean everywhere else, and `*` never crosses
// a separator because it is never handed one.
func matchGlob(pattern, wirePath string) (bool, error) {
	return matchSegments(strings.Split(pattern, "/"), strings.Split(wirePath, "/"))
}

func matchSegments(pattern, segments []string) (bool, error) {
	for len(pattern) > 0 {
		if pattern[0] == "**" {
			// Zero or more segments: try every split, shortest first.
			for i := 0; i <= len(segments); i++ {
				matched, err := matchSegments(pattern[1:], segments[i:])
				if err != nil || matched {
					return matched, err
				}
			}
			return false, nil
		}
		if len(segments) == 0 {
			return false, nil
		}
		matched, err := path.Match(pattern[0], segments[0])
		if err != nil || !matched {
			return false, err
		}
		pattern, segments = pattern[1:], segments[1:]
	}
	return len(segments) == 0, nil
}
