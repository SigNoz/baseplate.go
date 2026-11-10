package expr

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/SigNoz/baseplate.go/errors"
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

// Errors from a TypedStatic program name types as the schema does, never as
// Go spells them.
func TestCheckNames_SchemaTypeNames(t *testing.T) {
	testCases := []struct {
		name            string
		src             string
		expectedMessage string
	}{
		{
			name:            "Duration",
			src:             `response_time < 500`,
			expectedMessage: "invalid operation: < (mismatched types duration and int)",
		},
		{
			name:            "Object",
			src:             `tls.certificate = "x"`,
			expectedMessage: "invalid operation: = (mismatched types object tls.certificate and string)",
		},
		{
			name:            "ListOfString",
			src:             `tls.certificate.sans > 1`,
			expectedMessage: "invalid operation: > (mismatched types list(string) and int)",
		},
		{
			name:            "MapOfString",
			src:             `headers = 1`,
			expectedMessage: "invalid operation: = (mismatched types map(string) and int)",
		},
		{
			name:            "Time",
			src:             `tls.certificate.not_after > 0`,
			expectedMessage: "invalid operation: > (mismatched types time and int)",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := Compile(testCase.src, tHTTPEnv())
			require.Error(t, err)
			assert.Equal(t, CodeTypeInvalid, errors.CodeOf(err))
			assert.Equal(t, testCase.expectedMessage, errors.MessageOf(err))
		})
	}
}
