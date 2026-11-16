package expr

import (
	"github.com/expr-lang/expr/ast"
)

// visitor turns a function into an [ast.Visitor].
type visitor func(node *ast.Node)

func (v visitor) Visit(node *ast.Node) {
	v(node)
}

// visit calls fn with every node of the tree at root, children before their
// parent.
func visit(root *ast.Node, fn func(node *ast.Node)) {
	ast.Walk(root, visitor(fn))
}
