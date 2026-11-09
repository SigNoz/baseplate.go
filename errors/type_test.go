package errors

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewType(t *testing.T) {
	testCases := []struct {
		name              string
		typeName          string
		status            int
		retryable         bool
		expectedError     bool
		expectedRetryable bool
	}{
		{name: "KebabCase_Accepted", typeName: "license-unavailable", status: http.StatusUnavailableForLegalReasons, retryable: false},
		{name: "OneWord_Retryable_Accepted", typeName: "throttled", status: http.StatusTooManyRequests, retryable: true, expectedRetryable: true},
		{name: "Status599_Accepted", typeName: "upstream-broken", status: 599, retryable: true, expectedRetryable: true},
		{name: "SnakeCase_InvalidInput", typeName: "license_unavailable", status: http.StatusBadRequest, expectedError: true},
		{name: "Uppercase_InvalidInput", typeName: "LicenseUnavailable", status: http.StatusBadRequest, expectedError: true},
		{name: "TrailingDash_InvalidInput", typeName: "quota-", status: http.StatusBadRequest, expectedError: true},
		{name: "Empty_InvalidInput", typeName: "", status: http.StatusBadRequest, expectedError: true},
		{name: "SuccessStatus_InvalidInput", typeName: "accepted", status: http.StatusOK, expectedError: true},
		{name: "StatusAbove599_InvalidInput", typeName: "broken", status: 600, expectedError: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			typ, err := NewType(testCase.typeName, testCase.status, testCase.retryable)
			if testCase.expectedError {
				require.Error(t, err)
				assert.Equal(t, TypeInvalidInput, TypeOf(err))

				return
			}

			require.NoError(t, err)
			assert.Equal(t, testCase.typeName, typ.String())

			err = New(typ, MustNewCode("own_type"), "failure")
			assert.Equal(t, testCase.status, HTTPStatus(err))
			assert.Equal(t, testCase.expectedRetryable, IsRetryable(err))
			assert.True(t, Ast(err, MustNewType(testCase.typeName, testCase.status, testCase.retryable)))
		})
	}
}

func TestMustNewType_InvalidPanics(t *testing.T) {
	assert.Panics(t, func() { MustNewType("Not Kebab", http.StatusBadRequest, false) })
}

func TestType_Text_RoundTrips(t *testing.T) {
	for _, declared := range baseTypes {
		t.Run(declared.String(), func(t *testing.T) {
			text, err := declared.MarshalText()
			require.NoError(t, err)

			var parsed Type
			require.NoError(t, parsed.UnmarshalText(text))

			assert.Equal(t, declared, parsed)
		})
	}
}

func TestType_UnmarshalText(t *testing.T) {
	testCases := []struct {
		name          string
		text          string
		expectedType  Type
		expectedError bool
	}{
		{name: "BaseType_Parsed", text: "too-many-requests", expectedType: TypeTooManyRequests},
		{name: "Unknown_Parsed", text: "unknown", expectedType: TypeUnknown},
		{name: "PackageOwnType_InvalidInput", text: "license-unavailable", expectedError: true},
		{name: "SnakeCase_InvalidInput", text: "not_found", expectedError: true},
		{name: "Empty_InvalidInput", text: "", expectedError: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var parsed Type

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

func TestType_JSON(t *testing.T) {
	type tBody struct {
		Type Type `json:"type"`
	}

	body, err := json.Marshal(tBody{Type: TypeConflict})
	require.NoError(t, err)
	assert.JSONEq(t, `{"type":"conflict"}`, string(body))

	var parsed tBody
	require.NoError(t, json.Unmarshal(body, &parsed))
	assert.Equal(t, TypeConflict, parsed.Type)
}
