package patcher

import (
	"reflect"

	upstream "github.com/expr-lang/expr/ast"

	"github.com/SigNoz/baseplate.go/expr/ast"
)

const nilSafeTypedMemberFunc = "__nilSafeTypedMember"

// NilSafeTyped rewrites a string-keyed member access a.b, including a["b"],
// into a call that returns nil when a has no such key, but only where a is
// a map or of unknown type, such as an any value. Member access on a struct
// stays as written, so its fields keep their types and a misspelled field
// stays a compile error. Method calls stay as written too.
//
// The calls go to the function [NilSafeTyped.Function] returns, which
// [github.com/SigNoz/baseplate.go/expr] registers with the patch.
type NilSafeTyped struct{}

var _ ast.Visitor = NilSafeTyped{}

// Visit rewrites node when it is a string-keyed member access of a map or of
// a value of unknown type.
func (NilSafeTyped) Visit(node *ast.Node) {
	member, ok := (*node).(*ast.MemberNode)
	if !ok || member.Method {
		return
	}

	if _, ok := member.Property.(*ast.StringNode); !ok {
		return
	}

	if t := member.Node.Type(); t != nil && t.Kind() != reflect.Map && t.Kind() != reflect.Interface {
		return
	}

	upstream.Patch(node, &ast.CallNode{
		Callee:    &ast.IdentifierNode{Value: nilSafeTypedMemberFunc},
		Arguments: []ast.Node{member.Node, member.Property},
	})
}

// Function returns the name and the body of the function the rewritten
// calls go to. It returns a[key] when a is a map with string keys that has
// key, and nil otherwise.
func (NilSafeTyped) Function() (string, func(params ...any) (any, error)) {
	return nilSafeTypedMemberFunc, nilSafeTypedMember
}

func nilSafeTypedMember(params ...any) (any, error) {
	m := reflect.ValueOf(params[0])
	key, _ := params[1].(string)

	if m.Kind() != reflect.Map || m.Type().Key().Kind() != reflect.String {
		return nil, nil
	}

	value := m.MapIndex(reflect.ValueOf(key).Convert(m.Type().Key()))
	if !value.IsValid() {
		return nil, nil
	}

	return value.Interface(), nil
}
