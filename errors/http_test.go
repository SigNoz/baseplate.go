package errors

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAsJSON(t *testing.T) {
	testCases := []struct {
		name         string
		err          error
		expectedJSON string
	}{
		{
			name:         "Base_EmptyListsNotNull",
			err:          New(TypeNotFound, CodeNotFound, "dashboard not found"),
			expectedJSON: `{"type":"not-found","code":"not_found","message":"dashboard not found","errors":[],"suggestions":[]}`,
		},
		{
			name: "Base_AllDetails",
			err: New(TypeTooManyRequests, MustNewCode("rate_limited"), "slow down").
				WithURL("https://signoz.io/docs/limits").
				WithAdditional("100 requests per minute").
				WithSuggestiveAdditional("key servce", "did you mean: service").
				WithSuggestions("batch the requests").
				WithRetryAfter(2 * time.Second),
			expectedJSON: `{"type":"too-many-requests","code":"rate_limited","message":"slow down","url":"https://signoz.io/docs/limits",` +
				`"errors":[{"message":"100 requests per minute","suggestions":[]},{"message":"key servce","suggestions":["did you mean: service"]}],` +
				`"retry":{"delay":2000000000},"suggestions":["batch the requests"]}`,
		},
		{
			name:         "Wrapped_OwnMessageOnly",
			err:          Wrap(context.Canceled, TypeCanceled, CodeCanceled, "query canceled"),
			expectedJSON: `{"type":"canceled","code":"canceled","message":"query canceled","errors":[],"suggestions":[]}`,
		},
		{
			name:         "Foreign_UnknownWithFullText",
			err:          context.DeadlineExceeded,
			expectedJSON: `{"type":"unknown","code":"unknown","message":"context deadline exceeded","errors":[],"suggestions":[]}`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			body, err := json.Marshal(AsJSON(testCase.err))
			require.NoError(t, err)

			assert.JSONEq(t, testCase.expectedJSON, string(body))
		})
	}
}

func TestAsJSON_DoesNotShareSlices(t *testing.T) {
	err := New(TypeInvalidInput, CodeInvalidInput, "invalid query").
		WithSuggestiveAdditional("key servce", "did you mean: service").
		WithSuggestions("check the syntax")

	body := AsJSON(err)
	require.Len(t, body.Errors, 1)
	require.Len(t, body.Errors[0].Suggestions, 1)
	require.Len(t, body.Suggestions, 1)

	body.Errors[0].Suggestions[0] = "changed"
	body.Suggestions[0] = "changed"

	assert.Equal(t, []Additional{{Message: "key servce", Suggestions: []string{"did you mean: service"}}}, err.Additionals())
	assert.Equal(t, []string{"check the syntax"}, err.Suggestions())
}

func TestAsURLValues(t *testing.T) {
	values := AsURLValues(New(TypeUnauthenticated, MustNewCode("sso_failed"), "sign-in failed").WithURL("https://signoz.io/docs/sso").WithAdditional("state expired"))

	assert.Equal(t, "sso_failed", values.Get("code"))
	assert.Equal(t, "sign-in failed", values.Get("message"))
	assert.Equal(t, "https://signoz.io/docs/sso", values.Get("url"))
	assert.JSONEq(t, `[{"message":"state expired","suggestions":[]}]`, values.Get("errors"))
}

func TestHTTPStatus_TypeFromHTTPStatus(t *testing.T) {
	testCases := []struct {
		name           string
		t              typ
		expectedStatus int
		expectedType   typ
	}{
		{name: "InvalidInput", t: TypeInvalidInput, expectedStatus: http.StatusBadRequest, expectedType: TypeInvalidInput},
		{name: "Unauthenticated", t: TypeUnauthenticated, expectedStatus: http.StatusUnauthorized, expectedType: TypeUnauthenticated},
		{name: "Forbidden", t: TypeForbidden, expectedStatus: http.StatusForbidden, expectedType: TypeForbidden},
		{name: "NotFound", t: TypeNotFound, expectedStatus: http.StatusNotFound, expectedType: TypeNotFound},
		{name: "MethodNotAllowed", t: TypeMethodNotAllowed, expectedStatus: http.StatusMethodNotAllowed, expectedType: TypeMethodNotAllowed},
		{name: "Conflict", t: TypeConflict, expectedStatus: http.StatusConflict, expectedType: TypeConflict},
		{name: "TooManyRequests", t: TypeTooManyRequests, expectedStatus: http.StatusTooManyRequests, expectedType: TypeTooManyRequests},
		{name: "LicenseUnavailable", t: TypeLicenseUnavailable, expectedStatus: http.StatusUnavailableForLegalReasons, expectedType: TypeLicenseUnavailable},
		{name: "Canceled", t: TypeCanceled, expectedStatus: 499, expectedType: TypeCanceled},
		{name: "Unsupported", t: TypeUnsupported, expectedStatus: http.StatusNotImplemented, expectedType: TypeUnsupported},
		{name: "Unavailable", t: TypeUnavailable, expectedStatus: http.StatusServiceUnavailable, expectedType: TypeUnavailable},
		{name: "Timeout", t: TypeTimeout, expectedStatus: http.StatusGatewayTimeout, expectedType: TypeTimeout},
		{name: "Internal", t: TypeInternal, expectedStatus: http.StatusInternalServerError, expectedType: TypeInternal},
		{name: "Fatal_Internal", t: TypeFatal, expectedStatus: http.StatusInternalServerError, expectedType: TypeInternal},
		{name: "Unknown_Internal", t: TypeUnknown, expectedStatus: http.StatusInternalServerError, expectedType: TypeInternal},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			status := HTTPStatus(New(testCase.t, CodeUnknown, "status"))

			assert.Equal(t, testCase.expectedStatus, status)
			assert.Equal(t, testCase.expectedType, TypeFromHTTPStatus(status))
		})
	}
}

func TestHTTPStatus_Foreign_InternalServerError(t *testing.T) {
	assert.Equal(t, http.StatusInternalServerError, HTTPStatus(context.DeadlineExceeded))
}

func TestTypeFromHTTPStatus_Unmapped(t *testing.T) {
	testCases := []struct {
		name         string
		status       int
		expectedType typ
	}{
		{name: "BadGateway_Internal", status: http.StatusBadGateway, expectedType: TypeInternal},
		{name: "Teapot_Unknown", status: http.StatusTeapot, expectedType: TypeUnknown},
		{name: "Redirect_Unknown", status: http.StatusFound, expectedType: TypeUnknown},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expectedType, TypeFromHTTPStatus(testCase.status))
		})
	}
}
