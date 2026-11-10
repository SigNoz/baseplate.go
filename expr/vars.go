package expr

import (
	"reflect"
	"slices"
	"strings"

	"github.com/expr-lang/expr/builtin"
	"github.com/expr-lang/expr/checker/nature"
	"github.com/expr-lang/expr/file"
	"github.com/expr-lang/expr/parser/lexer"

	"github.com/SigNoz/baseplate.go/errors"
)

// reserved are the names a top-level var can't take: what the language
// itself gives meaning to.
var reserved = []string{"true", "false", "nil", "$env"}

// checkVars returns an error when vars can't declare the vars of a
// [TypedStatic] program: vars must be a struct or a map[string]any, and
// every var and field must have a name an expression can reach.
func checkVars(vars any) error {
	t := reflect.TypeOf(vars)

	switch {
	case t == nil:
		return errors.New(errors.TypeInvalidInput, CodeVarsInvalid, "vars are nil, expected a struct or a map[string]any")
	case t.Kind() == reflect.Struct:
		return checkFields(t, "", map[reflect.Type]bool{})
	case t == reflect.TypeFor[map[string]any]():
		value := reflect.ValueOf(vars)

		for _, key := range value.MapKeys() {
			name := key.String()
			if err := checkName(name, name); err != nil {
				return err
			}

			if element := value.MapIndex(key).Elem(); element.IsValid() {
				if err := checkHeld(element.Type(), name, map[reflect.Type]bool{}); err != nil {
					return err
				}
			}
		}

		return nil
	}

	return errors.Newf(errors.TypeInvalidInput, CodeVarsInvalid, "vars are %s, expected a struct or a map[string]any", t)
}

// checkFields checks the names of the fields of struct type t at path, as
// expr reads them, and of the structs they hold.
func checkFields(t reflect.Type, path string, seen map[reflect.Type]bool) error {
	if seen[t] {
		return nil
	}

	seen[t] = true

	fields := nature.StructFields(new(nature.Cache), t)

	for _, name := range slices.Sorted(func(yield func(string) bool) {
		for name := range fields {
			if !yield(name) {
				return
			}
		}
	}) {
		fieldPath := join(path, name)
		if err := checkName(name, fieldPath); err != nil {
			return err
		}

		if err := checkHeld(fields[name].Type, fieldPath, seen); err != nil {
			return err
		}
	}

	return nil
}

// checkHeld checks the struct that t is or holds, as the items of a slice
// or the values of a map.
func checkHeld(t reflect.Type, path string, seen map[reflect.Type]bool) error {
	for t != nil && slices.Contains([]reflect.Kind{reflect.Slice, reflect.Array, reflect.Map, reflect.Pointer}, t.Kind()) {
		t = t.Elem()
	}

	if t == nil || t.Kind() != reflect.Struct || t == timeType {
		return nil
	}

	return checkFields(t, path, seen)
}

// checkName returns an error when the var or field at path can't be
// reached by its name: the name must lex as one identifier that the shim
// leaves alone, and a var's name must not be a builtin function's or the
// language's own.
func checkName(name, path string) error {
	tokens, err := lexer.Lex(file.NewSource(name))
	if err != nil || len(tokens) != 2 || tokens[0].Kind != lexer.Identifier || tokens[0].Value != name || aliases[name] != "" {
		return errors.Newf(errors.TypeInvalidInput, CodeVarsInvalid, "%s has the name %q, expected one identifier other than AND, OR, NOT, IN, CONTAINS, REGEXP or regexp", subject(path), name)
	}

	if !strings.Contains(path, ".") && (slices.Contains(builtin.Names, name) || slices.Contains(reserved, name)) {
		return errors.Newf(errors.TypeInvalidInput, CodeVarsInvalid, "var %s takes the name of a builtin, expected another name", path)
	}

	return nil
}

func join(path, name string) string {
	if path == "" {
		return name
	}

	return path + "." + name
}

// subject names what is at path in an error: a var at the top level, a
// field in a struct.
func subject(path string) string {
	if strings.Contains(path, ".") {
		return "field " + path
	}

	return "var " + path
}
