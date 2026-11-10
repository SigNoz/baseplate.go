// Package patcher holds the syntax tree patches
// [github.com/SigNoz/baseplate.go/expr] applies when it compiles an
// expression. Each patch is a
// [github.com/SigNoz/baseplate.go/expr/ast.Visitor] that rewrites nodes in
// place, as the patches in [github.com/expr-lang/expr/patcher] do.
//
// [NilSafe] makes member access return nil instead of failing when the
// parent is missing or isn't a map. A
// [github.com/SigNoz/baseplate.go/expr.TypedDynamic] program applies it.
package patcher
