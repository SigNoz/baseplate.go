package errors

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestMustNewCode_InvalidPanics(t *testing.T) {
	assert.Panics(t, func() { MustNewCode("bad code") })
}

func TestCode_UnmarshalText(t *testing.T) {
	testCases := []struct {
		name          string
		text          string
		expectedCode  Code
		expectedError bool
	}{
		{name: "SnakeCase_Parsed", text: "rate_limited", expectedCode: MustNewCode("rate_limited")},
		{name: "CamelCase_Parsed", text: "SecretNotFound", expectedCode: MustNewCode("SecretNotFound")},
		{name: "Dash_InvalidInput", text: "rate-limited", expectedError: true},
		{name: "Empty_InvalidInput", text: "", expectedError: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var parsed Code

			err := parsed.UnmarshalText([]byte(testCase.text))
			if testCase.expectedError {
				require.Error(t, err)
				assert.Equal(t, TypeInvalidInput, TypeOf(err))

				return
			}

			require.NoError(t, err)
			assert.Equal(t, testCase.expectedCode, parsed)
		})
	}
}

func TestCode_JSON(t *testing.T) {
	type tBody struct {
		Code Code `json:"code"`
	}

	body, err := json.Marshal(tBody{Code: MustNewCode("license_expired")})
	require.NoError(t, err)
	assert.JSONEq(t, `{"code":"license_expired"}`, string(body))

	var parsed tBody
	require.NoError(t, json.Unmarshal(body, &parsed))
	assert.Equal(t, MustNewCode("license_expired"), parsed.Code)
}
