package repo

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strings"
)

// gitConfigEnvironment names the two variables that decide whether a
// machine's own git configuration reaches a subcommand. Pointed at the
// platform's null device they resolve to an empty configuration, so an alias,
// a pager, an `include`, or a global `core.excludesFile` cannot change what
// `ls-files` reports (specs 11 — Tool implementation strategy).
//
// They cover two of git's three configuration scopes. The third is the
// reviewed repository's own .git/config, and that one is the attacker's —
// see gitSafetyOptions.
var gitConfigEnvironment = []string{
	"GIT_CONFIG_GLOBAL=" + os.DevNull,
	"GIT_CONFIG_SYSTEM=" + os.DevNull,
}

// gitSafetyOptions are the git-level options every invocation carries,
// whatever the subcommand, because the repository this binary is pointed at
// wrote its own .git/config and several read-only subcommands run a program
// named there as a normal part of their work (issue #58). Argument validation
// cannot see any of this: the payload arrives through configuration.
//
// Command-line `-c` beats repository configuration, and this binary owns the
// argv it builds, so blanking a key here is what makes the repository's value
// unreachable. Each entry is a verified vector, not a precaution:
//
//   - core.fsmonitor runs on status and on both ls-files calls enumeration
//     makes, which is why this lives here rather than in git_read.
//   - diff.external runs on diff.
//   - gpg.program runs when a commit carries a signature to verify.
//   - core.pager is unreachable while output is captured rather than a
//     terminal, and is blanked so that stops being the only thing stopping it.
//
// --no-optional-locks is not a config override and closes a vector no
// override can: `git status` refreshes and rewrites the index, which runs
// .git/hooks/post-index-change — a file inside the repository, so no `-c`
// reaches it and blanking core.hooksPath only falls back to that same
// directory. Not writing the index is what stops it.
//
// Deliberately absent: core.sshCommand, credential.helper and
// url.<base>.insteadOf name programs but fire on network operations, and none
// of the four subcommands git_read allows performs one. They belong on this
// list the day a subcommand that talks to a remote is allowed.
var gitSafetyOptions = []string{
	"--no-optional-locks",
	"-c", "core.fsmonitor=",
	"-c", "core.pager=",
	"-c", "diff.external=",
	"-c", "gpg.program=",
}

// diffHelperFlags turn off the two mechanisms by which a diff runs a program
// the repository's .gitattributes selected: a textconv, and an external diff
// driver. Both are reached through a driver name the repository chooses, so
// neither can be pre-empted by a `-c` this binary writes in advance — the
// flags cover the whole class instead, which is what makes them the right
// instrument rather than belt-and-braces.
//
// --no-ext-diff rather than `-c diff.external=` alone: the per-driver form,
// diff.<driver>.command, is a second external-diff vector that the single key
// does not cover.
var diffHelperFlags = []string{"--no-ext-diff", "--no-textconv"}

// diffShapedSubcommands are the subcommands that produce a diff and therefore
// accept diffHelperFlags. status and ls-files reject both flags outright,
// which is the whole reason the flags are placed per subcommand instead of
// being appended to every invocation alongside gitSafetyOptions.
var diffShapedSubcommands = []string{"log", "diff", "show"}

// executableDriverKey matches a configuration key whose value git executes and
// whose driver name belongs to the repository rather than to this binary.
//
// filter.<driver>.clean is the vector that forces this to exist. A clean
// filter converts a modified working-tree file before it is compared, so `git
// diff` against the working tree runs it, and no diff flag disables filters
// the way --no-textconv disables textconv. The driver name being the
// attacker's, the only complete answer is to read back the names the
// repository actually configured and blank each one.
var executableDriverKey = regexp.MustCompile(`^(?i:filter\..+\.(?:clean|smudge|process)|diff\..+\.(?:textconv|command))$`)

// Output is what one git invocation produced: its standard output, and its
// standard error for the one caller that has to explain a failure to the model
// rather than to a human (internal/tools' git_read — a non-zero git exit is a
// tool error carrying git's own sentence, since the reviewer is the only one
// who can act on it).
type Output struct {
	Stdout []byte
	Stderr []byte
}

// Git runs one git subcommand against repoPath and returns both its streams.
// It is exported for git_read, which is the only other place in this binary
// allowed to run git, so that there is one exec in the codebase rather than
// two: the argv discipline, the `-C` form and the environment neutralisation
// below are properties of *every* git this process runs, not of enumeration.
//
// The error is returned unwrapped, because its two shapes are the caller's to
// tell apart: exec.ErrNotFound (git is not installed) and *exec.ExitError (git
// ran and refused).
func Git(ctx context.Context, repoPath string, args ...string) (Output, error) {
	return runGitStreams(ctx, repoPath, args...)
}

// runGit runs one git subcommand against repoPath and returns its stdout,
// discarding git's own stderr — which is what enumeration wants: git's
// sentences are capitalised, full-stopped and platform-spelled, and this run
// states its own diagnostics (CONVENTIONS § Error handling).
func runGit(ctx context.Context, repoPath string, args ...string) ([]byte, error) {
	output, err := runGitStreams(ctx, repoPath, args...)
	if err != nil {
		return nil, fmt.Errorf("running git %s: %w", strings.Join(args, " "), err)
	}
	return output.Stdout, nil
}

// runGitStreams is how every caller reaches git, and every constraint on that
// is here rather than at the call sites: the environment carries no GIT_*
// variable from this process, and the arguments are rewritten so that nothing
// the reviewed repository configured can run a program of its choosing.
func runGitStreams(ctx context.Context, repoPath string, args ...string) (Output, error) {
	return runGitRaw(ctx, repoPath, hardened(ctx, repoPath, args)...)
}

// hardened turns the arguments a caller asked for into the arguments git is
// actually handed: the safety options first, since git-level options have to
// precede the subcommand, then the overrides derived from this repository,
// then the subcommand with its own flags.
//
// The diff flags go immediately after the subcommand rather than at the end
// because git applies the last occurrence of a flag, and the caller's
// arguments follow. git_read refuses --ext-diff and --textconv for exactly
// that reason: they are the arguments that would undo this.
func hardened(ctx context.Context, repoPath string, args []string) []string {
	arguments := make([]string, 0, len(gitSafetyOptions)+len(diffHelperFlags)+len(args)+8)
	arguments = append(arguments, gitSafetyOptions...)
	arguments = append(arguments, driverOverrides(ctx, repoPath)...)
	if len(args) == 0 {
		return arguments
	}

	arguments = append(arguments, args[0])
	if slices.Contains(diffShapedSubcommands, args[0]) {
		arguments = append(arguments, diffHelperFlags...)
	}
	return append(arguments, args[1:]...)
}

// driverOverrides asks the repository which executable drivers it configures
// and returns a `-c` blanking each one. It runs on every invocation rather
// than once per run: the answer is a property of a file the reviewed
// repository owns and can rewrite mid-review, and a cached one would be a
// window during which a driver added after the first read went unneutralised.
//
// `--list` without `--local` is the deliberate spelling. `--local` reads
// .git/config alone: it does not follow the `include.path` directives inside
// it, and it does not see .git/config.worktree — both of which a repository
// can use to configure a driver that a `--local` listing never names. The
// plain listing sees all three, and it is confined to the repository because
// the environment has already emptied the global and system scopes.
//
// Reading configuration executes nothing, which is what makes this safe to do
// before the hardening is known. A failure means there is no local
// configuration to be read — the path is not a repository, or git is absent —
// and the invocation that follows fails on its own account and is reported by
// its caller; gitSafetyOptions still applies either way.
func driverOverrides(ctx context.Context, repoPath string) []string {
	listing, err := runGitRaw(ctx, repoPath, "config", "--list", "--name-only")
	if err != nil {
		return nil
	}

	var overrides []string
	for _, key := range strings.Split(string(listing.Stdout), "\n") {
		if key = strings.TrimSpace(key); executableDriverKey.MatchString(key) {
			overrides = append(overrides, "-c", key+"=")
		}
	}
	return overrides
}

// runGitRaw is the codebase's only process execution: git is named as a bare
// program resolved from PATH, its arguments are an argv slice that no shell
// ever parses, and its environment carries no GIT_* variable from this
// process. `-C repoPath` rather than cmd.Dir because it is git's own way of
// saying which repository it is being asked about, and it survives a repoPath
// the process's working directory could not.
//
// It is *raw* in one sense only — it adds no argument of its own — and the one
// caller entitled to that is driverOverrides, which has to read this
// repository's configuration in order to work out what the hardening should
// say. Reading configuration runs no program; everything else goes through
// runGitStreams.
//
// The subprocess reads the filesystem on its own authority, outside the
// *os.Root — which is exactly why nothing it reports is trusted: every path it
// names is re-resolved through the Scope before it is used (see enumerate.go),
// and every path a caller *sends* is resolved through the Scope before the
// command is built (see internal/tools/git_read.go).
func runGitRaw(ctx context.Context, repoPath string, args ...string) (Output, error) {
	// The single exec in this binary. exec.Command* is forbidden everywhere
	// else because a subprocess reads on its own authority, outside the root
	// handle (CONVENTIONS § Code style & formatting; ripgrep was rejected on
	// this exact point). git is the one program the confinement can survive,
	// because what it returns is names, and names are re-resolved.
	//
	// gosec's G204 fires here on principle — the arguments are not constants —
	// and it is answered rather than suppressed blindly: the program is the
	// literal "git", the arguments are an argv slice no shell ever sees, and
	// on the platform where argument splitting is a CommandLineToArgvW problem
	// that is the whole defence (specs 11). No caller passes model-supplied
	// text through here unexamined: enumeration's arguments are literals, and
	// git_read allowlists its subcommand and resolves every path it was given
	// through the Scope before the command is built.
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", repoPath}, args...)...) //nolint:forbidigo,gosec // the one allowed exec: argv-only, no shell, see above
	cmd.Env = gitEnvironment()

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	// git's own stderr is captured rather than forwarded: it is the OS's and
	// git's sentences, capitalised and platform-spelled, and this run states
	// its own diagnostics (CONVENTIONS § Error handling). The one caller that
	// surfaces it hands it to the model, which is the only reader that can act
	// on "fatal: invalid object name".
	cmd.Stderr = &stderr

	err := cmd.Run()
	return Output{Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}, err
}

// gitEnvironment is the process environment with every GIT_* variable dropped
// and the two configuration variables set explicitly.
//
// Dropping the whole prefix rather than the two names is the point: GIT_DIR,
// GIT_WORK_TREE, GIT_INDEX_FILE and GIT_CEILING_DIRECTORIES each redirect a
// subcommand at a different repository than the one this run was pointed at,
// and GIT_CONFIG_COUNT injects configuration without naming a file. The
// remaining environment is left alone, because git needs PATH — and, on
// Windows, SystemRoot — to run at all.
//
// The comparison folds case: Windows environment variable names are
// case-insensitive, so `git_config_global` set in the parent would otherwise
// survive a prefix test that only knows about `GIT_`.
func gitEnvironment() []string {
	inherited := os.Environ()
	environment := make([]string, 0, len(inherited)+len(gitConfigEnvironment))
	for _, entry := range inherited {
		if strings.HasPrefix(strings.ToUpper(entry), "GIT_") {
			continue
		}
		environment = append(environment, entry)
	}
	return append(environment, gitConfigEnvironment...)
}

// splitNUL splits the output of a `-z` git command into its records. `-z` is
// what makes this safe: git quotes a filename containing a non-ASCII byte, a
// quote or a newline under its default core.quotepath, and a NUL-separated
// stream is emitted raw instead — so a filename arrives spelled exactly as the
// filesystem holds it.
func splitNUL(output []byte) []string {
	records := make([]string, 0, bytes.Count(output, []byte{0}))
	for _, record := range bytes.Split(output, []byte{0}) {
		if len(record) == 0 {
			continue
		}
		records = append(records, string(record))
	}
	return records
}
