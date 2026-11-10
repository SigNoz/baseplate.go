package expr

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFieldType(t *testing.T) {
	testCases := []struct {
		name          string
		input         string
		expected      FieldType
		expectedError *tError
	}{
		{
			name:     "Duration_Parsed",
			input:    "duration",
			expected: FieldTypeDuration,
		},
		{
			name:     "Object_Parsed",
			input:    "object",
			expected: FieldTypeObject,
		},
		{
			name:  "Unknown_FieldTypeInvalid",
			input: "int64",
			expectedError: &tError{
				code:    CodeFieldTypeInvalid,
				message: `field type "int64" is invalid, expected one of int, float, string, bool, duration, time, any, list, map, object`,
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := NewFieldType(testCase.input)
			if testCase.expectedError != nil {
				tAssertError(t, testCase.expectedError, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, testCase.expected, got)

			var parsed FieldType
			require.NoError(t, parsed.UnmarshalText([]byte(got.String())))
			assert.Equal(t, got, parsed)
		})
	}
}
