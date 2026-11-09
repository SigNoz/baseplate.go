package errors

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTyp_Text_RoundTrips(t *testing.T) {
	for _, declared := range typs {
		t.Run(declared.String(), func(t *testing.T) {
			text, err := declared.MarshalText()
			require.NoError(t, err)

			var parsed typ
			require.NoError(t, parsed.UnmarshalText(text))

			assert.Equal(t, declared, parsed)
		})
	}
}

func TestTyp_UnmarshalText(t *testing.T) {
	testCases := []struct {
		name          string
		text          string
		expectedType  typ
		expectedError bool
	}{
		{name: "Declared_Parsed", text: "too-many-requests", expectedType: TypeTooManyRequests},
		{name: "Unknown_Parsed", text: "unknown", expectedType: TypeUnknown},
		{name: "Undeclared_InvalidInput", text: "teapot", expectedError: true},
		{name: "SnakeCase_InvalidInput", text: "not_found", expectedError: true},
		{name: "Empty_InvalidInput", text: "", expectedError: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var parsed typ

			err := parsed.UnmarshalText([]byte(testCase.text))
			if testCase.expectedError {
				require.Error(t, err)
				assert.Equal(t, TypeInvalidInput, TypeOf(err))
				assert.Contains(t, err.Error(), "expected one of unknown, invalid-input")

				return
			}

			require.NoError(t, err)
			assert.Equal(t, testCase.expectedType, parsed)
		})
	}
}

func TestTyp_JSON(t *testing.T) {
	type tBody struct {
		Type typ `json:"type"`
	}

	body, err := json.Marshal(tBody{Type: TypeLicenseUnavailable})
	require.NoError(t, err)
	assert.JSONEq(t, `{"type":"license-unavailable"}`, string(body))

	var parsed tBody
	require.NoError(t, json.Unmarshal(body, &parsed))
	assert.Equal(t, TypeLicenseUnavailable, parsed.Type)
}
