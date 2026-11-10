package expr

import (
	"reflect"
)

// Schema is what a program's names are, in a form a client can read: a
// client that checks expressions as they are typed compiles against it.
type Schema struct {
	Open   bool   `json:"open,omitempty"` // Any name is allowed, as in Untyped and TypedDynamic; Vars is empty.
	Vars   []Var  `json:"vars,omitempty"` // The declared vars, as in TypedStatic.
	Limits Limits `json:"limits"`
}

// Var is a declared var, or a field, item or value of one.
type Var struct {
	Name   string  `json:"name,omitempty"` // Empty for the Elem of a list or map.
	Type   VarType `json:"type"`
	Elem   *Var    `json:"elem,omitempty"`   // The items of a list or the values of a map.
	Fields []Var   `json:"fields,omitempty"` // The fields of an object.
}

// Limits are the limits a program is compiled with. Zero is unset. The
// memory limit is left out, as it only applies when a program runs.
type Limits struct {
	Nesting uint `json:"nesting,omitempty"`
	Pattern uint `json:"pattern,omitempty"`
	Nodes   uint `json:"nodes,omitempty"`
}

// SchemaOf returns the vars that vars declares, as [TypedStatic] reads them. It
// returns the errors [TypedStatic] reports for vars that break its rules.
func SchemaOf(vars any) (Schema, error) {
	if err := checkVars(vars); err != nil {
		return Schema{}, err
	}

	var declared []Var
	for _, v := range variables(vars) {
		declared = append(declared, varOf(v.name, v.t))
	}

	return Schema{Vars: declared}, nil
}

// Schema returns the schema of e's programs: the declared vars of a
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
		Nesting: e.nesting,
		Pattern: e.pattern,
		Nodes:   e.nodes,
	}

	return schema, nil
}

func varOf(name string, t reflect.Type) Var {
	field := Var{Name: name}

	switch {
	case t == durationType:
		field.Type = VarTypeDuration
	case t == timeType:
		field.Type = VarTypeTime
	case t.Kind() == reflect.Interface:
		field.Type = VarTypeAny
	case t.Kind() == reflect.Slice || t.Kind() == reflect.Map:
		field.Type = VarTypeList
		if t.Kind() == reflect.Map {
			field.Type = VarTypeMap
		}

		elem := varOf("", t.Elem())
		field.Elem = &elem
	case t.Kind() == reflect.Struct:
		field.Type = VarTypeObject
		for _, f := range fieldsOf(t) {
			field.Fields = append(field.Fields, varOf(f.name, f.t))
		}
	default:
		field.Type = map[reflect.Kind]VarType{
			reflect.Int:     VarTypeInt,
			reflect.Float64: VarTypeFloat,
			reflect.String:  VarTypeString,
			reflect.Bool:    VarTypeBool,
		}[t.Kind()]
	}

	return field
}
