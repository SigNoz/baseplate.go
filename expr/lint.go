package expr

import (
	"fmt"
	"unicode/utf8"

	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/file"
	"github.com/expr-lang/expr/parser"
)

// lint checks tree against the limits of env that upstream expr doesn't
// check itself, reporting errors on s.
func lint(tree *parser.Tree, env Env, s *source) error {
	if env.nesting > 0 {
		if err := lintNesting(tree, env.nesting, s); err != nil {
			return err
		}
	}

	if env.pattern > 0 {
		for _, binary := range nodesOf[*ast.BinaryNode](&tree.Node) {
			if binary.Operator != "matches" {
				continue
			}

			if err := lintPattern(binary.Right, env.pattern, s); err != nil {
				return err
			}
		}
	}

	return nil
}

// lintNesting rejects a predicate whose list functions nest deeper than the
// limit. A predicate's depth is one more than the deepest predicate inside
// it.
func lintNesting(tree *parser.Tree, limit uint, s *source) error {
	depths := map[*ast.PredicateNode]uint{}

	var depth func(predicate *ast.PredicateNode) uint
	depth = func(predicate *ast.PredicateNode) uint {
		if d, ok := depths[predicate]; ok {
			return d
		}

		var deepest uint
		for _, nested := range nodesOf[*ast.PredicateNode](&predicate.Node) {
			deepest = max(deepest, depth(nested))
		}

		depths[predicate] = deepest + 1

		return deepest + 1
	}

	for _, predicate := range nodesOf[*ast.PredicateNode](&tree.Node) {
		if d := depth(predicate); d > limit {
			return s.positionError(&file.Error{
				Location: predicate.Location(),
				Message:  fmt.Sprintf("list functions nest %d deep, expected at most %d", d, limit),
			}, CodeNestingTooDeep)
		}
	}

	return nil
}

// lintPattern requires the pattern of a matches operator to be a string
// literal within the length limit. Upstream expr checks that a literal
// pattern is a valid regular expression.
func lintPattern(pattern ast.Node, limit uint, s *source) error {
	literal, ok := pattern.(*ast.StringNode)

	var message string

	switch {
	case !ok:
		message = "pattern is not a string literal, expected a quoted pattern"
	case uint(utf8.RuneCountInString(literal.Value)) > limit:
		message = fmt.Sprintf("pattern is %d characters, expected at most %d", utf8.RuneCountInString(literal.Value), limit)
	default:
		return nil
	}

	return s.positionError(&file.Error{
		Location: pattern.Location(),
		Message:  message,
	}, CodePatternInvalid)
}
