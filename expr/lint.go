package expr

import (
	"fmt"
	"unicode/utf8"

	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/file"
	"github.com/expr-lang/expr/parser"
)

// lint checks tree against the limits of the program's Env that upstream
// expr doesn't check itself.
func (p *Program) lint(tree *parser.Tree) error {
	if p.env.nesting > 0 {
		if err := p.lintNesting(tree); err != nil {
			return err
		}
	}

	if p.env.pattern > 0 {
		for _, binary := range nodesOf[*ast.BinaryNode](&tree.Node) {
			if binary.Operator != "matches" {
				continue
			}

			if err := p.lintPattern(binary.Right); err != nil {
				return err
			}
		}
	}

	return nil
}

// lintNesting rejects a predicate whose list functions nest deeper than the
// limit. A predicate's depth is one more than the deepest predicate inside
// it.
func (p *Program) lintNesting(tree *parser.Tree) error {
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
		if d := depth(predicate); d > p.env.nesting {
			return p.source.positionError(&file.Error{
				Location: predicate.Location(),
				Message:  fmt.Sprintf("list functions nest %d deep, expected at most %d", d, p.env.nesting),
			}, CodeNestingTooDeep)
		}
	}

	return nil
}

// lintPattern requires the pattern of a matches operator to be a string
// literal within the length limit. Upstream expr checks that a literal
// pattern is a valid regular expression.
func (p *Program) lintPattern(pattern ast.Node) error {
	literal, ok := pattern.(*ast.StringNode)

	var message string

	switch {
	case !ok:
		message = "pattern is not a string literal, expected a quoted pattern"
	case uint(utf8.RuneCountInString(literal.Value)) > p.env.pattern:
		message = fmt.Sprintf("pattern is %d characters, expected at most %d", utf8.RuneCountInString(literal.Value), p.env.pattern)
	default:
		return nil
	}

	return p.source.positionError(&file.Error{
		Location: pattern.Location(),
		Message:  message,
	}, CodePatternInvalid)
}
