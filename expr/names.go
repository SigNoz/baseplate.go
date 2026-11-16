package expr

import (
	"slices"
	"strings"

	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/file"
	"github.com/expr-lang/expr/parser"

	"github.com/SigNoz/baseplate.go/errors"
)

// nameError returns the checker's error about a name that the vars of a
// [TypedStatic] program don't declare, reported on s with [CodeNameUnknown],
// or [CodeFunctionUnknown] when the name is called. It returns nil for any
// other error.
func nameError(tree *parser.Tree, err error, s *source) error {
	checked, ok := errors.AsType[*file.Error](err)
	if !ok {
		return nil
	}

	switch {
	case strings.HasPrefix(checked.Message, "unknown name "):
		identifier, found := nodeAt[*ast.IdentifierNode](tree, checked.Location)
		if found && slices.ContainsFunc(nodesOf[*ast.CallNode](&tree.Node), func(call *ast.CallNode) bool { return call.Callee == ast.Node(identifier) }) {
			return s.positionError(err, CodeFunctionUnknown)
		}

		return s.positionError(err, CodeNameUnknown)
	case strings.Contains(checked.Message, " has no field "):
		return s.positionError(err, CodeNameUnknown)
	}

	return nil
}

// nodeAt returns the node of type T at location in tree.
func nodeAt[T ast.Node](tree *parser.Tree, location file.Location) (T, bool) {
	for _, node := range nodesOf[T](&tree.Node) {
		if node.Location() == location {
			return node, true
		}
	}

	var zero T

	return zero, false
}
