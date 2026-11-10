package expr

import (
	"reflect"
)

// Schema is what a program's names are, in a form a client can read: a
// client that checks expressions as they are typed compiles against it.
type Schema struct {
	Open   bool    `json:"open,omitempty"`   // Any name is allowed, as in Untyped and TypedDynamic; Fields is empty.
	Fields []Field `json:"fields,omitempty"` // The declared variables, as in TypedStatic.
	Limits Limits  `json:"limits"`
}

// Field is a declared variable, or a field, item or value of one.
type Field struct {
	Name   string    `json:"name,omitempty"` // Empty for the Elem of a list or map.
	Type   FieldType `json:"type"`
	Elem   *Field    `json:"elem,omitempty"`   // The items of a list or the values of a map.
	Fields []Field   `json:"fields,omitempty"` // The fields of an object.
}

// Limits are the limits a program is compiled with. Zero is unset. The
// memory limit is left out, as it only applies when a program runs.
type Limits struct {
	Nesting       uint `json:"nesting,omitempty"`
	PatternLength uint `json:"pattern_length,omitempty"`
	Nodes         uint `json:"nodes,omitempty"`
}

// SchemaOf returns the fields of vars, as [TypedStatic] reads them. It
// returns the errors [TypedStatic] reports for vars that break its rules.
func SchemaOf(vars any) (Schema, error) {
	if err := checkVars(vars); err != nil {
		return Schema{}, err
	}

	var fields []Field
	for _, v := range variables(vars) {
		fields = append(fields, fieldOf(v.name, v.t))
	}

	return Schema{Fields: fields}, nil
}

// Schema returns the schema of e's programs: the declared variables of a
// [TypedStatic] Env, or an open schema for any other, with e's limits. It
// returns the errors [TypedStatic] reports for vars that break its rules.
func (e Env) Schema() (Schema, error) {
	schema := Schema{Open: true}

	if e.static {
		var err error

		schema, err = SchemaOf(e.vars)
		if err != nil {
			return Schema{}, err
		}
	}

	schema.Limits = Limits{
		Nesting:       e.nesting,
		PatternLength: e.patternLength,
		Nodes:         e.nodes,
	}

	return schema, nil
}

func fieldOf(name string, t reflect.Type) Field {
	field := Field{Name: name}

	switch {
	case t == durationType:
		field.Type = FieldTypeDuration
	case t == timeType:
		field.Type = FieldTypeTime
	case t.Kind() == reflect.Interface:
		field.Type = FieldTypeAny
	case t.Kind() == reflect.Slice || t.Kind() == reflect.Map:
		field.Type = FieldTypeList
		if t.Kind() == reflect.Map {
			field.Type = FieldTypeMap
		}

		elem := fieldOf("", t.Elem())
		field.Elem = &elem
	case t.Kind() == reflect.Struct:
		field.Type = FieldTypeObject
		for _, f := range fieldsOf(t) {
			field.Fields = append(field.Fields, fieldOf(f.name, f.t))
		}
	default:
		field.Type = map[reflect.Kind]FieldType{
			reflect.Int:     FieldTypeInt,
			reflect.Float64: FieldTypeFloat,
			reflect.String:  FieldTypeString,
			reflect.Bool:    FieldTypeBool,
		}[t.Kind()]
	}

	return field
}
