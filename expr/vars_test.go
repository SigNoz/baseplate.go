package expr

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCheckVars(t *testing.T) {
	testCases := []struct {
		name            string
		vars            any
		expectedMessage string
	}{
		{
			name: "Struct_Accepted",
			vars: tHTTP{},
		},
		{
			name: "TypedZeroValues_Accepted",
			vars: map[string]any{
				"status": 0,
				"tags":   []string{},
			},
		},
		{
			name: "NestedBuiltinName_Accepted",
			vars: struct {
				TLS struct {
					Duration time.Duration `expr:"duration"`
				} `expr:"tls"`
			}{},
		},
		{
			name:            "Nil_VarsInvalid",
			vars:            nil,
			expectedMessage: "variables are nil, expected a struct or a map[string]any",
		},
		{
			name:            "Int64Field_VarsInvalid",
			vars:            struct{ Count int64 }{},
			expectedMessage: "variable Count is int64, expected int, float64, string, bool, time.Duration, time.Time, any, a slice, a map with string keys or a struct",
		},
		{
			name:            "PointerField_VarsInvalid",
			vars:            struct{ Limit *int }{},
			expectedMessage: "variable Limit is *int, expected int, float64, string, bool, time.Duration, time.Time, any, a slice, a map with string keys or a struct",
		},
		{
			name:            "IntKeyedMap_VarsInvalid",
			vars:            struct{ Codes map[int]string }{},
			expectedMessage: "variable Codes is map[int]string, expected int, float64, string, bool, time.Duration, time.Time, any, a slice, a map with string keys or a struct",
		},
		{
			name: "BuiltinName_VarsInvalid",
			vars: struct {
				Duration int `expr:"duration"`
			}{},
			expectedMessage: "variable duration takes the name of a builtin, expected another name",
		},
		{
			name: "AliasName_VarsInvalid",
			vars: struct {
				Both bool `expr:"AND"`
			}{},
			expectedMessage: `variable AND has the name "AND", expected one identifier other than AND, OR, NOT, IN, CONTAINS, REGEXP or regexp`,
		},
		{
			name: "DashedName_VarsInvalid",
			vars: struct {
				ContentType string `expr:"content-type"`
			}{},
			expectedMessage: `variable content-type has the name "content-type", expected one identifier other than AND, OR, NOT, IN, CONTAINS, REGEXP or regexp`,
		},
		{
			name:            "TypeWithMethods_VarsInvalid",
			vars:            time.Time{},
			expectedMessage: "variables are time.Time, which has methods, expected a struct without exported methods",
		},
		{
			name: "MapValueNil_VarsInvalid",
			vars: map[string]any{
				"status": nil,
			},
			expectedMessage: "variable status is nil, expected a typed zero value",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := Compile(`true`, TypedStatic(testCase.vars))
			if testCase.expectedMessage == "" {
				require.NoError(t, err)

				return
			}

			tAssertError(t, &tError{
				code:    CodeVarsInvalid,
				message: testCase.expectedMessage,
			}, err)
		})
	}
}
