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
// # Envs
//
// An [Env] says what names exist and how they are typed:
//
//   - [Typed] is for label sets from [Nest]. Any name is allowed,
//     a.b is nil when a is missing or isn't a map, and a program is typed
//     against each label set it runs with.
//   - [Untyped] allows any name and types nothing until the program runs.
//
// # Usage
//
// Compile once and run as often as needed; a [Program] is safe for
// concurrent use:
//
//	program, err := expr.Compile(`env = "prod" AND region IN ["eu"]`, expr.Typed())
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
