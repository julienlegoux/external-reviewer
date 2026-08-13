package cli_test

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/julienlegoux/kern-link/ai"
	"github.com/julienlegoux/kern-link/ai/providers/faux"

	"github.com/julienlegoux/external-reviewer/internal/cli"
	"github.com/julienlegoux/external-reviewer/internal/config"
	"github.com/julienlegoux/external-reviewer/internal/fauxtest"
	"github.com/julienlegoux/external-reviewer/internal/reviewer"
)

// defaultTierFixture is the tier assignment every test in this package runs
// against unless it writes its own. It assigns `standard` to the same
// provider and model preflight_test.go's registry serves, so every test that
// predates tier resolution — and simply expects `review` to reach a model —
// keeps working without naming a tier.
const defaultTierFixture = `
[tiers.standard]
provider = "` + fixtureProvider + `"
model    = "` + fixtureModel + `"
`

// TestMain pins the two environment inputs tier resolution reads, for the
// whole package, before a single test runs.
//
// This is not convenience: without it, every test that performs a review
// would consult the *developer's own* config file through
// os.UserConfigDir(), and a machine with a real [tiers.standard] table would
// run this suite against whatever model it names — or fail on a machine
// that has none. The same goes for a stray EXTERNAL_REVIEWER_TIER_* in the
// author's shell. Individual tests still override the file with t.Setenv,
// which restores the value set here when they finish.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "external-reviewer-cli")
	if err != nil {
		panic("creating the package-wide config fixture directory: " + err.Error())
	}
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(defaultTierFixture), 0o600); err != nil {
		panic("writing the package-wide config fixture: " + err.Error())
	}
	if err := os.Setenv(config.EnvVar, path); err != nil {
		panic("pointing " + config.EnvVar + " at the fixture: " + err.Error())
	}
	for _, tier := range config.Tiers {
		if err := os.Unsetenv(reviewer.TierEnvPrefix + strings.ToUpper(tier)); err != nil {
			panic("clearing the tier environment overrides: " + err.Error())
		}
	}

	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

// tierFixture writes contents as this test's tier assignment file and points
// EXTERNAL_REVIEWER_CONFIG at it, returning the path so a test can assert
// that a diagnostic names the file that made the decision.
func tierFixture(t *testing.T, contents string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("writing the tier assignment fixture: %v", err)
	}
	t.Setenv(config.EnvVar, path)
	return path
}

// noTierFixture points EXTERNAL_REVIEWER_CONFIG at a path inside an empty
// temp directory: the file genuinely does not exist, which is the state of a
// machine nobody has set up. It is not the same as an empty file, and the
// distinction is the one SPECS' silent-fallback case turns on.
func noTierFixture(t *testing.T) {
	t.Helper()
	t.Setenv(config.EnvVar, filepath.Join(t.TempDir(), "config.toml"))
}

// catalogRegistry builds an offline registry serving several providers at
// once — what a machine whose tiers point at three different vendors looks
// like. fauxtest.NewRegistry serves one provider, which is all the
// round-trip tests need; a selection test has to prove that a tier resolves
// to *its own* model out of several reachable ones, and that an excluded one
// resolves to nothing rather than to a neighbour.
//
// Every provider answers the same scripted report, so a run that reaches any
// of them terminates normally and the assertion is about which one the model
// line names.
func catalogRegistry(t *testing.T, auth ai.ProviderAuth, catalog map[string][]string) ai.Models {
	t.Helper()

	models := ai.CreateModels(nil)
	for provider, ids := range catalog {
		definitions := make([]faux.ModelDefinition, 0, len(ids))
		for _, id := range ids {
			definitions = append(definitions, faux.ModelDefinition{ID: id})
		}
		handle := faux.New(&faux.Options{Provider: provider, Models: definitions})
		handle.SetResponses(faux.Step(faux.TextMessage(scriptedAnswer, nil)))
		models.SetProvider(fauxtest.NewAuthProvider(handle.Provider, auth))
	}
	return models
}

// runSelection invokes a syntactically valid review with flags under test,
// returning the exit code and both streams. The repository is an empty temp
// directory: nothing here is about what the reviewer reads.
func runSelection(t *testing.T, models ai.Models, flags ...string) (int, string, string) {
	t.Helper()

	argv := append([]string{"review", "--allow", ".", "--prompt", "review this"}, flags...)
	argv = append(argv, t.TempDir())

	var stdout, stderr bytes.Buffer
	code := cli.RunForTest(argv, strings.NewReader(""), &stdout, &stderr, models)
	return code, stdout.String(), stderr.String()
}

// modelLine returns the one `model` line from stderr, or "" when there is
// none — the assertion surface for "which reviewer did this run pick?".
func modelLine(stderr string) string {
	for _, line := range strings.Split(stderr, "\n") {
		if strings.HasPrefix(line, "model ") {
			return line
		}
	}
	return ""
}

// countErrorLines counts the prefixed error: lines on stderr. The exit-code
// contract is stated in exact counts — one line, never two, never none — so
// the assertion has to be a count rather than a Contains.
func countErrorLines(stderr string) int {
	return strings.Count(stderr, "error:  ")
}

// assertUsageError is the shape every malformed invocation takes: exit 2,
// stdout empty, exactly one error: line, and a done line reading stop=usage.
func assertUsageError(t *testing.T, code int, stdout, stderr string) {
	t.Helper()

	if code != 2 {
		t.Errorf("exit code = %d, want 2 (stderr: %q)", code, stderr)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty on a usage error", stdout)
	}
	if n := countErrorLines(stderr); n != 1 {
		t.Errorf("stderr = %q, want exactly one error: line, found %d", stderr, n)
	}
	assertOnlyKnownPrefixedLines(t, stderr)
	if fields := fauxtest.ParseDoneLine(t, stderr); fields.Stop != "usage" {
		t.Errorf("done stop = %q, want usage", fields.Stop)
	}
}

// TestRun_Review_TierResolvesTheConfiguredModel is the flag's whole point:
// the caller names a weight, the machine's own file names the vendor, and the
// model line reports what was actually reached — kern-link's own provider/id
// verbatim, with the credential's source label and nothing else.
func TestRun_Review_TierResolvesTheConfiguredModel(t *testing.T) {
	tierFixture(t, `
[tiers.standard]
provider = "openai-codex"
model    = "gpt-5.5"
`)
	models := catalogRegistry(t, fauxtest.CredentialedAuth("OAuth"), map[string][]string{
		"openai-codex": {"gpt-5.5"},
		"google":       {"gemini-3.1-flash-lite"},
	})

	code, stdout, stderr := runSelection(t, models, "--tier", "standard")

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %q)", code, stderr)
	}
	if stdout != scriptedAnswer {
		t.Errorf("stdout = %q, want the reviewer's answer %q", stdout, scriptedAnswer)
	}
	if got, want := modelLine(stderr), "model   openai-codex/gpt-5.5  auth=OAuth"; got != want {
		t.Errorf("model line = %q, want %q", got, want)
	}
	assertOnlyKnownPrefixedLines(t, stderr)
}

// TestRun_Review_NoConfigAndNoEnvironment_ExitsOneSilently is SPECS'
// silent-fallback case at the surface a script reads: nothing assigns the
// tier, so nothing was ever reached, and the caller gets exit 1 with an empty
// stdout and no error: line to put in its report.
func TestRun_Review_NoConfigAndNoEnvironment_ExitsOneSilently(t *testing.T) {
	noTierFixture(t)
	models := catalogRegistry(t, fauxtest.CredentialedAuth("OAuth"), map[string][]string{
		"openai-codex": {"gpt-5.5"},
	})

	code, stdout, stderr := runSelection(t, models, "--tier", "standard")

	if code != 1 {
		t.Fatalf("exit code = %d, want 1 (stderr: %q)", code, stderr)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty", stdout)
	}
	if n := countErrorLines(stderr); n != 0 {
		t.Errorf("stderr = %q, want no error: line on the silent-fallback path, found %d", stderr, n)
	}
	assertOnlyKnownPrefixedLines(t, stderr)
	if fields := fauxtest.ParseDoneLine(t, stderr); fields.Stop != "no_reviewer" {
		t.Errorf("done stop = %q, want no_reviewer", fields.Stop)
	}
}

// TestRun_Review_UnassignedTier_SaysWhereItLooked is the legibility half of
// the case above. Exit 1 writes no error: line by design, which would leave a
// human with an exit code and nothing else; a warn line naming the file and
// the environment variable that could have assigned the tier is what makes
// the silence diagnosable without being a failure the calling skill has to
// report.
func TestRun_Review_UnassignedTier_SaysWhereItLooked(t *testing.T) {
	noTierFixture(t)
	models := catalogRegistry(t, fauxtest.CredentialedAuth("OAuth"), map[string][]string{
		"openai-codex": {"gpt-5.5"},
	})

	_, _, stderr := runSelection(t, models, "--tier", "heavy")

	if !strings.Contains(stderr, "EXTERNAL_REVIEWER_TIER_HEAVY") {
		t.Errorf("stderr = %q, want it to name the environment variable that assigns this tier", stderr)
	}
	if !strings.Contains(stderr, "config.toml") {
		t.Errorf("stderr = %q, want it to name the config file it looked in", stderr)
	}
}

// TestRun_Review_AnthropicTier_ResolvesToNothingAndNamesNoSubstitute is the
// failure this epic exists to prevent, asserted on the full stderr text: the
// tier is assigned an Anthropic-family model, three other models are reachable
// and credentialed, and the run must end at exit 1 rather than quietly review
// with one of them.
func TestRun_Review_AnthropicTier_ResolvesToNothingAndNamesNoSubstitute(t *testing.T) {
	tierFixture(t, `
[tiers.heavy]
provider = "openrouter"
model    = "anthropic/claude-sonnet-4.5"

[tiers.standard]
provider = "openai-codex"
model    = "gpt-5.5"
`)
	models := catalogRegistry(t, fauxtest.CredentialedAuth("OAuth"), map[string][]string{
		"openrouter":   {"anthropic/claude-sonnet-4.5", "openai/gpt-5.5"},
		"openai-codex": {"gpt-5.5"},
		"google":       {"gemini-3.1-flash-lite"},
	})

	code, stdout, stderr := runSelection(t, models, "--tier", "heavy")

	if code != 1 {
		t.Fatalf("exit code = %d, want 1 (stderr: %q)", code, stderr)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty", stdout)
	}
	if fields := fauxtest.ParseDoneLine(t, stderr); fields.Stop != "no_reviewer" {
		t.Errorf("done stop = %q, want no_reviewer", fields.Stop)
	}
	for _, substitute := range []string{"openai/gpt-5.5", "gpt-5.5", "gemini-3.1-flash-lite"} {
		if strings.Contains(stderr, substitute) {
			t.Errorf("stderr names %q, want no substitute model anywhere in it: %q", substitute, stderr)
		}
	}
	if !strings.Contains(stderr, "anthropic/claude-sonnet-4.5") {
		t.Errorf("stderr = %q, want it to name the model that was refused", stderr)
	}
}

// TestRun_Review_ExplicitModel_BypassesTheConfig is journey 4: the author
// judging a model by hand, without editing the file that lives on the
// machine. The config assigns a different, reachable model, so the wrong
// answer here is a successful review by the wrong reviewer.
func TestRun_Review_ExplicitModel_BypassesTheConfig(t *testing.T) {
	tierFixture(t, `
[tiers.standard]
provider = "openai-codex"
model    = "gpt-5.5"
`)
	models := catalogRegistry(t, fauxtest.CredentialedAuth("OAuth"), map[string][]string{
		"openai-codex": {"gpt-5.5"},
		"google":       {"gemini-3.1-flash-lite"},
	})

	code, _, stderr := runSelection(t, models, "--model", "google/gemini-3.1-flash-lite")

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %q)", code, stderr)
	}
	if got, want := modelLine(stderr), "model   google/gemini-3.1-flash-lite  auth=OAuth"; got != want {
		t.Errorf("model line = %q, want %q — --model bypasses the config's own assignment", got, want)
	}
}

// TestRun_Review_TierAndModelTogether_IsUsageError: the grammar spells the
// two as alternatives, so naming both is a malformed invocation rather than a
// precedence question answered silently.
func TestRun_Review_TierAndModelTogether_IsUsageError(t *testing.T) {
	models := catalogRegistry(t, fauxtest.CredentialedAuth("OAuth"), map[string][]string{
		"openai-codex": {"gpt-5.5"},
	})

	code, stdout, stderr := runSelection(t, models, "--tier", "standard", "--model", "openai-codex/gpt-5.5")

	assertUsageError(t, code, stdout, stderr)
	if !strings.Contains(stderr, "--tier") || !strings.Contains(stderr, "--model") {
		t.Errorf("stderr = %q, want the error: line to name both flags", stderr)
	}
}

// TestRun_Review_UnknownTier_IsUsageError: the tier vocabulary is fixed and
// cannot be extended from the command line any more than from the config
// file, so a misspelled weight is a mistake to report rather than a tier that
// resolves to nothing.
func TestRun_Review_UnknownTier_IsUsageError(t *testing.T) {
	models := catalogRegistry(t, fauxtest.CredentialedAuth("OAuth"), map[string][]string{
		"openai-codex": {"gpt-5.5"},
	})

	code, stdout, stderr := runSelection(t, models, "--tier", "bogus")

	assertUsageError(t, code, stdout, stderr)
	if !strings.Contains(stderr, "bogus") {
		t.Errorf("stderr = %q, want the error: line to quote the value it refused", stderr)
	}
}

// TestRun_Review_EmptyExcludeFamily_IsUsageError: an empty value would
// silently switch off the one safety property this product exists for, so it
// is refused at the grammar rather than honoured as "exclude nothing".
func TestRun_Review_EmptyExcludeFamily_IsUsageError(t *testing.T) {
	models := catalogRegistry(t, fauxtest.CredentialedAuth("OAuth"), map[string][]string{
		"openai-codex": {"gpt-5.5"},
	})

	tests := []struct {
		name  string
		value string
	}{
		{"empty", ""},
		{"whitespace only", "  "},
		{"an empty element in the list", "anthropic,"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			code, stdout, stderr := runSelection(t, models, "--exclude-family", tc.value)

			assertUsageError(t, code, stdout, stderr)
			if !strings.Contains(stderr, "--exclude-family") {
				t.Errorf("stderr = %q, want the error: line to name the flag", stderr)
			}
		})
	}
}

// TestRun_Review_UnknownExcludeFamily_IsUsageError: family.NewExclusion keeps
// a name it does not recognise verbatim, so it never matches anything — which
// means a typo'd `--exclude-family anthropi` excludes nothing at all. That is
// the same fail-open shape the empty value is refused for, so it is refused
// the same way rather than warned about.
func TestRun_Review_UnknownExcludeFamily_IsUsageError(t *testing.T) {
	models := catalogRegistry(t, fauxtest.CredentialedAuth("OAuth"), map[string][]string{
		"openai-codex": {"gpt-5.5"},
	})

	code, stdout, stderr := runSelection(t, models, "--exclude-family", "anthropi")

	assertUsageError(t, code, stdout, stderr)
	if !strings.Contains(stderr, "anthropi") {
		t.Errorf("stderr = %q, want the error: line to quote the name it refused", stderr)
	}
}

// TestRun_Review_ExcludeFamilyList_ExcludesEveryNamedFamily proves the flag
// is a list rather than a boolean, and proves it the only way that means
// anything: the same assignment resolves without the flag and does not
// resolve with it.
func TestRun_Review_ExcludeFamilyList_ExcludesEveryNamedFamily(t *testing.T) {
	tierFixture(t, `
[tiers.standard]
provider = "google"
model    = "gemini-3.1-flash-lite"
`)
	models := catalogRegistry(t, fauxtest.CredentialedAuth("OAuth"), map[string][]string{
		"google":       {"gemini-3.1-flash-lite"},
		"openai-codex": {"gpt-5.5"},
	})

	code, _, stderr := runSelection(t, models)
	if code != 0 {
		t.Fatalf("without the flag: exit code = %d, want 0 (stderr: %q)", code, stderr)
	}

	code, stdout, stderr := runSelection(t, models, "--exclude-family", "anthropic,google")
	if code != 1 {
		t.Fatalf("with --exclude-family anthropic,google: exit code = %d, want 1 (stderr: %q)", code, stderr)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty", stdout)
	}
	if fields := fauxtest.ParseDoneLine(t, stderr); fields.Stop != "no_reviewer" {
		t.Errorf("done stop = %q, want no_reviewer", fields.Stop)
	}
}

// TestRun_Review_NoTierFlag_IsTierStandard: the caller that omits the flag is
// asking for an ordinary review. Asserted as an equivalence rather than
// against a literal, so the two paths cannot drift apart.
func TestRun_Review_NoTierFlag_IsTierStandard(t *testing.T) {
	tierFixture(t, `
[tiers.light]
provider = "google"
model    = "gemini-3.1-flash-lite"

[tiers.standard]
provider = "openai-codex"
model    = "gpt-5.5"
`)
	catalog := map[string][]string{
		"openai-codex": {"gpt-5.5"},
		"google":       {"gemini-3.1-flash-lite"},
	}

	// Two registries rather than one: each provider is scripted with a single
	// response, so a second run against the same handle would starve rather
	// than resolve — and would fail this test for a reason that has nothing
	// to do with the tier default.
	explicitCode, _, explicitStderr := runSelection(t, catalogRegistry(t, fauxtest.CredentialedAuth("OAuth"), catalog), "--tier", "standard")
	defaultCode, _, defaultStderr := runSelection(t, catalogRegistry(t, fauxtest.CredentialedAuth("OAuth"), catalog))

	if explicitCode != 0 || defaultCode != 0 {
		t.Fatalf("exit codes = %d (explicit) / %d (default), want 0 and 0", explicitCode, defaultCode)
	}
	if got, want := modelLine(defaultStderr), modelLine(explicitStderr); got != want {
		t.Errorf("model line without --tier = %q, want the same as with --tier standard: %q", got, want)
	}
}

// TestRun_Review_MalformedExplicitModel_IsUsageError is the first half of the
// deliberate split this PR makes over the third failure class. A
// *MalformedAssignmentError from --model is a value the caller typed on the
// command line, so it belongs with every other malformed invocation:
// stop=usage.
func TestRun_Review_MalformedExplicitModel_IsUsageError(t *testing.T) {
	models := catalogRegistry(t, fauxtest.CredentialedAuth("OAuth"), map[string][]string{
		"openai-codex": {"gpt-5.5"},
	})

	code, stdout, stderr := runSelection(t, models, "--model", "gpt-5.5")

	assertUsageError(t, code, stdout, stderr)
	if !strings.Contains(stderr, "gpt-5.5") {
		t.Errorf("stderr = %q, want the error: line to quote the value it could not parse", stderr)
	}
}

// TestRun_Review_MalformedEnvironmentAssignment_IsAFailure is the other half:
// the same error class from a layer the invocation did not choose. Nothing
// about the command line is wrong, so calling it a usage error would send a
// caller to reread its own argv; it is exit 2 with stop=failed and a line
// naming the variable a human has to fix.
func TestRun_Review_MalformedEnvironmentAssignment_IsAFailure(t *testing.T) {
	noTierFixture(t)
	t.Setenv("EXTERNAL_REVIEWER_TIER_STANDARD", "gpt-5.5")
	models := catalogRegistry(t, fauxtest.CredentialedAuth("OAuth"), map[string][]string{
		"openai-codex": {"gpt-5.5"},
	})

	code, stdout, stderr := runSelection(t, models, "--tier", "standard")

	if code != 2 {
		t.Fatalf("exit code = %d, want 2 (stderr: %q)", code, stderr)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty", stdout)
	}
	if n := countErrorLines(stderr); n != 1 {
		t.Errorf("stderr = %q, want exactly one error: line, found %d", stderr, n)
	}
	assertOnlyKnownPrefixedLines(t, stderr)
	if fields := fauxtest.ParseDoneLine(t, stderr); fields.Stop != "failed" {
		t.Errorf("done stop = %q, want failed — argv was well formed, the machine's configuration is not", fields.Stop)
	}
	if !strings.Contains(stderr, "EXTERNAL_REVIEWER_TIER_STANDARD") {
		t.Errorf("stderr = %q, want it to name the variable that carried the malformed value", stderr)
	}
}

// TestRun_Review_ExcludedTier_SaysWhichLayerAssignedIt: a correctly spelled
// tier that resolves to nothing is the hardest state to debug, because the
// exit code alone cannot say whether the file, the environment or the flag
// chose the model that was refused. The warn line names the origin.
func TestRun_Review_ExcludedTier_SaysWhichLayerAssignedIt(t *testing.T) {
	path := tierFixture(t, `
[tiers.standard]
provider = "openrouter"
model    = "anthropic/claude-sonnet-4.5"
`)
	models := catalogRegistry(t, fauxtest.CredentialedAuth("OAuth"), map[string][]string{
		"openrouter": {"anthropic/claude-sonnet-4.5"},
	})

	code, _, stderr := runSelection(t, models, "--tier", "standard")

	if code != 1 {
		t.Fatalf("exit code = %d, want 1 (stderr: %q)", code, stderr)
	}
	if !strings.Contains(stderr, path) {
		t.Errorf("stderr = %q, want it to name the config file that assigned the refused model (%s)", stderr, path)
	}
}

// TestRun_Review_EnvironmentOverride_ResolvesAndNamesItsOrigin drives the
// middle layer of the precedence chain through the command line, which is the
// only place the environment layer is reachable from at all — `--model`
// bypasses it and the config file loses to it.
func TestRun_Review_EnvironmentOverride_ResolvesAndNamesItsOrigin(t *testing.T) {
	tierFixture(t, `
[tiers.standard]
provider = "openai-codex"
model    = "gpt-5.5"
`)
	t.Setenv("EXTERNAL_REVIEWER_TIER_STANDARD", "google/gemini-3.1-flash-lite")
	models := catalogRegistry(t, fauxtest.CredentialedAuth("OAuth"), map[string][]string{
		"openai-codex": {"gpt-5.5"},
		"google":       {"gemini-3.1-flash-lite"},
	})

	code, _, stderr := runSelection(t, models, "--tier", "standard")

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %q)", code, stderr)
	}
	if got, want := modelLine(stderr), "model   google/gemini-3.1-flash-lite  auth=OAuth"; got != want {
		t.Errorf("model line = %q, want %q — the environment overrides the file", got, want)
	}
}

// TestRun_Review_BrokenCredentialOnATier_ExitsTwo is the sharp edge of the
// taxonomy at the surface a script reads: a credential that exists and does
// not work is worth exactly one line in the caller's report, never the silent
// exit 1 an unconfigured provider gets.
func TestRun_Review_BrokenCredentialOnATier_ExitsTwo(t *testing.T) {
	tierFixture(t, `
[tiers.standard]
provider = "openai-codex"
model    = "gpt-5.5"
`)
	models := catalogRegistry(t, fauxtest.BrokenAPIKeyAuth(), map[string][]string{
		"openai-codex": {"gpt-5.5"},
	})

	code, stdout, stderr := runSelection(t, models, "--tier", "standard")

	if code != 2 {
		t.Fatalf("exit code = %d, want 2 (stderr: %q)", code, stderr)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty", stdout)
	}
	if n := countErrorLines(stderr); n != 1 {
		t.Errorf("stderr = %q, want exactly one error: line, found %d", stderr, n)
	}
	assertOnlyKnownPrefixedLines(t, stderr)
	if fields := fauxtest.ParseDoneLine(t, stderr); fields.Stop != "failed" {
		t.Errorf("done stop = %q, want failed", fields.Stop)
	}
	if strings.Contains(stderr, fauxtest.Secret) {
		t.Errorf("stderr leaked a credential value: %q", stderr)
	}
}

// mentionsFlag reports whether usage documents the flag named name, matching
// it as a whole word. A plain Contains would find "--all" inside "--allow"
// and report that this binary promises a flag only the models command will
// ever have.
func mentionsFlag(usage, name string) bool {
	for _, field := range strings.Fields(usage) {
		if strings.TrimRight(field, ",.") == name {
			return true
		}
	}
	return false
}

// TestUsage_ListsExactlyTheFlagsThatWork: the usage text is a promise, and a
// promise of a flag that is itself a usage error is worse than silence. It
// must gain the three flags this PR ships and still name none of the surfaces
// later issues add.
func TestUsage_ListsExactlyTheFlagsThatWork(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := cli.Run([]string{"help"}, strings.NewReader(""), &stdout, &stderr); code != 0 {
		t.Fatalf("help exit code = %d, want 0", code)
	}
	usage := stdout.String()

	for _, flag := range []string{"--allow", "--prompt", "--tier", "--model", "--exclude-family"} {
		if !mentionsFlag(usage, flag) {
			t.Errorf("usage text does not document %s, which this binary accepts:\n%s", flag, usage)
		}
	}
	for _, absent := range []string{"--system"} {
		if mentionsFlag(usage, absent) {
			t.Errorf("usage text promises %s, which is still a usage error:\n%s", absent, usage)
		}
	}
	for _, absent := range []string{"version"} {
		if strings.Contains(usage, "\n  "+absent) {
			t.Errorf("usage text promises the %s command, which does not exist yet:\n%s", absent, usage)
		}
	}
}

// TestBinary_NamesNoProviderOutsideTheFamilyTables is the issue's grep
// criterion as a test, so it keeps holding: after this PR the mechanism knows
// no vendor. The family classifier's data table is the one place a provider
// id may be written down, because classifying one is its entire job.
//
// Test files are exempt — a fixture has to name something reachable — and so
// are dot-directories, which on the development machine hold sibling git
// worktrees of this same repository.
func TestBinary_NamesNoProviderOutsideTheFamilyTables(t *testing.T) {
	root := filepath.Join("..", "..")
	allowed := filepath.Join(root, "internal", "family", "vendors.go")

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			// path != root matters: the root is spelled "..\..", whose
			// Name() is "..", and skipping it would make this whole test
			// vacuously pass.
			if path != root && strings.HasPrefix(entry.Name(), ".") {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") || path == allowed {
			return nil
		}
		source, err := os.ReadFile(path) //nolint:gosec // G304/G122: a test reads this module's own source tree, whose paths come from walking it
		if err != nil {
			return err
		}
		if bytes.Contains(source, []byte("openai-codex")) {
			t.Errorf("%s names a provider; only %s may", path, allowed)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking the module: %v", err)
	}
}
