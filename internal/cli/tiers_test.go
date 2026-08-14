package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/julienlegoux/kern-link/ai"

	"github.com/julienlegoux/external-reviewer/internal/cli"
	"github.com/julienlegoux/external-reviewer/internal/fauxtest"
)

// runTiers invokes the `tiers` subcommand against models, returning the exit
// code and both streams. Unlike runSelection, no repository path or --allow
// is needed: tiers never opens a repository.
func runTiers(t *testing.T, models ai.Models, flags ...string) (int, string, string) {
	t.Helper()

	argv := append([]string{"tiers"}, flags...)
	var stdout, stderr bytes.Buffer
	code := cli.RunForTest(argv, strings.NewReader(""), &stdout, &stderr, models)
	return code, stdout.String(), stderr.String()
}

// tierLine returns the row of stdout naming tier at the start of the line,
// or "" when there is none — the assertion surface for "what did this tier
// resolve to?", mirroring selection_test.go's modelLine.
func tierLine(stdout, tier string) string {
	for _, line := range strings.Split(stdout, "\n") {
		fields := strings.Fields(line)
		if len(fields) > 0 && fields[0] == tier {
			return line
		}
	}
	return ""
}

// TestRun_Tiers_AllThreeAssigned_PrintsOneRowEachAndExitsZero is written
// failing first: a fixture assigning all three tiers to reachable models
// prints one row per tier, each naming provider/id, and exits 0.
func TestRun_Tiers_AllThreeAssigned_PrintsOneRowEachAndExitsZero(t *testing.T) {
	tierFixture(t, `
[tiers.light]
provider = "google"
model    = "gemini-3.1-flash-lite"

[tiers.standard]
provider = "openai-codex"
model    = "gpt-5.5"

[tiers.heavy]
provider = "google"
model    = "gemini-3.1-flash-lite"
`)
	models := catalogRegistry(t, fauxtest.CredentialedAuth("OAuth"), map[string][]string{
		"google":       {"gemini-3.1-flash-lite"},
		"openai-codex": {"gpt-5.5"},
	})

	code, stdout, stderr := runTiers(t, models)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %q)", code, stderr)
	}
	for _, tc := range []struct {
		tier string
		want string
	}{
		{"light", "google/gemini-3.1-flash-lite"},
		{"standard", "openai-codex/gpt-5.5"},
		{"heavy", "google/gemini-3.1-flash-lite"},
	} {
		line := tierLine(stdout, tc.tier)
		if line == "" {
			t.Fatalf("stdout has no row for tier %q:\n%s", tc.tier, stdout)
		}
		if !strings.Contains(line, tc.want) {
			t.Errorf("row for %q = %q, want it to name %q", tc.tier, line, tc.want)
		}
	}
}

// TestRun_Tiers_NoConfigFile_EveryTierUnassignedAndPathPrinted: with no
// config file at all, every tier reports unassigned, the path that was
// looked at is printed, and the exit code is 0.
func TestRun_Tiers_NoConfigFile_EveryTierUnassignedAndPathPrinted(t *testing.T) {
	noTierFixture(t)
	models := catalogRegistry(t, fauxtest.CredentialedAuth("OAuth"), map[string][]string{
		"openai-codex": {"gpt-5.5"},
	})

	code, stdout, stderr := runTiers(t, models)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %q)", code, stderr)
	}
	for _, tier := range []string{"light", "standard", "heavy"} {
		line := tierLine(stdout, tier)
		if !strings.Contains(line, "unassigned") {
			t.Errorf("row for %q = %q, want it to report unassigned", tier, line)
		}
	}
	if !strings.Contains(stdout, "config.toml") {
		t.Errorf("stdout = %q, want it to print the config path that was looked at", stdout)
	}
}

// TestRun_Tiers_TypoKeyFixture_OneWarnLineAndTierUnassigned: the `models =`
// typo fixture produces exactly one warn line on stderr naming the key and
// the tier, that tier reports unassigned on stdout, and the command still
// exits 0 — the epic's acceptance criterion that an unrecognised key
// produces a warn line rather than a silently missing tier.
func TestRun_Tiers_TypoKeyFixture_OneWarnLineAndTierUnassigned(t *testing.T) {
	tierFixture(t, `
[tiers.standard]
provider = "openai-codex"
models   = "gpt-5.5"
`)
	models := catalogRegistry(t, fauxtest.CredentialedAuth("OAuth"), map[string][]string{
		"openai-codex": {"gpt-5.5"},
	})

	code, stdout, stderr := runTiers(t, models)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %q)", code, stderr)
	}
	wantWarning := `config: unrecognised key "models" in [tiers.standard]`
	if n := strings.Count(stderr, wantWarning); n != 1 {
		t.Errorf("stderr = %q, want exactly one warn line %q, found %d", stderr, wantWarning, n)
	}
	line := tierLine(stdout, "standard")
	if !strings.Contains(line, "unassigned") {
		t.Errorf("row for standard = %q, want unassigned", line)
	}
}

// TestRun_Tiers_RootLevelTypoKeyFixture_ExitsThroughTheNormalPath is the same
// criterion one key path over, and it is written against the command rather
// than against config.LoadFile because what it guards is the shape of the
// failure: a root-level key used to end the process from inside the warning
// builder, so the run produced no error: line, a done line with an empty
// stop=, and an exit code that came from the runtime rather than from the
// classification.
func TestRun_Tiers_RootLevelTypoKeyFixture_ExitsThroughTheNormalPath(t *testing.T) {
	tierFixture(t, `
verbose = true

[tiers.standard]
provider = "openai-codex"
model    = "gpt-5.5"
`)
	models := catalogRegistry(t, fauxtest.CredentialedAuth("OAuth"), map[string][]string{
		"openai-codex": {"gpt-5.5"},
	})

	code, stdout, stderr := runTiers(t, models)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %q)", code, stderr)
	}
	wantWarning := `config: unrecognised key "verbose" at the top level`
	if n := strings.Count(stderr, wantWarning); n != 1 {
		t.Errorf("stderr = %q, want exactly one warn line %q, found %d", stderr, wantWarning, n)
	}
	if !strings.Contains(stderr, "stop=answered") {
		t.Errorf("stderr = %q, want a done line carrying a real stop= word", stderr)
	}
	if strings.Contains(stderr, "error:") {
		t.Errorf("stderr = %q, want no error: line — an unrecognised key is a warning", stderr)
	}
	// The rest of the file is still honoured: the warning explains the key it
	// names and nothing else.
	if line := tierLine(stdout, "standard"); !strings.Contains(line, "gpt-5.5") {
		t.Errorf("row for standard = %q, want the tier assigned despite the root-level key", line)
	}
}

// TestRun_Tiers_AnthropicFamilyTier_ReportsExcludedByFamily: a tier whose
// model is Anthropic-family reports excluded by family, naming anthropic —
// not "model not found", which would send the reader looking for a catalog
// problem that does not exist.
func TestRun_Tiers_AnthropicFamilyTier_ReportsExcludedByFamily(t *testing.T) {
	tierFixture(t, `
[tiers.standard]
provider = "openrouter"
model    = "anthropic/claude-sonnet-4.5"
`)
	models := catalogRegistry(t, fauxtest.CredentialedAuth("OAuth"), map[string][]string{
		"openrouter": {"anthropic/claude-sonnet-4.5"},
	})

	code, stdout, stderr := runTiers(t, models)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %q)", code, stderr)
	}
	line := tierLine(stdout, "standard")
	if !strings.Contains(line, "excluded") || !strings.Contains(line, "anthropic") {
		t.Errorf("row for standard = %q, want it to report excluded by family, naming anthropic", line)
	}
	if strings.Contains(line, "not in the catalog") || strings.Contains(line, "not found") {
		t.Errorf("row for standard = %q, want no catalog-shaped wording for a family exclusion", line)
	}
}

// TestRun_Tiers_UnknownFamilyTier_ReportsExcludedNotReachable: a tier whose
// model is of unknown family is excluded, not admitted, and the row must say
// so rather than reading as reachable.
func TestRun_Tiers_UnknownFamilyTier_ReportsExcludedNotReachable(t *testing.T) {
	tierFixture(t, `
[tiers.standard]
provider = "some-unclassified-provider"
model    = "mystery-model"
`)
	models := catalogRegistry(t, fauxtest.CredentialedAuth("OAuth"), map[string][]string{
		"some-unclassified-provider": {"mystery-model"},
	})

	code, stdout, stderr := runTiers(t, models)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %q)", code, stderr)
	}
	line := tierLine(stdout, "standard")
	if !strings.Contains(line, "excluded") {
		t.Errorf("row for standard = %q, want it to report excluded", line)
	}
	if strings.Contains(line, "reachable") {
		t.Errorf("row for standard = %q, want it not to read as reachable", line)
	}
}

// TestRun_Tiers_UnconfiguredProvider_DistinctFromBrokenCredential: a tier
// whose provider has no credential reports provider unconfigured, distinctly
// from a broken credential.
func TestRun_Tiers_UnconfiguredProvider_DistinctFromBrokenCredential(t *testing.T) {
	tierFixture(t, `
[tiers.standard]
provider = "openai-codex"
model    = "gpt-5.5"
`)

	t.Run("unconfigured", func(t *testing.T) {
		models := catalogRegistry(t, fauxtest.UnconfiguredAuth(), map[string][]string{
			"openai-codex": {"gpt-5.5"},
		})
		code, stdout, stderr := runTiers(t, models)
		if code != 0 {
			t.Fatalf("exit code = %d, want 0 (stderr: %q)", code, stderr)
		}
		line := tierLine(stdout, "standard")
		if !strings.Contains(line, "provider unconfigured") {
			t.Errorf("row for standard = %q, want it to report provider unconfigured", line)
		}
	})

	t.Run("broken credential", func(t *testing.T) {
		models := catalogRegistry(t, fauxtest.BrokenAPIKeyAuth(), map[string][]string{
			"openai-codex": {"gpt-5.5"},
		})
		code, stdout, stderr := runTiers(t, models)
		if code != 0 {
			t.Fatalf("exit code = %d, want 0 (stderr: %q)", code, stderr)
		}
		line := tierLine(stdout, "standard")
		if !strings.Contains(line, "credential broken") {
			t.Errorf("row for standard = %q, want it to report credential broken", line)
		}
		if strings.Contains(line, "provider unconfigured") {
			t.Errorf("row for standard = %q, want a broken credential distinguished from an unconfigured one", line)
		}
	})
}

// TestRun_Tiers_EnvironmentOverride_NamesItsOriginOtherTiersReadTheConfig:
// EXTERNAL_REVIEWER_TIER_STANDARD set — the standard row names the
// environment variable as the source, and the other two rows still read
// from the config.
func TestRun_Tiers_EnvironmentOverride_NamesItsOriginOtherTiersReadTheConfig(t *testing.T) {
	tierFixture(t, `
[tiers.light]
provider = "google"
model    = "gemini-3.1-flash-lite"

[tiers.standard]
provider = "openai-codex"
model    = "gpt-5.5"

[tiers.heavy]
provider = "google"
model    = "gemini-3.1-flash-lite"
`)
	t.Setenv("EXTERNAL_REVIEWER_TIER_STANDARD", "google/gemini-3.1-flash-lite")
	models := catalogRegistry(t, fauxtest.CredentialedAuth("OAuth"), map[string][]string{
		"google":       {"gemini-3.1-flash-lite"},
		"openai-codex": {"gpt-5.5"},
	})

	code, stdout, stderr := runTiers(t, models)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %q)", code, stderr)
	}
	standardLine := tierLine(stdout, "standard")
	if !strings.Contains(standardLine, "EXTERNAL_REVIEWER_TIER_STANDARD") {
		t.Errorf("row for standard = %q, want it to name the environment variable as the source", standardLine)
	}
	for _, tier := range []string{"light", "heavy"} {
		line := tierLine(stdout, tier)
		if !strings.Contains(line, "config") {
			t.Errorf("row for %q = %q, want it to still read from the config", tier, line)
		}
	}
}

// TestRun_Tiers_ExcludeFamilyFlag_FlipsAGoogleTierToExcluded proves the
// flag is honoured the same way review's is: the same assignment resolves
// without it and does not resolve with it.
func TestRun_Tiers_ExcludeFamilyFlag_FlipsAGoogleTierToExcluded(t *testing.T) {
	tierFixture(t, `
[tiers.standard]
provider = "google"
model    = "gemini-3.1-flash-lite"
`)
	models := catalogRegistry(t, fauxtest.CredentialedAuth("OAuth"), map[string][]string{
		"google": {"gemini-3.1-flash-lite"},
	})

	code, stdout, stderr := runTiers(t, models)
	if code != 0 {
		t.Fatalf("without the flag: exit code = %d, want 0 (stderr: %q)", code, stderr)
	}
	if line := tierLine(stdout, "standard"); !strings.Contains(line, "reachable") {
		t.Fatalf("without the flag: row = %q, want reachable", line)
	}

	models = catalogRegistry(t, fauxtest.CredentialedAuth("OAuth"), map[string][]string{
		"google": {"gemini-3.1-flash-lite"},
	})
	code, stdout, stderr = runTiers(t, models, "--exclude-family", "anthropic,google")
	if code != 0 {
		t.Fatalf("with the flag: exit code = %d, want 0 (stderr: %q)", code, stderr)
	}
	line := tierLine(stdout, "standard")
	if !strings.Contains(line, "excluded") {
		t.Errorf("with --exclude-family anthropic,google: row = %q, want excluded", line)
	}
}

// TestRun_Tiers_StdoutCarriesOnlyTheTable: every warning and diagnostic
// belongs on stderr, so the stdout table must stay parseable even when a
// warning fires.
func TestRun_Tiers_StdoutCarriesOnlyTheTable(t *testing.T) {
	tierFixture(t, `
[tiers.standard]
provider = "openai-codex"
models   = "gpt-5.5"
`)
	models := catalogRegistry(t, fauxtest.CredentialedAuth("OAuth"), map[string][]string{
		"openai-codex": {"gpt-5.5"},
	})

	_, stdout, _ := runTiers(t, models)

	for _, forbidden := range []string{"warn", "error:", "done"} {
		for _, line := range strings.Split(stdout, "\n") {
			fields := strings.Fields(line)
			if len(fields) > 0 && fields[0] == forbidden {
				t.Errorf("stdout contains a %q-prefixed diagnostic line: %q (full stdout: %q)", forbidden, line, stdout)
			}
		}
	}
}

// TestRun_Tiers_UnknownFlag_ExitsTwo and
// TestRun_Tiers_ExtraPositional_ExitsTwo are the malformed-invocation half of
// the exit-code contract, matching review's own usage-error shape.
func TestRun_Tiers_UnknownFlag_ExitsTwo(t *testing.T) {
	models := catalogRegistry(t, fauxtest.CredentialedAuth("OAuth"), map[string][]string{
		"openai-codex": {"gpt-5.5"},
	})

	code, stdout, stderr := runTiers(t, models, "--nope")

	assertUsageError(t, code, stdout, stderr)
}

func TestRun_Tiers_ExtraPositional_ExitsTwo(t *testing.T) {
	models := catalogRegistry(t, fauxtest.CredentialedAuth("OAuth"), map[string][]string{
		"openai-codex": {"gpt-5.5"},
	})

	code, stdout, stderr := runTiers(t, models, "extra-positional")

	assertUsageError(t, code, stdout, stderr)
}

// TestRun_Tiers_NoCredentialValueReachesEitherStream is the security
// boundary: only AuthResult.Source may ever travel, whatever state a tier's
// credential is in.
func TestRun_Tiers_NoCredentialValueReachesEitherStream(t *testing.T) {
	oauthAuth, oauthCredentials := fauxtest.ExpiredOAuthAuth(t, "openai-codex")

	tests := []struct {
		name string
		auth ai.ProviderAuth
	}{
		{"resolved successfully", fauxtest.CredentialedAuth("OAuth")},
		{"broken api key", fauxtest.BrokenAPIKeyAuth()},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tierFixture(t, `
[tiers.standard]
provider = "openai-codex"
model    = "gpt-5.5"
`)
			models := catalogRegistry(t, tc.auth, map[string][]string{"openai-codex": {"gpt-5.5"}})
			_, stdout, stderr := runTiers(t, models)
			if strings.Contains(stdout, fauxtest.Secret) {
				t.Errorf("stdout leaked a credential value: %q", stdout)
			}
			if strings.Contains(stderr, fauxtest.Secret) {
				t.Errorf("stderr leaked a credential value: %q", stderr)
			}
		})
	}

	t.Run("broken oauth", func(t *testing.T) {
		tierFixture(t, `
[tiers.standard]
provider = "openai-codex"
model    = "gpt-5.5"
`)
		_, stdout, stderr := runTiers(t, registry(t, oauthAuth, oauthCredentials, "gpt-5.5"))
		if strings.Contains(stdout, fauxtest.Secret) {
			t.Errorf("stdout leaked a credential value: %q", stdout)
		}
		if strings.Contains(stderr, fauxtest.Secret) {
			t.Errorf("stderr leaked a credential value: %q", stderr)
		}
	})
}

// TestRun_Tiers_MalformedEnvironmentAssignment_ExitsTwo is not itself named
// as an acceptance criterion, but resolveTierRow has to do something with a
// value that is not provider/id, and "the query could not be answered at
// all" (the issue's own words for the exit-2 case) is exactly what a
// hand-typed environment override that cannot even be parsed amounts to.
func TestRun_Tiers_MalformedEnvironmentAssignment_ExitsTwo(t *testing.T) {
	noTierFixture(t)
	t.Setenv("EXTERNAL_REVIEWER_TIER_STANDARD", "gpt-5.5")
	models := catalogRegistry(t, fauxtest.CredentialedAuth("OAuth"), map[string][]string{
		"openai-codex": {"gpt-5.5"},
	})

	code, stdout, stderr := runTiers(t, models)

	if code != 2 {
		t.Fatalf("exit code = %d, want 2 (stderr: %q)", code, stderr)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty when the query could not be answered", stdout)
	}
	if !strings.Contains(stderr, "EXTERNAL_REVIEWER_TIER_STANDARD") {
		t.Errorf("stderr = %q, want it to name the variable that carried the malformed value", stderr)
	}
}

// TestUsage_DocumentsTiersCommand: usageText must promise the command that
// now exists.
func TestUsage_DocumentsTiersCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := cli.Run([]string{"help"}, strings.NewReader(""), &stdout, &stderr); code != 0 {
		t.Fatalf("help exit code = %d, want 0", code)
	}
	usage := stdout.String()

	if !strings.Contains(usage, "\n  tiers") {
		t.Errorf("usage text does not document the tiers command:\n%s", usage)
	}
}
