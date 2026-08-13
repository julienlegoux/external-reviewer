// Package config reads the machine-local tier assignment file: one
// hand-written, never-committed TOML that assigns a provider and model to
// each weight tier (specs 05, specs 06). It only ever reads — locating the
// file, decoding it and reporting every warning worth a human's attention —
// and has no opinion about whether an assignment is actually reachable:
// catalog lookup, refresh, credentials and family are the resolution
// chain's job, not this package's.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/BurntSushi/toml"
)

// EnvVar is the environment variable that overrides platform discovery of
// the config file's location — an absolute path, consulted before
// os.UserConfigDir (specs 06). It is also what makes the file's location
// testable without ever touching a developer's real profile.
const EnvVar = "EXTERNAL_REVIEWER_CONFIG"

// Tiers is the fixed vocabulary a [tiers.<name>] table may assign. A name
// outside this set is reported and ignored, never silently accepted as a
// new tier (specs 05) — there is no mechanism to extend it from the file.
var Tiers = []string{"light", "standard", "heavy"}

var tierSet = func() map[string]bool {
	m := make(map[string]bool, len(Tiers))
	for _, t := range Tiers {
		m[t] = true
	}
	return m
}()

// Assignment is one tier's provider and model, exactly as the file spelled
// them: kern-link's own ProviderId and Model.ID verbatim, with no aliasing
// or translation layer (specs 05).
type Assignment struct {
	Provider string
	Model    string
}

// Config is what one read of the tier assignment file produced.
type Config struct {
	// Path is the file this run looked at, whether or not anything was
	// there — callers print it rather than re-deriving it.
	Path string
	// Assignments holds one entry per tier the file assigned completely.
	// A tier that was absent from the file, out of vocabulary, or missing
	// a required key has no entry here — never a half-filled one.
	Assignments map[string]Assignment
	// Warnings are every unrecognised key, out-of-vocabulary tier name
	// and incomplete assignment found while decoding, each in the wording
	// this package fixes. An absent file produces none.
	Warnings []string
}

// document is the shape BurntSushi/toml decodes into. Its two fields being
// the only recognised keys is what makes MetaData.Undecoded() report
// everything else — an inherited family key, a typo'd "models", or
// anything invented later — without this package having to enumerate them.
type document struct {
	Tiers map[string]tierTable `toml:"tiers"`
}

type tierTable struct {
	Provider string `toml:"provider"`
	Model    string `toml:"model"`
}

// Locate returns the path Load reads from, without touching the
// filesystem: EnvVar when set, else os.UserConfigDir() joined with
// "external-reviewer" and "config.toml" — the one stdlib call that
// implements the %APPDATA% / $XDG_CONFIG_HOME split (specs 06).
func Locate() (string, error) {
	if p := os.Getenv(EnvVar); p != "" {
		return p, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("locating the user config directory: %w", err)
	}
	return filepath.Join(dir, "external-reviewer", "config.toml"), nil
}

// Load locates the tier assignment file and decodes it. See LoadFile for
// what an absent file, a malformed file and every warning mean.
func Load() (*Config, error) {
	path, err := Locate()
	if err != nil {
		return nil, err
	}
	return LoadFile(path)
}

// LoadFile decodes the tier assignment file at path directly, bypassing
// platform discovery — the seam every test in this package reads through,
// so no test ever opens a developer's real profile.
//
// An absent file is not an error: it is the default state before a human
// has set anything up, and the result reports it as zero assignments plus
// the path that was looked at. Malformed TOML is an error, because it names
// a file a human wrote by hand and got wrong — something only a human can
// fix.
func LoadFile(path string) (*Config, error) {
	cfg := &Config{Path: path, Assignments: map[string]Assignment{}}

	var doc document
	meta, err := toml.DecodeFile(path, &doc)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return cfg, nil
		}
		// path is an OS path, not a wire path — printed as the OS renders
		// it (CONVENTIONS § Paths and platforms), so %s rather than %q,
		// which would escape a Windows path's backslashes.
		return nil, fmt.Errorf("parsing config file %s: %w", path, err)
	}

	// A tier with any undecoded key is already flagged by that warning; a
	// human fixing the typo supplies the value, so a second "missing
	// required key" warning for the same table would be redundant noise
	// rather than a second, independent fact.
	undecodedTiers := make(map[string]bool)
	for _, key := range meta.Undecoded() {
		cfg.Warnings = append(cfg.Warnings, undecodedWarning(key))
		if len(key) >= 2 && key[0] == "tiers" {
			undecodedTiers[key[1]] = true
		}
	}
	for name, table := range doc.Tiers {
		warnings, assignment, ok := resolveTier(name, table, undecodedTiers[name])
		cfg.Warnings = append(cfg.Warnings, warnings...)
		if ok {
			cfg.Assignments[name] = assignment
		}
	}

	// Map iteration order is random; a fixed order makes the warning list
	// (and any test or "tiers" rendering built on it) reproducible.
	sort.Strings(cfg.Warnings)
	return cfg, nil
}

// resolveTier turns one decoded [tiers.<name>] table into either an
// assignment or the warnings explaining why there is none — never both. A
// name outside the fixed vocabulary is warned once and the table's
// contents are ignored entirely. A table missing provider, model, or both
// is warned once per missing key and reported as no assignment rather than
// a half-filled one — unless hasUndecodedKey is set, meaning an
// unrecognised key in the same table already explains it (a likely typo of
// the very key that is missing); a second, redundant warning would only
// bury the one that names the fix.
func resolveTier(name string, table tierTable, hasUndecodedKey bool) (warnings []string, assignment Assignment, ok bool) {
	if !tierSet[name] {
		return []string{fmt.Sprintf("config: unrecognised tier %q", name)}, Assignment{}, false
	}
	if table.Provider == "" || table.Model == "" {
		if !hasUndecodedKey {
			if table.Provider == "" {
				warnings = append(warnings, fmt.Sprintf("config: [tiers.%s] missing required key %q", name, "provider"))
			}
			if table.Model == "" {
				warnings = append(warnings, fmt.Sprintf("config: [tiers.%s] missing required key %q", name, "model"))
			}
		}
		return warnings, Assignment{}, false
	}
	return nil, Assignment(table), true
}

// undecodedWarning renders one key BurntSushi/toml could not place into
// document, in the exact wording specs 05 and specs 12 fix:
// `config: unrecognised key "models" in [tiers.standard]`. key's last
// element is the leaf name that was not recognised; everything before it
// names the enclosing table.
func undecodedWarning(key toml.Key) string {
	leaf := key[len(key)-1]
	section := key[:len(key)-1]
	return fmt.Sprintf("config: unrecognised key %q in [%s]", leaf, joinDotted(section))
}

func joinDotted(parts []string) string {
	out := parts[0]
	for _, p := range parts[1:] {
		out += "." + p
	}
	return out
}
