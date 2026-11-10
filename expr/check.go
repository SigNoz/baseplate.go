package expr

import (
	"reflect"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/builtin"
	"github.com/expr-lang/expr/file"
	"github.com/expr-lang/expr/parser"

	"github.com/SigNoz/baseplate.go/errors"
)

// variable is a declared variable or struct field: its name and Go type.
type variable struct {
	name string
	t    reflect.Type
}

// variables returns the top-level variables vars declares, in declaration
// order for a struct and sorted for a map.
func variables(vars any) []variable {
	t := reflect.TypeOf(vars)
	if t.Kind() == reflect.Struct {
		return fieldsOf(t)
	}

	value := reflect.ValueOf(vars)

	var declared []variable
	for _, key := range value.MapKeys() {
		declared = append(declared, variable{
			name: key.String(),
			t:    value.MapIndex(key).Elem().Type(),
		})
	}

	slices.SortFunc(declared, func(a, b variable) int {
		return strings.Compare(a.name, b.name)
	})

	return declared
}

// fieldsOf returns the fields expr reads on a struct type, with embedded
// structs' fields in place of the embedded struct.
func fieldsOf(t reflect.Type) []variable {
	var fields []variable

	for field := range t.Fields() {
		if !field.IsExported() {
			continue
		}

		if field.Anonymous && field.Type.Kind() == reflect.Struct {
			fields = append(fields, fieldsOf(field.Type)...)

			continue
		}

		if name, ok := fieldName(field); ok {
			fields = append(fields, variable{
				name: name,
				t:    field.Type,
			})
		}
	}

	return fields
}

// checkNames reports, in the schema's own terms, the first name in tree
// that the declared variables or the builtins don't have.
func (p *Program) checkNames(tree *parser.Tree) error {
	checker := &nameChecker{
		program: p,
		top:     variables(p.env.vars),
		callees: map[ast.Node]bool{},
		lets:    map[string]bool{},
	}

	ast.Walk(&tree.Node, &scopeCollector{checker: checker})
	ast.Walk(&tree.Node, checker)

	return checker.err
}

type nameChecker struct {
	program *Program
	top     []variable
	callees map[ast.Node]bool
	lets    map[string]bool
	err     error
}

// scopeCollector records the names that aren't variables: called names and
// names bound by let.
type scopeCollector struct {
	checker *nameChecker
}

func (c *scopeCollector) Visit(node *ast.Node) {
	switch n := (*node).(type) {
	case *ast.CallNode:
		c.checker.callees[n.Callee] = true
	case *ast.VariableDeclaratorNode:
		c.checker.lets[n.Name] = true
	}
}

func (c *nameChecker) Visit(node *ast.Node) {
	if c.err != nil {
		return
	}

	switch n := (*node).(type) {
	case *ast.CallNode:
		callee, ok := n.Callee.(*ast.IdentifierNode)
		if !ok || slices.ContainsFunc(c.top, func(v variable) bool { return v.name == callee.Value }) {
			return
		}

		message := "unknown function " + callee.Value

		suggestion := suggest(callee.Value, builtin.Names)
		if suggestion != "" {
			message += "; did you mean " + suggestion + "?"
		}

		c.fail(callee, message, CodeFunctionUnknown, suggestion)
	case *ast.IdentifierNode:
		if !c.callees[n] {
			c.resolve(n)
		}
	case *ast.MemberNode:
		c.resolve(n)
	}
}

// resolve returns the declared type and path of node, a name or a chain of
// string-keyed members on one. It stops, knowing nothing, at a map, an any
// value or anything that isn't such a chain, and records an error for a
// name or field that isn't declared.
func (c *nameChecker) resolve(node ast.Node) (reflect.Type, string, bool) {
	switch n := node.(type) {
	case *ast.IdentifierNode:
		if c.lets[n.Value] || n.Value == "$env" {
			return nil, "", false
		}

		if i := slices.IndexFunc(c.top, func(v variable) bool { return v.name == n.Value }); i >= 0 {
			return c.top[i].t, n.Value, true
		}

		c.unknownField(n, "", n.Value, c.top)
	case *ast.MemberNode:
		key, ok := n.Property.(*ast.StringNode)
		if !ok || n.Method {
			return nil, "", false
		}

		t, path, known := c.resolve(n.Node)
		if !known || t.Kind() != reflect.Struct || t == timeType {
			return nil, "", false
		}

		fields := fieldsOf(t)
		if i := slices.IndexFunc(fields, func(v variable) bool { return v.name == key.Value }); i >= 0 {
			return fields[i].t, join(path, key.Value), true
		}

		c.unknownField(key, path, key.Value, fields)
	}

	return nil, "", false
}

func (c *nameChecker) unknownField(node ast.Node, path, name string, siblings []variable) {
	if c.err != nil {
		return
	}

	names := make([]string, 0, len(siblings))
	for _, sibling := range siblings {
		names = append(names, sibling.name)
	}

	message := "unknown field " + join(path, name)

	suggestion := suggest(name, names)
	if suggestion != "" {
		suggestion = join(path, suggestion)
		message += "; did you mean " + suggestion + "?"
	}

	c.fail(node, message, CodeFieldUnknown, suggestion)
}

func (c *nameChecker) fail(node ast.Node, message string, code errors.Code, suggestion string) {
	var suggestions []string
	if suggestion != "" {
		suggestions = []string{suggestion}
	}

	c.err = c.program.source.positionError(&file.Error{
		Location: node.Location(),
		Message:  message,
	}, code, suggestions...)
}

// suggest returns the candidate closest to name, within two edits, or one
// for a name of four characters or fewer, when no other candidate is as
// close. It returns "" otherwise.
func suggest(name string, candidates []string) string {
	limit := 2
	if utf8.RuneCountInString(name) <= 4 {
		limit = 1
	}

	best, bestDistance, tie := "", limit+1, false

	for _, candidate := range candidates {
		d := distance(name, candidate)

		switch {
		case d < bestDistance:
			best, bestDistance, tie = candidate, d, false
		case d == bestDistance:
			tie = true
		}
	}

	if tie || bestDistance > limit {
		return ""
	}

	return best
}

// distance is the optimal string alignment distance between a and b: the
// edits, counting a swap of two neighbors as one, that turn a into b.
func distance(a, b string) int {
	x, y := []rune(a), []rune(b)

	d := make([][]int, len(x)+1)
	for i := range d {
		d[i] = make([]int, len(y)+1)
		d[i][0] = i
	}

	for j := range d[0] {
		d[0][j] = j
	}

	for i := 1; i <= len(x); i++ {
		for j := 1; j <= len(y); j++ {
			cost := 1
			if x[i-1] == y[j-1] {
				cost = 0
			}

			d[i][j] = min(d[i-1][j]+1, d[i][j-1]+1, d[i-1][j-1]+cost)

			if i > 1 && j > 1 && x[i-1] == y[j-2] && x[i-2] == y[j-1] {
				d[i][j] = min(d[i][j], d[i-2][j-2]+1)
			}
		}
	}

	return d[len(x)][len(y)]
}

// typeReplacer spells the Go types the variables of vars use the way their
// schema does: duration for [time.Duration], list(string) for []string and
// object tls for the struct at tls.
func typeReplacer(vars any) *strings.Replacer {
	names := map[string]string{
		"[]interface {}":                    "list(any)",
		"map[string]interface {}":           "map(any)",
		"interface {}":                      "any",
		durationType.String():               "duration",
		timeType.String():                   "time",
		reflect.TypeFor[float64]().String(): "float",
	}

	var walk func(path string, t reflect.Type)
	walk = func(path string, t reflect.Type) {
		name := schemaTypeName(path, t)
		if name != t.String() {
			names[t.String()] = name
		}

		switch {
		case t.Kind() == reflect.Slice || t.Kind() == reflect.Map:
			walk(path, t.Elem())
		case t.Kind() == reflect.Struct && t != timeType:
			for _, field := range fieldsOf(t) {
				walk(join(path, field.name), field.t)
			}
		}
	}

	for _, v := range variables(vars) {
		walk(v.name, v.t)
	}

	olds := make([]string, 0, len(names))
	for old := range names {
		olds = append(olds, old)
	}

	slices.SortFunc(olds, func(a, b string) int {
		return len(b) - len(a)
	})

	pairs := make([]string, 0, 2*len(olds))
	for _, old := range olds {
		pairs = append(pairs, old, names[old])
	}

	return strings.NewReplacer(pairs...)
}

// schemaTypeName is the schema's name for the type of the variable at path.
func schemaTypeName(path string, t reflect.Type) string {
	switch {
	case t == durationType:
		return "duration"
	case t == timeType:
		return "time"
	case t.Kind() == reflect.Interface:
		return "any"
	case t.Kind() == reflect.Float64:
		return "float"
	case t.Kind() == reflect.Slice:
		return "list(" + schemaTypeName(path, t.Elem()) + ")"
	case t.Kind() == reflect.Map:
		return "map(" + schemaTypeName(path, t.Elem()) + ")"
	case t.Kind() == reflect.Struct:
		return "object " + path
	}

	return t.String()
}
