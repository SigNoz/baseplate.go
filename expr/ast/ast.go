package ast

import (
	upstream "github.com/expr-lang/expr/ast"
)

// Node is any node of the syntax tree.
type Node = upstream.Node

// Visitor is a patch: Visit is called with every node of the tree, children
// before their parent, and may replace the node it is given.
type Visitor = upstream.Visitor

// MemberNode is a member access, a.b or a["b"].
type MemberNode = upstream.MemberNode

// IdentifierNode is a name, such as a variable or a function's name.
type IdentifierNode = upstream.IdentifierNode

// CallNode is a function call.
type CallNode = upstream.CallNode

// StringNode is a string literal.
type StringNode = upstream.StringNode
