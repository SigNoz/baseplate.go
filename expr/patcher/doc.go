// Package patcher holds the syntax tree patches
// [github.com/SigNoz/baseplate.go/expr] applies when it compiles an
// expression. Each patch is a [github.com/expr-lang/expr/ast.Visitor] that
// rewrites nodes in place, as the patches in
// [github.com/expr-lang/expr/patcher] do.
//
// [NilSafe] makes member access return nil instead of failing when the
// parent is missing or isn't a map. A
// [github.com/SigNoz/baseplate.go/expr.TypedDynamic] program applies it.
//
// [NilSafeTyped] does the same only where the parent is a map or of unknown
// type, so struct fields keep their types. A
// [github.com/SigNoz/baseplate.go/expr.TypedStatic] program applies it.
package patcher
