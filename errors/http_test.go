package errors

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
		t              Type
		expectedStatus int
		expectedType   Type
	}{
		{name: "InvalidInput", t: TypeInvalidInput, expectedStatus: http.StatusBadRequest, expectedType: TypeInvalidInput},
		{name: "Unauthenticated", t: TypeUnauthenticated, expectedStatus: http.StatusUnauthorized, expectedType: TypeUnauthenticated},
		{name: "Forbidden", t: TypeForbidden, expectedStatus: http.StatusForbidden, expectedType: TypeForbidden},
		{name: "NotFound", t: TypeNotFound, expectedStatus: http.StatusNotFound, expectedType: TypeNotFound},
		{name: "MethodNotAllowed", t: TypeMethodNotAllowed, expectedStatus: http.StatusMethodNotAllowed, expectedType: TypeMethodNotAllowed},
		{name: "Conflict", t: TypeConflict, expectedStatus: http.StatusConflict, expectedType: TypeConflict},
		{name: "TooManyRequests", t: TypeTooManyRequests, expectedStatus: http.StatusTooManyRequests, expectedType: TypeTooManyRequests},
		{name: "OwnType_ItsStatus_ReverseUnknown", t: MustNewType("license-unavailable", http.StatusUnavailableForLegalReasons, false), expectedStatus: http.StatusUnavailableForLegalReasons, expectedType: TypeUnknown},
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
		expectedType Type
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
