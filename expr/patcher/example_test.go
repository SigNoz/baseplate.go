package patcher_test

import (
	"fmt"

	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/parser"

	"github.com/SigNoz/baseplate.go/expr/patcher"
)

func ExampleNilSafe() {
	tree, err := parser.Parse(`service.name == "checkout" && host["name"] != nil`)
	if err != nil {
		panic(err)
	}

	ast.Walk(&tree.Node, patcher.NilSafe{})

	_, _ = fmt.Println(tree.Node.String())
	// Output:
	// __nilSafeMember(service, "name") == "checkout" && __nilSafeMember(host, "name") != nil
}
