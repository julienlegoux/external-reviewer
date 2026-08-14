package tools_test

import (
	"context"
	"slices"
	"testing"

	"github.com/julienlegoux/kern-link/ai"

	"github.com/julienlegoux/external-reviewer/internal/tools"
)

// fixtureProvider and fixtureModel are the provider and model the scripted
// registries in this package's end-to-end tool tests serve. They live here
// rather than in internal/reviewer, which no longer exports a default
// reviewer: the CLI resolves a tier, and the binary names no provider outside
// the family classifier's data table (issue 05 of Epic 3).
const (
	fixtureProvider = "openai-codex"
	fixtureModel    = "gpt-5.5"
)

// stub is a tool with no behaviour, for the registry's own contract: what it
// declares, and what it dispatches to.
func stub(name string) tools.Tool {
	return tools.Tool{
		Declaration: ai.Tool{Name: name, Description: name, Parameters: ai.JSONSchema(`{"type":"object"}`)},
		Handler: func(context.Context, map[string]any) tools.Result {
			return tools.Result{Text: name + " answered\n"}
		},
	}
}

func TestRegistry_DeclaresToolsInTheOrderTheyWereGiven(t *testing.T) {
	registry := tools.NewRegistry(stub("list"), stub("read_file"), stub("search"))

	names := make([]string, 0, 3)
	for _, declaration := range registry.Declarations() {
		names = append(names, declaration.Name)
	}

	if want := []string{"list", "read_file", "search"}; !slices.Equal(names, want) {
		t.Errorf("the registry declared %q, want %q", names, want)
	}
	if got := registry.Names(); !slices.Equal(got, []string{"list", "read_file", "search"}) {
		t.Errorf("Names returned %q", got)
	}
}

func TestRegistry_LooksUpAToolByTheNameTheModelUses(t *testing.T) {
	registry := tools.NewRegistry(stub("list"))

	tool, found := registry.Lookup("list")
	if !found {
		t.Fatal("the registry did not find the tool it declares")
	}
	if got := tool.Handler(t.Context(), nil); got.Text != "list answered\n" {
		t.Errorf("Lookup returned a tool whose handler answered %q", got.Text)
	}

	if _, found := registry.Lookup("delete_everything"); found {
		t.Error("the registry found a tool it never declared")
	}
}
