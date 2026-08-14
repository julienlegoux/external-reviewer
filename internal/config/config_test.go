package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/julienlegoux/external-reviewer/internal/config"
)

// threeTierFixture is the worked example from specs 05 verbatim: three
// tiers, each with provider and model set, plus the human comments TOML was
// chosen to carry (which a decoder must ignore, not warn about).
const threeTierFixture = `# Reviewer assignment. Judgments about which model suits which weight of review.
# Never committed to a repository.

[tiers.light]
provider = "google"
model    = "gemini-3.1-flash-lite"
# Issue files against one epic — a cheap read, and volume matters more than depth.

[tiers.standard]
provider = "google"
model    = "gemini-3.1-pro-preview"

[tiers.heavy]
provider = "google"
model    = "gemini-3.5-flash"
# An entire epic's merged diff. Wants the largest context reachable on this machine.
`

func writeFixture(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}
	return path
}

func TestLoadFile_ThreeTierFixture_DecodesEveryAssignmentVerbatim(t *testing.T) {
	path := writeFixture(t, threeTierFixture)

	cfg, err := config.LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile: unexpected error: %v", err)
	}

	want := map[string]config.Assignment{
		"light":    {Provider: "google", Model: "gemini-3.1-flash-lite"},
		"standard": {Provider: "google", Model: "gemini-3.1-pro-preview"},
		"heavy":    {Provider: "google", Model: "gemini-3.5-flash"},
	}
	if len(cfg.Assignments) != len(want) {
		t.Fatalf("Assignments = %#v, want %#v", cfg.Assignments, want)
	}
	for tier, assignment := range want {
		got, ok := cfg.Assignments[tier]
		if !ok {
			t.Errorf("tier %q: missing assignment", tier)
			continue
		}
		if got != assignment {
			t.Errorf("tier %q: assignment = %#v, want %#v", tier, got, assignment)
		}
	}
	if len(cfg.Warnings) != 0 {
		t.Errorf("Warnings = %v, want none", cfg.Warnings)
	}
	if cfg.Path != path {
		t.Errorf("Path = %q, want %q", cfg.Path, path)
	}
}

func TestLoadFile_AbsentFile_ReturnsNoAssignmentsAndNoError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist.toml")

	cfg, err := config.LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile: unexpected error for an absent file: %v", err)
	}
	if len(cfg.Assignments) != 0 {
		t.Errorf("Assignments = %#v, want none", cfg.Assignments)
	}
	if len(cfg.Warnings) != 0 {
		t.Errorf("Warnings = %v, want none", cfg.Warnings)
	}
	if cfg.Path != path {
		t.Errorf("Path = %q, want %q", cfg.Path, path)
	}
}

func TestLocate_EnvVarSet_TakesPrecedenceOverPlatformDiscovery(t *testing.T) {
	want := filepath.Join(t.TempDir(), "somewhere-else", "config.toml")
	t.Setenv(config.EnvVar, want)

	got, err := config.Locate()
	if err != nil {
		t.Fatalf("Locate: unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("Locate() = %q, want %q", got, want)
	}
}

func TestLocate_EnvVarUnset_UsesUserConfigDir(t *testing.T) {
	t.Setenv(config.EnvVar, "")

	got, err := config.Locate()
	if err != nil {
		t.Fatalf("Locate: unexpected error: %v", err)
	}

	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		t.Fatalf("os.UserConfigDir: unexpected error: %v", err)
	}
	want := filepath.Join(userConfigDir, "external-reviewer", "config.toml")
	if got != want {
		t.Errorf("Locate() = %q, want %q", got, want)
	}
}

func TestLoadFile_UnrecognisedKey_WarnsInSpecsWordingAndLeavesTierUnassigned(t *testing.T) {
	path := writeFixture(t, `
[tiers.standard]
provider = "openai-codex"
models   = "gpt-5.5"
`)

	cfg, err := config.LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile: unexpected error: %v", err)
	}

	wantWarning := `config: unrecognised key "models" in [tiers.standard]`
	if len(cfg.Warnings) != 1 || cfg.Warnings[0] != wantWarning {
		t.Errorf("Warnings = %v, want exactly [%q]", cfg.Warnings, wantWarning)
	}
	if _, ok := cfg.Assignments["standard"]; ok {
		t.Errorf("Assignments[standard] = %#v, want no assignment (missing model)", cfg.Assignments["standard"])
	}
}

// TestLoadFile_RootLevelUnrecognisedKey_WarnsNamingTheKeyAndAssignsTheRest is
// the same criterion as the nested case above — a typo is warned, not silently
// ignored — for the key path that has no enclosing table to name. It is also
// the likelier typo: the tier file is the first thing a user edits, and a key
// written before any [tiers.<name>] header lands at the root.
func TestLoadFile_RootLevelUnrecognisedKey_WarnsNamingTheKeyAndAssignsTheRest(t *testing.T) {
	path := writeFixture(t, `
verbose = true

[tiers.heavy]
provider = "openai-codex"
model    = "gpt-5.6-sol"
`)

	cfg, err := config.LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile: unexpected error: %v", err)
	}

	wantWarning := `config: unrecognised key "verbose" at the top level`
	if len(cfg.Warnings) != 1 || cfg.Warnings[0] != wantWarning {
		t.Errorf("Warnings = %v, want exactly [%q]", cfg.Warnings, wantWarning)
	}
	// The unrecognised key explains itself and nothing else: a well-formed
	// tier alongside it is still assigned.
	if got := cfg.Assignments["heavy"]; got != (config.Assignment{Provider: "openai-codex", Model: "gpt-5.6-sol"}) {
		t.Errorf("Assignments[heavy] = %#v, want the tier assigned despite the root-level key", got)
	}
}

func TestLoadFile_FamilyKey_WarnedAsUnrecognisedAndNeverHonoured(t *testing.T) {
	path := writeFixture(t, `
[tiers.standard]
provider = "openai-codex"
model    = "gpt-5.5"
family   = "openai"
`)

	cfg, err := config.LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile: unexpected error: %v", err)
	}

	wantWarning := `config: unrecognised key "family" in [tiers.standard]`
	if len(cfg.Warnings) != 1 || cfg.Warnings[0] != wantWarning {
		t.Errorf("Warnings = %v, want exactly [%q]", cfg.Warnings, wantWarning)
	}
	got, ok := cfg.Assignments["standard"]
	if !ok {
		t.Fatalf("Assignments[standard] missing, want provider/model assigned despite the extra key")
	}
	if got != (config.Assignment{Provider: "openai-codex", Model: "gpt-5.5"}) {
		t.Errorf("Assignments[standard] = %#v, family must never be honoured", got)
	}
}

func TestLoadFile_TierMissingRequiredKey_WarnsAndLeavesTierUnassigned(t *testing.T) {
	tests := []struct {
		name    string
		toml    string
		missing string
	}{
		{
			name:    "missing provider",
			toml:    "[tiers.standard]\nmodel = \"gpt-5.5\"\n",
			missing: "provider",
		},
		{
			name:    "missing model",
			toml:    "[tiers.standard]\nprovider = \"openai-codex\"\n",
			missing: "model",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeFixture(t, tt.toml)

			cfg, err := config.LoadFile(path)
			if err != nil {
				t.Fatalf("LoadFile: unexpected error: %v", err)
			}

			wantWarning := `config: [tiers.standard] missing required key "` + tt.missing + `"`
			if len(cfg.Warnings) != 1 || cfg.Warnings[0] != wantWarning {
				t.Errorf("Warnings = %v, want exactly [%q]", cfg.Warnings, wantWarning)
			}
			if _, ok := cfg.Assignments["standard"]; ok {
				t.Errorf("Assignments[standard] = %#v, want no half-assignment", cfg.Assignments["standard"])
			}
		})
	}
}

func TestLoadFile_MalformedTOML_ReturnsErrorNamingTheFile(t *testing.T) {
	path := writeFixture(t, `[tiers.standard\nprovider = "unterminated table header"`)

	_, err := config.LoadFile(path)
	if err == nil {
		t.Fatal("LoadFile: want an error for malformed TOML, got nil")
	}
	if got := err.Error(); !strings.Contains(got, path) {
		t.Errorf("error %q does not name the file %q", got, path)
	}
}

func TestLoadFile_TierNameOutsideVocabulary_WarnedAndIgnored(t *testing.T) {
	path := writeFixture(t, `
[tiers.urgent]
provider = "openai-codex"
model    = "gpt-5.5"
`)

	cfg, err := config.LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile: unexpected error: %v", err)
	}

	wantWarning := `config: unrecognised tier "urgent"`
	if len(cfg.Warnings) != 1 || cfg.Warnings[0] != wantWarning {
		t.Errorf("Warnings = %v, want exactly [%q]", cfg.Warnings, wantWarning)
	}
	if len(cfg.Assignments) != 0 {
		t.Errorf("Assignments = %#v, want none", cfg.Assignments)
	}
}
