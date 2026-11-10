package expr

import (
	"slices"

	"github.com/expr-lang/expr/ast"
)

// Env is how an expression's names are typed, the patches its programs
// apply and the limits they keep to. The zero Env is [Untyped].
type Env struct {
	dynamic bool
	static  bool
	vars    any
	varsErr error
	patches []ast.Visitor
	nesting uint
	pattern uint
	nodes   uint
	memory  uint
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
	return Env{dynamic: true}
}

// TypedStatic returns an Env that types a program when it compiles, against
// vars: a struct, whose fields are named by their expr tag or else their Go
// name, or a map of typed zero values. Only the names vars declares exist,
// so an unknown name, a wrong type or a misspelled field is a compile error.
// An unknown var, field or function comes with the nearest declared one,
// if any, as a suggestion.
// Member access is nil-safe on map and any values, as [patcher.NilSafeTyped]
// makes it, and checked on struct fields. A nil or non-bool result of a
// [Compile]d program is an error. Run the program with a value of the same
// type as vars.
//
// The name of every var and field must be one identifier other than an
// alias such as AND, and a var's name must not be a builtin function, true,
// false, nil or $env. [Compile] returns a
// [github.com/SigNoz/baseplate.go/errors.TypeInvalidInput] error with
// [CodeVarsInvalid] when vars breaks these rules, or isn't a struct or a
// map[string]any.
//
// With vars of type struct{ Value string `expr:"value"` }:
//
//	value > 80           compile error (CodeTypeInvalid): string and int
//	cluster.name = "c1"  compile error (CodeNameUnknown): unknown name
//	plan                 compile error (CodeNameUnknown): unknown name
func TypedStatic(vars any) Env {
	return Env{
		static:  true,
		vars:    vars,
		varsErr: checkVars(vars),
	}
}

// LimitNesting returns a copy of e whose programs reject a list function,
// such as all or filter, nested more than n deep in another's predicate, so
// a program's run time stays linear in the size of its input. Zero, the
// default, sets no limit. [Compile] returns a
// [github.com/SigNoz/baseplate.go/errors.TypeInvalidInput] error with
// [CodeNestingTooDeep] for a program over it.
func (e Env) LimitNesting(n uint) Env {
	e.nesting = n

	return e
}

// LimitPattern returns a copy of e whose programs require the pattern of
// every REGEXP (or matches) to be a string literal of at most n characters,
// so patterns compile once. Expr itself rejects an invalid literal pattern,
// with [CodeTypeInvalid]. Zero, the default, sets no limit.
// [Compile] returns a [github.com/SigNoz/baseplate.go/errors.TypeInvalidInput]
// error with [CodePatternInvalid] for a pattern that breaks it.
func (e Env) LimitPattern(n uint) Env {
	e.pattern = n

	return e
}

// LimitNodes returns a copy of e whose programs have at most n nodes. Zero,
// the default, keeps expr's own limit. [Compile] returns a
// [github.com/SigNoz/baseplate.go/errors.TypeInvalidInput] error with
// [CodeNodesExceeded] for a program over it.
func (e Env) LimitNodes(n uint) Env {
	e.nodes = n

	return e
}

// LimitMemory returns a copy of e whose programs allocate at most n units
// of expr's memory budget in a run. Zero, the default, keeps expr's own
// budget. [Program.Run] and [Program.Bool] return a
// [github.com/SigNoz/baseplate.go/errors.TypeInvalidInput] error with
// [CodeMemoryExceeded] for a run over it.
func (e Env) LimitMemory(n uint) Env {
	e.memory = n

	return e
}

// Patch returns a copy of e whose programs apply v when they compile. A
// patch rewrites the syntax tree in place, as those in
// [github.com/SigNoz/baseplate.go/expr/patcher] do. Patches run in the order
// they were added, and before the nil-safe patch of a [TypedDynamic]
// program, so they see member access as written.
//
// An [Untyped] program runs each patch once, in [Compile]. A [TypedDynamic]
// program runs it again for each new set of top-level variable types, so a
// patch that keeps state sees every tree it rewrites. Since a patch can
// make the program read any name, a TypedDynamic program with patches
// compiles again when the type of any of its variables changes, not only
// of the names it was written with.
func (e Env) Patch(v ast.Visitor) Env {
	e.patches = append(slices.Clone(e.patches), v)

	return e
}
