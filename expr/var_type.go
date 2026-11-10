package expr

import (
	"github.com/SigNoz/baseplate.go/errors"
)

// The types a [Var] of a [Schema] can have.
var (
	VarTypeInt      = VarType{"int"}
	VarTypeFloat    = VarType{"float"}
	VarTypeString   = VarType{"string"}
	VarTypeBool     = VarType{"bool"}
	VarTypeDuration = VarType{"duration"}
	VarTypeTime     = VarType{"time"}
	VarTypeAny      = VarType{"any"}
	VarTypeList     = VarType{"list"}   // Elem is the type of its items.
	VarTypeMap      = VarType{"map"}    // Keys are strings; Elem is the type of its values.
	VarTypeObject   = VarType{"object"} // Fields are its fields.
)

var varTypes = []VarType{
	VarTypeInt,
	VarTypeFloat,
	VarTypeString,
	VarTypeBool,
	VarTypeDuration,
	VarTypeTime,
	VarTypeAny,
	VarTypeList,
	VarTypeMap,
	VarTypeObject,
}

// VarType is the type of a [Var], in the schema's own terms.
type VarType struct{ s string }

// NewVarType returns the var type named s. It returns a
// [github.com/SigNoz/baseplate.go/errors.TypeInvalidInput] error with
// [CodeVarTypeInvalid] for any other name.
func NewVarType(s string) (VarType, error) {
	for _, varType := range varTypes {
		if varType.s == s {
			return varType, nil
		}
	}

	return VarType{}, errors.Newf(errors.TypeInvalidInput, CodeVarTypeInvalid, "var type %q is invalid, expected one of int, float, string, bool, duration, time, any, list, map, object", s)
}

// String returns the var type's name.
func (varType VarType) String() string {
	return varType.s
}

// MarshalText returns the var type's name.
func (varType VarType) MarshalText() ([]byte, error) {
	return []byte(varType.s), nil
}

// UnmarshalText sets the var type from its name, as [NewVarType] parses
// it.
func (varType *VarType) UnmarshalText(text []byte) error {
	parsed, err := NewVarType(string(text))
	if err != nil {
		return err
	}

	*varType = parsed

	return nil
}
