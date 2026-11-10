package expr

import (
	"reflect"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/builtin"
	"github.com/expr-lang/expr/checker/nature"
	"github.com/expr-lang/expr/conf"
	"github.com/expr-lang/expr/file"
	"github.com/expr-lang/expr/parser"

	"github.com/SigNoz/baseplate.go/errors"
)

// nameError returns the checker's error about a name that vars, the vars of
// a [TypedStatic] program, don't declare, reported on s with
// [CodeNameUnknown] or [CodeFunctionUnknown] and the nearest declared name
// as a suggestion. It returns nil for any other error.
func nameError(tree *parser.Tree, err error, vars any, s *source) error {
	checked, ok := errors.AsType[*file.Error](err)
	if !ok {
		return nil
	}

	cache := new(nature.Cache)

	var (
		code       = CodeNameUnknown
		path       string
		candidates []string
	)

	switch {
	case strings.HasPrefix(checked.Message, "unknown name "):
		identifier, found := nodeAt[*ast.IdentifierNode](tree, checked.Location)
		if !found {
			return nil
		}

		if slices.ContainsFunc(nodesOf[*ast.CallNode](&tree.Node), func(call *ast.CallNode) bool { return call.Callee == ast.Node(identifier) }) {
			code = CodeFunctionUnknown
			candidates = builtin.Names
		} else {
			env := conf.EnvWithCache(cache, vars)
			candidates = keys(env.All(cache))
		}

		candidates = suggestions(identifier.Value, candidates)
	case strings.Contains(checked.Message, " has no field "):
		member, found := memberAt(tree, checked.Location)
		if !found {
			return nil
		}

		key, _ := member.Property.(*ast.StringNode)
		parent := member.Node.Type()

		for parent != nil && parent.Kind() == reflect.Pointer {
			parent = parent.Elem()
		}

		if parent == nil || parent.Kind() != reflect.Struct {
			return nil
		}

		path, _ = referencePath(member.Node, nil)
		candidates = suggestions(key.Value, keys(nature.StructFields(cache, parent)))
	default:
		return nil
	}

	message := checked.Message
	if len(candidates) > 0 {
		candidates[0] = join(path, candidates[0])
		message += "; did you mean " + candidates[0] + "?"
	}

	return s.positionError(&file.Error{
		Location: checked.Location,
		Message:  message,
	}, code, candidates...)
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

// memberAt returns the string-keyed member access whose key, or itself, is
// at location in tree.
func memberAt(tree *parser.Tree, location file.Location) (*ast.MemberNode, bool) {
	for _, member := range nodesOf[*ast.MemberNode](&tree.Node) {
		if _, ok := member.Property.(*ast.StringNode); ok && (member.Property.Location() == location || member.Location() == location) {
			return member, true
		}
	}

	return nil, false
}

func keys[V any](m map[string]V) []string {
	names := make([]string, 0, len(m))
	for name := range m {
		names = append(names, name)
	}

	return names
}

// suggestions returns the candidate closest to name, within two edits, or
// one for a name of four characters or fewer, when no other candidate is as
// close. It returns nil otherwise.
func suggestions(name string, candidates []string) []string {
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
		return nil
	}

	return []string{best}
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
