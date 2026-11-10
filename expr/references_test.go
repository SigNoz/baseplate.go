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
