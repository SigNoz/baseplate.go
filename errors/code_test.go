package errors

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewCode checks which code formats NewCode accepts.
func TestNewCode(t *testing.T) {
	testCases := []struct {
		name          string
		s             string
		expectedError bool
	}{
		{name: "SnakeCase_Accepted", s: "license_unavailable"},
		{name: "CamelCase_Accepted", s: "SecretNotFound"},
		{name: "Digits_Accepted", s: "http2_disabled"},
		{name: "Empty_InvalidInput", s: "", expectedError: true},
		{name: "LeadingDigit_InvalidInput", s: "2fa_required", expectedError: true},
		{name: "LeadingUnderscore_InvalidInput", s: "_internal", expectedError: true},
		{name: "Dash_InvalidInput", s: "not-found", expectedError: true},
		{name: "Space_InvalidInput", s: "not found", expectedError: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			code, err := NewCode(testCase.s)
			if testCase.expectedError {
				require.Error(t, err)
				assert.Equal(t, TypeInvalidInput, TypeOf(err))
				assert.Equal(t, CodeInvalidInput, CodeOf(err))
				assert.Contains(t, err.Error(), "expected a letter")

				return
			}

			require.NoError(t, err)
			assert.Equal(t, testCase.s, code.String())
		})
	}
}

// TestMustNewCode_InvalidPanics checks that MustNewCode panics on an invalid
// code.
func TestMustNewCode_InvalidPanics(t *testing.T) {
	assert.Panics(t, func() { MustNewCode("bad code") })
}
