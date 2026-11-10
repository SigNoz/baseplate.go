// Package expr compiles and runs [github.com/expr-lang/expr] expressions,
// with SQL-style spellings for its operators.
//
// # Language
//
// The language is expr's, with these spellings added:
//
//	env = "prod" AND region IN ["eu", "us"]
//	NOT (service.name CONTAINS "test") OR host REGEXP "^web-"
//
// AND, OR, NOT, IN and CONTAINS are expr's and, or, not, in and contains,
// REGEXP and regexp are matches, and = is ==. Every expr operator and
// builtin works too. The aliases are all upper case or all lower case, so
// "And" is an error, and NOT binds tighter than =, so NOT (a = b) needs its
// parentheses. Text in quotes and member names, as in attributes.regexp, are
// never respelled, and neither is the = of let x = 1.
//
// # Untyped and TypedDynamic
//
// An [Env] says when names get their types. [Untyped] never types them.
// [TypedDynamic] types them when the program runs, from the variables it
// runs with. Both allow any name, and when the types fit, as in
// value = "92", both give the same result. They differ in when a type
// error shows and in how a missing value reads. Run with the variables
// {"value": "92"}:
//
//	expression           Untyped                   TypedDynamic
//	value > 80           run error: string > int   type error: string and int
//	cluster.name = "c1"  run error: no cluster     false
//	plan                 false                     error: the result is nil
//
// An Untyped program is compiled once and knows no types until it runs,
// so a mistake shows as [CodeRunFailed], a.b fails when a is missing, and a
// nil result is false.
//
// A TypedDynamic program is typed against the top-level types of the
// variables it runs with, compiled once for each such shape, so a mistake
// shows as [CodeTypeInvalid]. Member access is nil-safe: a.b is nil when a
// is missing or isn't a map. A nil result is an error. [Nest] turns flat
// dotted keys into the nested maps a TypedDynamic program reads.
//
// # Usage
//
// Compile once and run as often as needed; a [Program] is safe for
// concurrent use:
//
//	program, err := expr.Compile(`env = "prod" AND region IN ["eu"]`, expr.TypedDynamic())
//	if err != nil {
//		return err
//	}
//
//	env, _ := expr.Nest(flat)
//	matched, err := program.Bool(env)
//
// [Compile] is for expressions whose result is a bool, and [CompileValue]
// for any value.
//
// # Errors
//
// Every error is a [github.com/SigNoz/baseplate.go/errors.TypeInvalidInput]
// error with one of the package's codes, such as [CodeSyntaxInvalid] or
// [CodeTypeInvalid], and a one-line message. An error at a place in the
// expression wraps an [*PositionError] with its offsets, line and column in the
// text as written:
//
//	if position, ok := errors.AsType[*expr.PositionError](err); ok {
//		fmt.Println(position.Line, position.Column)
//	}
//
// The error's text ends with the line and a caret under the error, as expr
// prints it.
package expr
