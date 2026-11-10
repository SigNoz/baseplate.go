package patcher

import (
	"github.com/expr-lang/expr/ast"
)

const nilSafeMemberFunc = "__nilSafeMember"

// NilSafe rewrites every string-keyed member access a.b, including a["b"],
// into a call that returns nil unless a is a map[string]any. A name that is
// missing, or a prefix that holds a plain value, then reads as nil instead
// of failing the run.
//
// The calls go to the function [NilSafe.Function] returns, which
// [github.com/SigNoz/baseplate.go/expr] registers with the patch.
type NilSafe struct{}

var _ ast.Visitor = NilSafe{}

// Visit rewrites node when it is a string-keyed member access.
func (NilSafe) Visit(node *ast.Node) {
	member, ok := (*node).(*ast.MemberNode)
	if !ok {
		return
	}

	if _, ok := member.Property.(*ast.StringNode); !ok {
		return
	}

	ast.Patch(node,
		&ast.CallNode{
			Callee:    &ast.IdentifierNode{Value: nilSafeMemberFunc},
			Arguments: []ast.Node{member.Node, member.Property},
		},
	)
}

// Function returns the name and the body of the function the rewritten
// calls go to. It returns a[key] when a is a map[string]any, and nil
// otherwise.
func (NilSafe) Function() (string, func(params ...any) (any, error)) {
	return nilSafeMemberFunc, nilSafeMember
}

func nilSafeMember(params ...any) (any, error) {
	m, _ := params[0].(map[string]any)
	key, _ := params[1].(string)

	return m[key], nil
}
