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
			name: "AnyGoType_Accepted",
			vars: struct {
				Count int64
				Limit *int
				Codes map[int]string
			}{},
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
			expectedMessage: "vars are nil, expected a struct or a map[string]any",
		},
		{
			name:            "NotStructOrMap_VarsInvalid",
			vars:            []string{},
			expectedMessage: "vars are []string, expected a struct or a map[string]any",
		},
		{
			name: "BuiltinName_VarsInvalid",
			vars: struct {
				Duration int `expr:"duration"`
			}{},
			expectedMessage: "var duration takes the name of a builtin, expected another name",
		},
		{
			name: "AliasName_VarsInvalid",
			vars: struct {
				Both bool `expr:"AND"`
			}{},
			expectedMessage: `var AND has the name "AND", expected one identifier other than AND, OR, NOT, IN, CONTAINS, REGEXP or regexp`,
		},
		{
			name: "DashedName_VarsInvalid",
			vars: struct {
				ContentType string `expr:"content-type"`
			}{},
			expectedMessage: `var content-type has the name "content-type", expected one identifier other than AND, OR, NOT, IN, CONTAINS, REGEXP or regexp`,
		},
		{
			name: "NestedAliasName_VarsInvalid",
			vars: struct {
				Answers []struct {
					In string `expr:"IN"`
				} `expr:"answers"`
			}{},
			expectedMessage: `field answers.IN has the name "IN", expected one identifier other than AND, OR, NOT, IN, CONTAINS, REGEXP or regexp`,
		},
		{
			name: "MapValueAliasName_VarsInvalid",
			vars: map[string]any{
				"tls": struct {
					Or string `expr:"OR"`
				}{},
			},
			expectedMessage: `field tls.OR has the name "OR", expected one identifier other than AND, OR, NOT, IN, CONTAINS, REGEXP or regexp`,
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
