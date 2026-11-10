package expr

import (
	"github.com/SigNoz/baseplate.go/errors"
)

// The types a [Field] of a [Schema] can have.
var (
	FieldTypeInt      = FieldType{"int"}
	FieldTypeFloat    = FieldType{"float"}
	FieldTypeString   = FieldType{"string"}
	FieldTypeBool     = FieldType{"bool"}
	FieldTypeDuration = FieldType{"duration"}
	FieldTypeTime     = FieldType{"time"}
	FieldTypeAny      = FieldType{"any"}
	FieldTypeList     = FieldType{"list"}   // Elem is the type of its items.
	FieldTypeMap      = FieldType{"map"}    // Keys are strings; Elem is the type of its values.
	FieldTypeObject   = FieldType{"object"} // Fields are its fields.
)

var fieldTypes = []FieldType{
	FieldTypeInt,
	FieldTypeFloat,
	FieldTypeString,
	FieldTypeBool,
	FieldTypeDuration,
	FieldTypeTime,
	FieldTypeAny,
	FieldTypeList,
	FieldTypeMap,
	FieldTypeObject,
}

// FieldType is the type of a [Field], in the schema's own terms.
type FieldType struct{ s string }

// NewFieldType returns the field type named s. It returns a
// [github.com/SigNoz/baseplate.go/errors.TypeInvalidInput] error with
// [CodeFieldTypeInvalid] for any other name.
func NewFieldType(s string) (FieldType, error) {
	for _, fieldType := range fieldTypes {
		if fieldType.s == s {
			return fieldType, nil
		}
	}

	return FieldType{}, errors.Newf(errors.TypeInvalidInput, CodeFieldTypeInvalid, "field type %q is invalid, expected one of int, float, string, bool, duration, time, any, list, map, object", s)
}

// String returns the field type's name.
func (fieldType FieldType) String() string {
	return fieldType.s
}

// MarshalText returns the field type's name.
func (fieldType FieldType) MarshalText() ([]byte, error) {
	return []byte(fieldType.s), nil
}

// UnmarshalText sets the field type from its name, as [NewFieldType] parses
// it.
func (fieldType *FieldType) UnmarshalText(text []byte) error {
	parsed, err := NewFieldType(string(text))
	if err != nil {
		return err
	}

	*fieldType = parsed

	return nil
}
