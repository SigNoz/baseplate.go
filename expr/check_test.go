package expr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSuggest(t *testing.T) {
	testCases := []struct {
		name       string
		input      string
		candidates []string
		expected   string
	}{
		{
			name:       "Swap_OneEdit",
			input:      "stauts_code",
			candidates: []string{"status_code", "body"},
			expected:   "status_code",
		},
		{
			name:       "TwoEdits_Suggested",
			input:      "verzion",
			candidates: []string{"version", "verified"},
			expected:   "version",
		},
		{
			name:       "ThreeEdits_NoSuggestion",
			input:      "latency",
			candidates: []string{"status_code", "body"},
		},
		{
			name:       "ShortName_TwoEdits_NoSuggestion",
			input:      "bdy",
			candidates: []string{"body"},
			expected:   "body",
		},
		{
			name:       "ShortName_OverOneEdit_NoSuggestion",
			input:      "bx",
			candidates: []string{"body"},
		},
		{
			name:       "Tie_NoSuggestion",
			input:      "cat",
			candidates: []string{"bat", "car"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expected, suggest(testCase.input, testCase.candidates))
		})
	}
}
