package expr_test

import (
	"fmt"

	"github.com/SigNoz/baseplate.go/errors"
	"github.com/SigNoz/baseplate.go/expr"
	"github.com/SigNoz/baseplate.go/expr/ast"
)

func Example() {
	program, err := expr.Compile(`env = "prod" AND service.name IN ["api", "web"]`, expr.TypedDynamic())
	if err != nil {
		panic(err)
	}

	env, _ := expr.Nest(map[string]string{"env": "prod", "service.name": "api"})

	matched, err := program.Bool(env)
	if err != nil {
		panic(err)
	}

	_, _ = fmt.Println(matched)
	// Output:
	// true
}

func ExampleCompile_error() {
	_, err := expr.Compile(`env = "prod" AND "5" > 3`, expr.TypedDynamic())
	if err != nil {
		_, _ = fmt.Println(errors.CodeOf(err))
		_, _ = fmt.Println(errors.MessageOf(err))
	}

	program, err := expr.Compile(`env = "prod" AND`, expr.Untyped())
	_, _ = fmt.Println(program == nil, errors.CodeOf(err))

	if position, ok := errors.AsType[*expr.PositionError](err); ok {
		_, _ = fmt.Println(position.Line, position.Column)
	}

	_, _ = fmt.Println(err)
	// Output:
	// type_invalid
	// invalid operation: > (mismatched types string and int)
	// true syntax_invalid
	// 1 16
	// unexpected token EOF: (1:16)
	//  | env = "prod" AND
	//  | ...............^
}

func ExampleTypedDynamic() {
	program, err := expr.Compile(`cluster.name = "alpha"`, expr.TypedDynamic())
	if err != nil {
		panic(err)
	}

	for _, flat := range []map[string]string{
		{"cluster.name": "alpha"},
		{"cluster": "alpha"},
		{"env": "prod"},
	} {
		env, _ := expr.Nest(flat)

		matched, err := program.Bool(env)
		_, _ = fmt.Println(matched, err)
	}
	// Output:
	// true <nil>
	// false <nil>
	// false <nil>
}

func ExampleUntyped() {
	program, err := expr.Compile(`attributes["level"] = "error" AND body CONTAINS "timeout"`, expr.Untyped())
	if err != nil {
		panic(err)
	}

	matched, err := program.Bool(map[string]any{
		"attributes": map[string]any{"level": "error"},
		"body":       "upstream timeout after 30s",
	})
	_, _ = fmt.Println(matched, err)

	missing, err := program.Bool(map[string]any{"attributes": map[string]any{}})
	_, _ = fmt.Println(missing, err)
	// Output:
	// true <nil>
	// false <nil>
}

func ExampleCompileValue() {
	program, err := expr.CompileValue(`lower(attributes["level"]) + "-" + string(severity_number)`, expr.Untyped())
	if err != nil {
		panic(err)
	}

	value, err := program.Run(map[string]any{
		"attributes":      map[string]any{"level": "ERROR"},
		"severity_number": 17,
	})
	_, _ = fmt.Println(value, err)
	// Output:
	// error-17 <nil>
}

func ExampleNest() {
	env, conflict := expr.Nest(map[string]string{"k8s.pod.name": "api-7f9c", "k8s.namespace": "alpha"})
	_, _ = fmt.Println(env, conflict)

	_, conflict = expr.Nest(map[string]string{"host": "web-1", "host.name": "web-1.internal"})
	_, _ = fmt.Println(conflict)
	// Output:
	// map[k8s:map[namespace:alpha pod:map[name:api-7f9c]]] false
	// true
}

// bodyField reads a member of body from a parsed copy of it, body_map.
type bodyField struct{}

func (bodyField) Visit(node *ast.Node) {
	member, ok := (*node).(*ast.MemberNode)
	if !ok {
		return
	}

	if root, ok := member.Node.(*ast.IdentifierNode); ok && root.Value == "body" {
		root.Value = "body_map"
	}
}

func ExampleEnv_Patch() {
	program, err := expr.Compile(`body.level = "error"`, expr.Untyped().Patch(bodyField{}))
	if err != nil {
		panic(err)
	}

	matched, err := program.Bool(map[string]any{
		"body":     `{"level": "error"}`,
		"body_map": map[string]any{"level": "error"},
	})
	_, _ = fmt.Println(matched, err)
	// Output:
	// true <nil>
}
