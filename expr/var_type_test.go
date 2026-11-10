package expr

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewVarType(t *testing.T) {
	testCases := []struct {
		name          string
		input         string
		expected      VarType
		expectedError *tError
	}{
		{
			name:     "Duration_Parsed",
			input:    "duration",
			expected: VarTypeDuration,
		},
		{
			name:     "Object_Parsed",
			input:    "object",
			expected: VarTypeObject,
		},
		{
			name:  "Unknown_FieldTypeInvalid",
			input: "int64",
			expectedError: &tError{
				code:    CodeVarTypeInvalid,
				message: `var type "int64" is invalid, expected one of int, float, string, bool, duration, time, any, list, map, object`,
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := NewVarType(testCase.input)
			if testCase.expectedError != nil {
				tAssertError(t, testCase.expectedError, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, testCase.expected, got)

			var parsed VarType
			require.NoError(t, parsed.UnmarshalText([]byte(got.String())))
			assert.Equal(t, got, parsed)
		})
	}
}
