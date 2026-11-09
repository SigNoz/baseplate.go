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
		{name: "BaseTypeName_InvalidInput", typeName: "not-found", status: http.StatusNotFound, expectedError: true},
		{name: "UnknownTypeName_InvalidInput", typeName: "unknown", status: http.StatusInternalServerError, expectedError: true},
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

func TestType_MarshalJSON(t *testing.T) {
	testCases := []struct {
		name         string
		typ          Type
		expectedJSON string
	}{
		{name: "BaseType", typ: TypeConflict, expectedJSON: `{"type":"conflict"}`},
		{name: "OwnType", typ: MustNewType("license-unavailable", http.StatusUnavailableForLegalReasons, false), expectedJSON: `{"type":"license-unavailable"}`},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			body, err := json.Marshal(struct {
				Type Type `json:"type"`
			}{Type: testCase.typ})
			require.NoError(t, err)

			assert.JSONEq(t, testCase.expectedJSON, string(body))
		})
	}
}
