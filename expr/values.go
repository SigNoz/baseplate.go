package expr

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"unicode/utf8"

	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/parser"
)

// maxValueLength caps the characters of a value that
// [Program.ReferenceValues] returns.
const maxValueLength = 256

var identifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// References returns the variables the expression reads, sorted, as paths
// such as tls.version or headers["content-type"]. A key that is a plain
// name is written with a dot, so headers["server"] gives headers.server. A
// path stops where the expression stops naming members by string, so
// json.items[0].ok gives json.items.
func (p *Program) References() []string {
	return slices.Clone(p.references)
}

// ReferenceValues runs each of [Program.References] with vars and returns
// its value as text, at most 256 characters, to show what an expression
// saw. Strings are as they are, nil is "nil", lists, maps and structs are
// JSON, and other values are formatted as Go formats them. A reference
// that fails to compile or run is left out.
func (p *Program) ReferenceValues(vars any) map[string]string {
	p.values.once.Do(func() {
		p.values.programs = map[string]*Program{}

		for _, reference := range p.references {
			if program, err := CompileValue(reference, p.env); err == nil {
				p.values.programs[reference] = program
			}
		}
	})

	values := map[string]string{}

	for reference, program := range p.values.programs {
		if value, err := program.Run(vars); err == nil {
			values[reference] = stringify(value)
		}
	}

	return values
}

func stringify(value any) string {
	var text string

	switch v := value.(type) {
	case nil:
		text = "nil"
	case string:
		text = v
	default:
		text = fmt.Sprint(v)

		kind := reflect.TypeOf(v).Kind()
		if kind == reflect.Map || kind == reflect.Slice || (kind == reflect.Struct && reflect.TypeOf(v) != timeType) {
			if encoded, err := json.Marshal(v); err == nil {
				text = string(encoded)
			}
		}
	}

	if utf8.RuneCountInString(text) > maxValueLength {
		text = string([]rune(text)[:maxValueLength])
	}

	return text
}

// references returns the paths of the variables tree reads: names, and
// chains of string-keyed members on them, longest first in each chain.
func references(tree *parser.Tree) []string {
	collector := &referenceCollector{
		parents: map[ast.Node]bool{},
		skip:    map[ast.Node]bool{},
		lets:    map[string]bool{},
	}

	ast.Walk(&tree.Node, &referenceScope{collector: collector})
	ast.Walk(&tree.Node, collector)

	slices.Sort(collector.found)

	return slices.Compact(collector.found)
}

type referenceCollector struct {
	parents map[ast.Node]bool // Nodes another string-keyed member is read on.
	skip    map[ast.Node]bool // Called names.
	lets    map[string]bool
	found   []string
}

// referenceScope records what isn't the end of a reference: the nodes
// members are read on, called names and names bound by let.
type referenceScope struct {
	collector *referenceCollector
}

func (s *referenceScope) Visit(node *ast.Node) {
	switch n := (*node).(type) {
	case *ast.MemberNode:
		if _, ok := n.Property.(*ast.StringNode); ok && !n.Method {
			s.collector.parents[n.Node] = true
		}
	case *ast.CallNode:
		s.collector.skip[n.Callee] = true
	case *ast.VariableDeclaratorNode:
		s.collector.lets[n.Name] = true
	}
}

func (c *referenceCollector) Visit(node *ast.Node) {
	if c.parents[*node] || c.skip[*node] {
		return
	}

	if path, ok := c.path(*node); ok {
		c.found = append(c.found, path)
	}
}

// path returns node as a reference, when it is a name or a chain of
// string-keyed members on one.
func (c *referenceCollector) path(node ast.Node) (string, bool) {
	switch n := node.(type) {
	case *ast.IdentifierNode:
		if c.lets[n.Value] || n.Value == "$env" {
			return "", false
		}

		return n.Value, true
	case *ast.MemberNode:
		key, ok := n.Property.(*ast.StringNode)
		if !ok || n.Method {
			return "", false
		}

		parent, ok := c.path(n.Node)
		if !ok {
			return "", false
		}

		if identifierPattern.MatchString(key.Value) && aliases[key.Value] == "" {
			return parent + "." + key.Value, true
		}

		return parent + "[" + strconv.Quote(key.Value) + "]", true
	}

	return "", false
}
