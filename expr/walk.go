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

// nodesOf returns every node of type T in the tree at root, children before
// their parent.
func nodesOf[T ast.Node](root *ast.Node) []T {
	var found []T

	visit(root, func(node *ast.Node) {
		if n, ok := (*node).(T); ok {
			found = append(found, n)
		}
	})

	return found
}
