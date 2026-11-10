package expr

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/parser"
)

// timeType is [time.Time], a struct that is printed as Go formats it, not as
// JSON.
var timeType = reflect.TypeFor[time.Time]()

// maxValueLength caps the characters of a value that
// [Program.ReferenceValues] returns.
const maxValueLength = 256

var identifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

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
	ends := map[ast.Node]bool{}
	lets := map[string]bool{}

	visit(&tree.Node, func(node *ast.Node) {
		switch n := (*node).(type) {
		case *ast.MemberNode:
			if _, ok := n.Property.(*ast.StringNode); ok && !n.Method {
				ends[n.Node] = true
			}
		case *ast.CallNode:
			ends[n.Callee] = true
		case *ast.VariableDeclaratorNode:
			lets[n.Name] = true
		}
	})

	var found []string

	visit(&tree.Node, func(node *ast.Node) {
		if ends[*node] {
			return
		}

		if path, ok := referencePath(*node, lets); ok {
			found = append(found, path)
		}
	})

	slices.Sort(found)

	return slices.Compact(found)
}

// referencePath returns node as a reference when it is a name, other than
// one bound by let, or a chain of string-keyed members on one.
func referencePath(node ast.Node, lets map[string]bool) (string, bool) {
	switch n := node.(type) {
	case *ast.IdentifierNode:
		if lets[n.Value] || n.Value == "$env" {
			return "", false
		}

		return n.Value, true
	case *ast.MemberNode:
		key, ok := n.Property.(*ast.StringNode)
		if !ok || n.Method {
			return "", false
		}

		parent, ok := referencePath(n.Node, lets)
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
