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
		t := reflect.TypeOf(v)

		switch {
		case slices.Contains([]reflect.Kind{reflect.Map, reflect.Slice, reflect.Array}, t.Kind()) || (t.Kind() == reflect.Struct && t != timeType):
			// Go's formatting follows a value that holds itself until the
			// stack overflows, so a value JSON rejects shows its type.
			encoded, err := json.Marshal(v)
			if err != nil {
				text = t.String()
			} else {
				text = string(encoded)
			}
		default:
			text = fmt.Sprint(v)
		}
	}

	if utf8.RuneCountInString(text) > maxValueLength {
		text = string([]rune(text)[:maxValueLength])
	}

	return text
}

// references returns the paths of the variables tree reads: names, and
// chains of string-keyed members on them, longest first in each chain. A
// name that a let binds is a variable only outside the let's body, so the
// x of let x = x + 1 is read and the x after the semicolon isn't.
func references(tree *parser.Tree) []string {
	ends := map[ast.Node]bool{}
	bound := map[*ast.IdentifierNode]bool{}

	visit(&tree.Node, func(node *ast.Node) {
		switch n := (*node).(type) {
		case *ast.MemberNode:
			if _, ok := n.Property.(*ast.StringNode); ok && !n.Method {
				ends[n.Node] = true
			}
		case *ast.CallNode:
			ends[n.Callee] = true
		case *ast.VariableDeclaratorNode:
			for _, identifier := range nodesOf[*ast.IdentifierNode](&n.Expr) {
				if identifier.Value == n.Name {
					bound[identifier] = true
				}
			}
		}
	})

	var found []string

	visit(&tree.Node, func(node *ast.Node) {
		if ends[*node] {
			return
		}

		if path, ok := referencePath(*node, bound); ok {
			found = append(found, path)
		}
	})

	slices.Sort(found)

	return slices.Compact(found)
}

// referencePath returns node as a reference when it is a name, other than
// one in bound, or a chain of string-keyed members on one. A member of
// $env is the variable it names: $env.status_code is status_code, and a key
// that isn't a plain name stays on $env, as in $env["k8s.pod"].
func referencePath(node ast.Node, bound map[*ast.IdentifierNode]bool) (string, bool) {
	switch n := node.(type) {
	case *ast.IdentifierNode:
		if bound[n] || n.Value == "$env" {
			return "", false
		}

		return n.Value, true
	case *ast.MemberNode:
		key, ok := n.Property.(*ast.StringNode)
		if !ok || n.Method {
			return "", false
		}

		plain := identifierPattern.MatchString(key.Value) && aliases[key.Value] == ""

		if root, ok := n.Node.(*ast.IdentifierNode); ok && root.Value == "$env" {
			if plain {
				return key.Value, true
			}

			return "$env[" + strconv.Quote(key.Value) + "]", true
		}

		parent, ok := referencePath(n.Node, bound)
		if !ok {
			return "", false
		}

		if plain {
			return parent + "." + key.Value, true
		}

		return parent + "[" + strconv.Quote(key.Value) + "]", true
	}

	return "", false
}
