package expr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStringify_Truncates(t *testing.T) {
	testCases := []struct {
		name           string
		value          any
		expectedLength int
	}{
		{
			name:           "Short_Kept",
			value:          "degraded",
			expectedLength: 8,
		},
		{
			name:           "Long_CutTo256",
			value:          string(make([]rune, 300)),
			expectedLength: 256,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Len(t, []rune(stringify(testCase.value)), testCase.expectedLength)
		})
	}
}

func TestStringify(t *testing.T) {
	cycle := map[string]any{}
	cycle["self"] = cycle

	testCases := []struct {
		name     string
		value    any
		expected string
	}{
		{
			name:     "Int_GoFormat",
			value:    7,
			expected: "7",
		},
		{
			name:     "Array_JSON",
			value:    [2]int{1, 2},
			expected: "[1,2]",
		},
		{
			name:     "MapHoldingItself_TypeName",
			value:    cycle,
			expected: "map[string]interface {}",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expected, stringify(testCase.value))
		})
	}
}
