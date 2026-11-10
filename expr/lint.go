package expr

import (
	"fmt"
	"regexp"
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

	if p.env.patternLength > 0 {
		patterns := &patternCollector{}
		ast.Walk(&tree.Node, patterns)

		for _, pattern := range patterns.found {
			if err := p.lintPattern(pattern); err != nil {
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
	all := &predicateCollector{}
	ast.Walk(&tree.Node, all)

	depths := map[*ast.PredicateNode]uint{}

	var depth func(predicate *ast.PredicateNode) uint
	depth = func(predicate *ast.PredicateNode) uint {
		if d, ok := depths[predicate]; ok {
			return d
		}

		inner := &predicateCollector{}
		ast.Walk(&predicate.Node, inner)

		var deepest uint
		for _, nested := range inner.found {
			deepest = max(deepest, depth(nested))
		}

		depths[predicate] = deepest + 1

		return deepest + 1
	}

	for _, predicate := range all.found {
		if d := depth(predicate); d > p.env.nesting {
			return p.source.positionError(&file.Error{
				Location: predicate.Location(),
				Message:  fmt.Sprintf("list functions nest %d deep, expected at most %d", d, p.env.nesting),
			}, CodeNestingTooDeep)
		}
	}

	return nil
}

// lintPattern requires the pattern of a matches operator to be a valid
// string literal within the length limit.
func (p *Program) lintPattern(pattern ast.Node) error {
	literal, ok := pattern.(*ast.StringNode)

	var message string

	switch {
	case !ok:
		message = "pattern is not a string literal, expected a quoted pattern"
	case uint(utf8.RuneCountInString(literal.Value)) > p.env.patternLength:
		message = fmt.Sprintf("pattern is %d characters, expected at most %d", utf8.RuneCountInString(literal.Value), p.env.patternLength)
	default:
		if _, err := regexp.Compile(literal.Value); err != nil {
			message = "pattern is invalid: " + err.Error()
		}
	}

	if message == "" {
		return nil
	}

	return p.source.positionError(&file.Error{
		Location: pattern.Location(),
		Message:  message,
	}, CodePatternInvalid)
}

type predicateCollector struct {
	found []*ast.PredicateNode
}

func (c *predicateCollector) Visit(node *ast.Node) {
	if predicate, ok := (*node).(*ast.PredicateNode); ok {
		c.found = append(c.found, predicate)
	}
}

// patternCollector records the right operand of every matches operator.
type patternCollector struct {
	found []ast.Node
}

func (c *patternCollector) Visit(node *ast.Node) {
	if binary, ok := (*node).(*ast.BinaryNode); ok && binary.Operator == "matches" {
		c.found = append(c.found, binary.Right)
	}
}
