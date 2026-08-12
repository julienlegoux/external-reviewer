// Package tools owns the read-only tools the reviewer calls and the registry
// the loop declares them from.
//
// Each tool is one value: the declaration the model reads — name, description
// and JSON schema, all snake_case, because that half is a wire contract with
// the model rather than Go API (CONVENTIONS § Naming) — and the handler that
// answers a call. They live together so a description cannot drift from the
// implementation it describes (CONVENTIONS § Documentation & comments).
//
// A handler returns a Result, never an error. A failing tool is a value the
// loop feeds back as ToolResultMessage{IsError: true} so the reviewer can try
// another path; only a failing *turn* ends a run (CONVENTIONS § Error
// handling, specs 13).
package tools

import (
	"context"

	"github.com/julienlegoux/kern-link/ai"
)

// Result is one tool call's answer: the text the model sees, and whether that
// text is a refusal it should learn from rather than material to reason about.
type Result struct {
	Text    string
	IsError bool
}

// errorResult is the shape every refusal takes: what was wrong, in the same
// plain text a successful call returns, since the model reads both the same
// way.
func errorResult(message string) Result {
	return Result{Text: message, IsError: true}
}

// Handler answers one tool call. Arguments arrive already decoded from the
// model's JSON (ai.ToolCall.Arguments is a map[string]any), so no tool parses
// JSON of its own.
type Handler func(ctx context.Context, arguments map[string]any) Result

// Tool is a declaration and the handler that answers it.
type Tool struct {
	Declaration ai.Tool
	Handler     Handler
}

// Registry is the set of tools one run offers. The loop declares from it and
// dispatches through it, and the later tool issues append to the same list.
type Registry struct {
	ordered []Tool
	byName  map[string]Tool
}

// NewRegistry collects tools in the order they are declared to the model,
// which is the order they are given here.
func NewRegistry(tools ...Tool) *Registry {
	registry := &Registry{
		ordered: make([]Tool, 0, len(tools)),
		byName:  make(map[string]Tool, len(tools)),
	}
	for _, tool := range tools {
		if _, duplicate := registry.byName[tool.Declaration.Name]; duplicate {
			// Two tools under one name is a programming error, not a runtime
			// condition: the registry is built from a fixed list in code. The
			// first declaration wins rather than the process panicking,
			// because nothing panics across the CLI boundary (CONVENTIONS §
			// Error handling).
			continue
		}
		registry.byName[tool.Declaration.Name] = tool
		registry.ordered = append(registry.ordered, tool)
	}
	return registry
}

// Declarations returns what the model is told it may call, in declaration
// order.
func (r *Registry) Declarations() []ai.Tool {
	declarations := make([]ai.Tool, 0, len(r.ordered))
	for _, tool := range r.ordered {
		declarations = append(declarations, tool.Declaration)
	}
	return declarations
}

// Lookup finds the tool a model asked for by name. A miss is not an error
// here: the loop turns it into a tool-error result naming what does exist, so
// a model that invented a tool name can correct itself.
func (r *Registry) Lookup(name string) (Tool, bool) {
	tool, found := r.byName[name]
	return tool, found
}

// Names returns the declared tool names, for the message a dispatch miss
// produces.
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.ordered))
	for _, tool := range r.ordered {
		names = append(names, tool.Declaration.Name)
	}
	return names
}
