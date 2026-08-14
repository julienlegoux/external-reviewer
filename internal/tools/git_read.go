package tools

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/julienlegoux/kern-link/ai"

	"github.com/julienlegoux/external-reviewer/internal/confine"
	"github.com/julienlegoux/external-reviewer/internal/repo"
)

// MaxGitReadLines caps one git_read answer. History reads are the least
// bounded thing a reviewer can ask for — `git log -p` over a year of commits
// is megabytes — and the binding limit is the model's context window, as it is
// for every other tool (specs 09 — Read-only tool contract).
const MaxGitReadLines = 2000

// gitReadCommands is the whole of what this tool will run, and the reason it
// is a fixed list rather than a filter: the four read-shaped subcommands were
// decided at specs 09, and widening the list is a SPECS change rather than a
// PR decision. A subcommand outside it never reaches git.
var gitReadCommands = []string{"log", "diff", "show", "status"}

// pathspecSeparator is the argument this tool owns. Everything after the first
// `--` is a pathspec, so the granted subtrees are only a confinement while the
// model cannot spell one itself: a second `--` would append pathspecs of its
// own choosing after ours, and git reads them all.
const pathspecSeparator = "--"

// outputOption is the one git option that writes a file. `git diff
// --output=<file>` is accepted by three of the four subcommands and would put
// this binary's own output on disk — the write half of the filesystem API is
// absent from this codebase precisely so that "it writes nothing" is a
// property of the code (CONVENTIONS § Code style & formatting), and a
// subprocess doing it on our behalf is the same thing at one remove.
const outputOption = "--output"

// helperOptions switch back on the two mechanisms internal/repo turns off
// because the reviewed repository's own .git/config names the programs they
// run (issue #58). They are refused rather than merely overridden: the
// hardening is placed immediately after the subcommand, these arguments follow
// it, and git applies the last occurrence of a flag — so allowing one would be
// allowing the model to undo the confinement on the repository's behalf.
var helperOptions = []string{"--ext-diff", "--textconv"}

// gitReadDescription is the model's only instruction on how to read history
// here: the caller owns the system prompt, so this stands alone (specs 09).
const gitReadDescription = `Read this repository's git history.

The command is one of log, diff, show or status — nothing else, so no commit, push, blame or ls-tree. The args parameter is an array of strings passed to git as arguments, never a single string: there is no shell here, so nothing is ever split on spaces. ["--oneline", "-n", "20"] is one call's arguments; "--oneline -n 20" is not.

History is confined exactly as the working tree is. log, diff, show and status are all scoped to the subtrees this run was granted, appended as pathspecs, so a commit that only touched an ungranted subtree is not reported. An argument of the form <rev>:<path> — the form that reads a file's contents at a revision, and the index forms :<path> and :<stage>:<path> — has its path checked against those same subtrees and against the sensitive-file floor before git is run at all; a refused path never reaches git. Any revision is allowed: HEAD, a branch, a tag, a SHA, a range.

The paths parameter narrows a read to part of what was granted, and is how a diff is scoped to a subtree: an array of repo-relative, /-separated paths, checked against the granted subtrees and the sensitive-file floor exactly as a <rev>:<path> is, then used as the pathspecs in place of the whole grant. {"command": "diff", "args": ["--unified=0", "HEAD~3..HEAD"], "paths": ["internal/confine"]} diffs that subtree alone rather than everything. Leave paths out and the whole granted surface is read, which is what hits the line cap on a large change. It cannot be combined with a <rev>:<path> object, since git takes pathspecs or an object and not both.

Four arguments are refused: a literal -- (this tool appends the pathspecs itself, and paths is how to choose them), --output (this run writes no files), and --ext-diff or --textconv (they let this repository's own configuration name a program for git to run while reading it, so diffs come back without external diff drivers or textconv filters applied). When one argument names a <rev>:<path> object, all of them must, since git cannot take pathspecs alongside a blob.

At most 2000 lines are returned. When there are more, the last line reads "[truncated: showing 2000 of N lines]" with the real total — narrow the request with -n, --stat, or a pathspec-free argument like --oneline.`

// gitReadParameters is the wire schema, snake_case like the tool name, because
// it is a contract with the model rather than Go API (CONVENTIONS § Naming).
// args is declared as an array of strings, which is the schema half of the
// no-shell rule the handler enforces again on the value that actually arrives.
var gitReadParameters = ai.JSONSchema(`{
  "type": "object",
  "properties": {
    "command": {
      "type": "string",
      "enum": ["log", "diff", "show", "status"],
      "description": "The git subcommand to run. Only these four are available."
    },
    "args": {
      "type": "array",
      "items": {"type": "string"},
      "description": "Arguments passed to git as an argv array, one element per argument. Never a single string: there is no shell to split one."
    },
    "paths": {
      "type": "array",
      "items": {"type": "string"},
      "description": "Repo-relative, /-separated paths to narrow this read to, inside the subtrees this run was granted. This is how a diff is scoped to a subtree. Omit it to read the whole granted surface."
    }
  },
  "required": ["command"],
  "additionalProperties": false
}`)

// GitRead builds the tool that reads history, over the run's confinement and
// the repository path git is pointed at.
//
// It is the only tool that reaches material *os.Root cannot guard: a
// subprocess reads on its own authority, and `git show HEAD:.env` prints a
// credential out of history without opening a single file inside the root. The
// confinement is therefore made one layer earlier, in argument space — every
// path the model names is resolved through Scope.Resolve before the command is
// built, and the granted subtrees are appended as pathspecs the model cannot
// reach past. Scope.Resolve performs no I/O for exactly this reason: a path
// that exists only in history has no file to stat (specs 10, and Epic 2's
// drift record 01).
func GitRead(scope *confine.Scope, repoPath string) Tool {
	return Tool{
		Declaration: ai.Tool{
			Name:        "git_read",
			Description: gitReadDescription,
			Parameters:  gitReadParameters,
		},
		Handler: func(ctx context.Context, arguments map[string]any) Result {
			return gitRead(ctx, scope, repoPath, arguments)
		},
	}
}

func gitRead(ctx context.Context, scope *confine.Scope, repoPath string, arguments map[string]any) Result {
	command, err := gitReadCommand(arguments)
	if err != nil {
		return errorResult(err.Error())
	}
	args, err := stringsArgument(arguments, "args")
	if err != nil {
		return errorResult(err.Error())
	}
	paths, err := stringsArgument(arguments, "paths")
	if err != nil {
		return errorResult(err.Error())
	}
	argv, err := gitReadArgv(scope, command, args, paths)
	if err != nil {
		return errorResult(err.Error())
	}

	output, err := repo.Git(ctx, repoPath, argv...)
	if err != nil {
		return errorResult(gitFailure(command, output.Stderr, err))
	}
	return Result{Text: capLines(string(output.Stdout), MaxGitReadLines)}
}

// gitReadCommand reads the subcommand and answers the allowlist question
// before anything else, so a refused subcommand never reaches git and the
// refusal names the four that would have worked — a model that asked for
// `blame` can correct itself from a list and cannot from "not allowed".
func gitReadCommand(arguments map[string]any) (string, error) {
	command, err := stringArgument(arguments, "command", "")
	if err != nil {
		return "", err
	}
	if !contains(gitReadCommands, command) {
		return "", fmt.Errorf("git_read cannot run %q; the commands it allows are: %s",
			command, strings.Join(gitReadCommands, ", "))
	}
	return command, nil
}

// gitReadArgv turns the model's arguments into the argv git is handed, or
// refuses. Nothing here runs git: every rule is decided on the strings, which
// is what makes "a refused path never reaches git" true rather than likely.
func gitReadArgv(scope *confine.Scope, command string, args, paths []string) ([]string, error) {
	pathspecs, err := gitReadPathspecs(scope, paths)
	if err != nil {
		return nil, err
	}

	objects := 0
	commits := 0
	for _, arg := range args {
		if err := allowedArgument(arg); err != nil {
			return nil, err
		}
		if strings.HasPrefix(arg, "-") {
			continue
		}
		wirePath, isObject := objectPath(arg)
		if !isObject {
			commits++
			continue
		}
		objects++
		if _, err := scope.Resolve(wirePath); err != nil {
			return nil, objectRefusal(arg, wirePath, err, scope.Allowed())
		}
	}

	argv := append([]string{command}, args...)
	if objects == 0 {
		return append(append(argv, pathspecSeparator), pathspecs...), nil
	}
	if len(paths) > 0 {
		return nil, errors.New("git_read cannot take paths alongside a <rev>:<path> object, " +
			"because git takes pathspecs or an object and not both; ask for the object on its own, " +
			"which already names the one path it reads")
	}
	if commits > 0 {
		// git takes pathspecs *or* a blob pair, never both — `git diff <blob>
		// <blob> -- docs` is a usage error — so a call naming an object gets no
		// pathspecs, and its confinement is the path validation above. Mixing
		// the two would therefore be the one shape that runs a commit unscoped.
		return nil, fmt.Errorf("git_read cannot mix a <rev>:<path> object with a plain revision in one call, "+
			"because git takes pathspecs or an object and not both, and %s is scoped by the granted subtrees "+
			"appended as pathspecs (%s); ask for the object on its own",
			command, strings.Join(scope.Allowed(), ", "))
	}
	return argv, nil
}

// gitReadPathspecs decides what follows the single `--` this tool owns: the
// granted subtrees when the model named no paths, and the paths it named when
// it did. That is the whole of the parameter — a path-scoped diff was
// unreachable not because scoping was refused but because there was no way to
// spell one, the literal `--` being reserved and everything before it read by
// git as a revision.
//
// Every entry goes through the same gate as a <rev>:<path> object, before a
// command exists, so what comes back is always a subset of the grant: a
// narrowing, never a widening. That is also what makes handing the model's own
// strings to git safe where pathspec magic is concerned — `:(exclude)…`, `:/…`
// and a wildcard are all just strings to Scope.Resolve, and any of them that
// does not name something inside a granted subtree is refused. The one grant
// they survive is `.`, which has nothing narrower to reach past; a pathspec
// cannot leave the repository at all.
func gitReadPathspecs(scope *confine.Scope, paths []string) ([]string, error) {
	if len(paths) == 0 {
		return scope.Allowed(), nil
	}

	pathspecs := make([]string, 0, len(paths))
	for _, wirePath := range paths {
		cleaned, err := scope.Resolve(wirePath)
		if err != nil {
			return nil, pathsRefusal(wirePath, err, scope.Allowed())
		}
		pathspecs = append(pathspecs, cleaned)
	}
	return pathspecs, nil
}

// allowedArgument refuses the two arguments that reach past the confinement
// rather than through it.
func allowedArgument(arg string) error {
	if arg == pathspecSeparator {
		return fmt.Errorf("the argument %q is refused: git_read appends the granted subtrees as pathspecs itself, "+
			"and a second -- would let the arguments after it widen that scope", arg)
	}
	if name, _, _ := strings.Cut(arg, "="); name == outputOption {
		return fmt.Errorf("the argument %q is refused: it writes git's output to a file, and this run writes none — "+
			"the result comes back here, inside the granted pathspecs", arg)
	}
	if contains(helperOptions, arg) {
		return fmt.Errorf("the argument %q is refused: it lets this repository's own configuration name a program "+
			"for git to run while reading it, and a reviewed repository's configuration is untrusted input; "+
			"the diff comes back without the helper applied", arg)
	}
	return nil
}

// objectPath reports whether an argument names an object by path — the form
// that reads a file's contents rather than a commit — and returns the wire
// path it names.
//
// Three spellings do it, and all three are the same hole: `<rev>:<path>` reads
// from history, `:<path>` and `:<stage>:<path>` read from the index. The split
// is at the *first* colon, which is git's own rule, so `refs/heads/x:notes.md`
// names notes.md rather than a branch called `heads/x:notes.md`.
//
// An empty path — `HEAD:`, the root tree — is answered as ".", so the
// allow-list decides it: listing the repository root's top-level names is
// exactly what a run granted a single subtree must not do, and exactly what
// `--allow .` may.
func objectPath(arg string) (string, bool) {
	rest, isIndexForm := strings.CutPrefix(arg, ":")
	if isIndexForm {
		// `:<stage>:<path>`, stage being a single digit 0-3.
		if len(rest) > 1 && rest[0] >= '0' && rest[0] <= '3' && rest[1] == ':' {
			rest = rest[2:]
		}
		return objectWirePath(rest), true
	}
	_, path, found := strings.Cut(arg, ":")
	if !found {
		return "", false
	}
	return objectWirePath(path), true
}

// objectWirePath maps git's spelling of a path inside an object to the wire
// path the resolver answers for: the root tree is ".", and `HEAD:./notes.md`
// is the same file as `HEAD:notes.md` because git resolves it against the
// prefix, which `-C repoPath` fixes at the repository root.
func objectWirePath(path string) string {
	if path == "" {
		return "."
	}
	return path
}

// objectRefusal states which of the three rules refused, in words that differ
// per rule: "widen --allow" and "that file is a credential" are different
// things for a reviewer to learn, and a message that blurred them would leave
// the model retrying the one it cannot win (specs 10).
func objectRefusal(arg, wirePath string, err error, allowed []string) error {
	switch {
	case errors.Is(err, confine.ErrDeniedFilename):
		return fmt.Errorf("%s is refused: %q is denied by the sensitive-file floor, "+
			"which covers credentials and .git/ inside every granted subtree and which no --allow widens",
			arg, wirePath)
	case errors.Is(err, confine.ErrOutsideAllowList):
		return fmt.Errorf("%s is refused: %q is outside the subtrees this run was granted; "+
			"the --allow subtrees are: %s", arg, wirePath, strings.Join(allowed, ", "))
	default:
		return fmt.Errorf("%s is refused: %q is not a path inside the repository, "+
			"which is repo-relative and /-separated whichever operating system this runs on", arg, wirePath)
	}
}

// pathsRefusal states which of the three rules refused a paths entry, in
// objectRefusal's own words. The rules are the same three rules, so a reviewer
// that learned "widen --allow" from an object argument reads it identically
// here rather than having to learn a second vocabulary for the same boundary.
func pathsRefusal(wirePath string, err error, allowed []string) error {
	return objectRefusal(fmt.Sprintf("the paths entry %q", wirePath), wirePath, err, allowed)
}

// gitFailure renders a git that would not run, or ran and refused, as text the
// model can act on. git's own stderr is quoted here — the one place in this
// binary where it is — because "fatal: invalid object name 'deadbeef'" is the
// answer to the model's question, and the model is the only reader who can ask
// a better one.
func gitFailure(command string, stderr []byte, err error) string {
	if errors.Is(err, exec.ErrNotFound) {
		return "git is not on PATH, so this repository's history cannot be read; " +
			"the other tools still work on the working tree"
	}
	if message := strings.TrimSpace(string(stderr)); message != "" {
		return fmt.Sprintf("git %s failed: %s", command, message)
	}
	return fmt.Sprintf("git %s failed: %v", command, err)
}

// capLines truncates an answer to at most max lines and says so with both real
// numbers. Truncation is always announced: a result that silently showed a
// third of a diff would read exactly as confident as one that showed all of it
// (specs 09).
func capLines(output string, max int) string {
	if output == "" {
		return output
	}

	lines := strings.SplitAfter(output, "\n")
	if last := len(lines) - 1; lines[last] == "" {
		lines = lines[:last]
	}
	if len(lines) <= max {
		return output
	}

	var out strings.Builder
	for _, line := range lines[:max] {
		out.WriteString(line)
	}
	if !strings.HasSuffix(out.String(), "\n") {
		out.WriteByte('\n')
	}
	fmt.Fprintf(&out, "[truncated: showing %d of %d lines]\n", max, len(lines))
	return out.String()
}

// stringsArgument reads an array-of-strings parameter out of a decoded tool
// call. The two mistakes it names are the two the model actually makes: the
// whole parameter sent as one string — which is what a shell would have split,
// and there is no shell — and an element that is a number.
func stringsArgument(arguments map[string]any, name string) ([]string, error) {
	raw, present := arguments[name]
	if !present || raw == nil {
		return nil, nil
	}
	list, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("the %s parameter must be an array of strings, one element per argument, "+
			"not a single string: there is no shell here to split one on spaces", name)
	}

	values := make([]string, 0, len(list))
	for index, element := range list {
		value, ok := element.(string)
		if !ok {
			return nil, fmt.Errorf("the %s parameter must be an array of strings, and %s[%d] is not one", name, name, index)
		}
		values = append(values, value)
	}
	return values, nil
}

// contains is slices.Contains under a name that reads in the one place it is
// used, and without importing a package for a four-element list.
func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
