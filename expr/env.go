package expr

import (
	"slices"

	"github.com/SigNoz/baseplate.go/expr/ast"
)

// Env is how an expression's names are typed, and the patches its programs
// apply. The zero Env is [Untyped].
type Env struct {
	typed   bool
	patches []ast.Visitor
}

// Untyped returns an Env in which any name is allowed and nothing is typed
// until the program runs. Member access isn't nil-safe: a.b fails when a is
// missing. A [Compile]d program returns false for a nil result.
//
// With the variables {"value": "92"}:
//
//	value > 80           run error (CodeRunFailed): string > int
//	cluster.name = "c1"  run error (CodeRunFailed): cannot fetch name
//	plan                 false
func Untyped() Env {
	return Env{}
}

// TypedDynamic returns an Env that types a program when it runs, against
// each set of variables it runs with, such as the nested maps [Nest]
// returns. Any name is allowed, and member access is nil-safe, as
// [patcher.NilSafe] makes it. A program is typed against the top-level
// types of each set of variables and compiled once for each set of those
// types. A
// nil or non-bool result of a [Compile]d program is an error.
//
// With the variables {"value": "92"}:
//
//	value > 80           type error (CodeTypeInvalid): string and int
//	cluster.name = "c1"  false
//	plan                 error (CodeResultNotBool): the result is nil
func TypedDynamic() Env {
	return Env{typed: true}
}

// Patch returns a copy of e whose programs apply v when they compile. A
// patch rewrites the syntax tree in place, as those in
// [github.com/SigNoz/baseplate.go/expr/patcher] do. Patches run in the order
// they were added, and before the nil-safe patch of a [TypedDynamic]
// program, so they see member access as written.
//
// An [Untyped] program runs each patch once, in [Compile]. A [TypedDynamic]
// program runs it again for each new shape of variables, so a patch that
// keeps state sees every tree it rewrites. Since a patch can make the
// program read any name, a TypedDynamic program with patches tells shapes
// apart by the types of all its variables, not only the names it was
// written with.
func (e Env) Patch(v ast.Visitor) Env {
	e.patches = append(slices.Clone(e.patches), v)

	return e
}
