package expr

import (
	"reflect"
	"slices"
	"time"

	"github.com/expr-lang/expr/builtin"
	"github.com/expr-lang/expr/file"
	"github.com/expr-lang/expr/parser/lexer"

	"github.com/SigNoz/baseplate.go/errors"
)

var (
	durationType = reflect.TypeFor[time.Duration]()
	timeType     = reflect.TypeFor[time.Time]()
	leafTypes    = []reflect.Type{
		reflect.TypeFor[int](),
		reflect.TypeFor[float64](),
		reflect.TypeFor[string](),
		reflect.TypeFor[bool](),
		durationType,
		timeType,
	}
)

// reserved are the top-level names a variable can't take: what the language
// itself gives meaning to.
var reserved = []string{"true", "false", "nil", "$env"}

// checkVars returns an error when vars can't declare the variables of a
// [TypedStatic] program.
func checkVars(vars any) error {
	t := reflect.TypeOf(vars)

	switch {
	case t == nil:
		return errors.New(errors.TypeInvalidInput, CodeVarsInvalid, "variables are nil, expected a struct or a map[string]any")
	case t.Kind() == reflect.Struct:
		return checkStruct(t, "", true)
	case t == reflect.TypeFor[map[string]any]():
		value := reflect.ValueOf(vars)

		for _, key := range value.MapKeys() {
			name := key.String()
			if err := checkName(name, name, true); err != nil {
				return err
			}

			element := value.MapIndex(key).Elem()
			if !element.IsValid() {
				return errors.Newf(errors.TypeInvalidInput, CodeVarsInvalid, "variable %s is nil, expected a typed zero value", name)
			}

			if err := checkType(element.Type(), name); err != nil {
				return err
			}
		}

		return nil
	}

	return errors.Newf(errors.TypeInvalidInput, CodeVarsInvalid, "variables are %s, expected a struct or a map[string]any", t)
}

// checkType returns an error when a variable at path has a type outside the
// ones a [TypedStatic] program supports.
func checkType(t reflect.Type, path string) error {
	switch {
	case slices.Contains(leafTypes, t):
		return nil
	case t.Kind() == reflect.Interface && t.NumMethod() == 0:
		return nil
	case t.Kind() == reflect.Slice:
		return checkType(t.Elem(), path+"[]")
	case t.Kind() == reflect.Map && t.Key() == reflect.TypeFor[string]():
		return checkType(t.Elem(), path+"[]")
	case t.Kind() == reflect.Struct:
		return checkStruct(t, path, false)
	}

	return errors.Newf(errors.TypeInvalidInput, CodeVarsInvalid, "variable %s is %s, expected int, float64, string, bool, time.Duration, time.Time, any, a slice, a map with string keys or a struct", path, t)
}

// checkStruct checks the fields of a struct at path, top being the
// variables themselves. Embedded structs add their fields to the struct's
// own, as expr reads them.
func checkStruct(t reflect.Type, path string, top bool) error {
	if t.NumMethod() > 0 || reflect.PointerTo(t).NumMethod() > 0 {
		if top {
			return errors.Newf(errors.TypeInvalidInput, CodeVarsInvalid, "variables are %s, which has methods, expected a struct without exported methods", t)
		}

		return errors.Newf(errors.TypeInvalidInput, CodeVarsInvalid, "variable %s is %s, which has methods, expected a struct without exported methods", path, t)
	}

	names := map[string]bool{}

	for _, field := range fieldsOf(t) {
		fieldPath := join(path, field.name)
		if names[field.name] {
			return errors.Newf(errors.TypeInvalidInput, CodeVarsInvalid, "variable %s is declared twice, expected unique names", fieldPath)
		}

		names[field.name] = true

		if err := checkName(field.name, fieldPath, top); err != nil {
			return err
		}

		if err := checkType(field.t, fieldPath); err != nil {
			return err
		}
	}

	return nil
}

// checkName returns an error when name can't name a variable: it must lex
// as one identifier that the shim leaves alone, and a top-level name must
// not take a builtin function's or the language's own name.
func checkName(name, path string, top bool) error {
	tokens, err := lexer.Lex(file.NewSource(name))
	if err != nil || len(tokens) != 2 || tokens[0].Kind != lexer.Identifier || tokens[0].Value != name || aliases[name] != "" {
		return errors.Newf(errors.TypeInvalidInput, CodeVarsInvalid, "variable %s has the name %q, expected one identifier other than AND, OR, NOT, IN, CONTAINS, REGEXP or regexp", path, name)
	}

	if top && (slices.Contains(builtin.Names, name) || slices.Contains(reserved, name)) {
		return errors.Newf(errors.TypeInvalidInput, CodeVarsInvalid, "variable %s takes the name of a builtin, expected another name", path)
	}

	return nil
}

// fieldName is the name expr gives a struct field: its expr tag, or else
// its Go name. A field tagged "-" has none.
func fieldName(field reflect.StructField) (string, bool) {
	switch tag := field.Tag.Get("expr"); tag {
	case "-":
		return "", false
	case "":
		return field.Name, true
	default:
		return tag, true
	}
}

func join(path, name string) string {
	if path == "" {
		return name
	}

	return path + "." + name
}
