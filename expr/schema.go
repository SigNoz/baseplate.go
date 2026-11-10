package expr

import (
	"github.com/expr-lang/expr/docgen"
)

// Schema is what an Env's programs can read, for a client such as an
// editor or a docs page: the declared vars of a [TypedStatic] Env, as expr's
// docgen describes them, and the limits programs compile with.
type Schema struct {
	Vars   *docgen.Context `json:"vars,omitempty"` // Nil unless the Env is TypedStatic; the others allow any name.
	Limits Limits          `json:"limits"`
}

// Limits are the limits a program is compiled with. Zero is unset. The
// memory limit is left out, as it only applies when a program runs.
type Limits struct {
	Nesting uint `json:"nesting,omitempty"`
	Pattern uint `json:"pattern,omitempty"`
	Nodes   uint `json:"nodes,omitempty"`
}

// Schema returns the schema of e's programs. It returns the errors
// [TypedStatic] reports for vars that break its rules.
func (e Env) Schema() (Schema, error) {
	schema := Schema{
		Limits: Limits{
			Nesting: e.nesting,
			Pattern: e.pattern,
			Nodes:   e.nodes,
		},
	}

	if !e.static {
		return schema, nil
	}

	if e.varsErr != nil {
		return Schema{}, e.varsErr
	}

	schema.Vars = docgen.CreateDoc(e.vars)

	return schema, nil
}
