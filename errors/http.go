package errors

import (
	"encoding/json"
	"net/http"
	"net/url"
)

const (
	// The statusClientClosedRequest code is nginx's non-standard status for a
	// client that went away before the response.
	statusClientClosedRequest int = 499
)

// AsURLValues returns err's code, message and url, and its details as a JSON
// array under errors, for the query string of a redirect that reports an
// error. It reads err the way [AsException] does.
func AsURLValues(err error) url.Values {
	body := AsException(err)

	values := url.Values{
		"code":    {body.Code},
		"message": {body.Message},
		"url":     {body.URL},
	}

	errs, marshalErr := json.Marshal(body.Errors)
	if marshalErr == nil {
		values.Set("errors", string(errs))
	}

	return values
}

// HTTPStatus returns the HTTP status code of the type of the first typed error
// in err's chain, which [NewType] sets for a package's own types. The base
// types map to the matching 4xx and 5xx statuses, with TypeCanceled as nginx's
// 499. TypeInternal, TypeFatal, TypeUnknown and an untyped error are 500.
func HTTPStatus(err error) int {
	return TypeOf(err).status
}

// TypeFromHTTPStatus returns the base type for an HTTP status code, the reverse
// of [HTTPStatus], for building an error from another service's response. An
// unmapped 5xx status is TypeInternal, and any other unmapped status is
// TypeUnknown.
func TypeFromHTTPStatus(status int) Type {
	switch status {
	case http.StatusBadRequest:
		return TypeInvalidInput
	case http.StatusUnauthorized:
		return TypeUnauthenticated
	case http.StatusForbidden:
		return TypeForbidden
	case http.StatusNotFound:
		return TypeNotFound
	case http.StatusMethodNotAllowed:
		return TypeMethodNotAllowed
	case http.StatusConflict:
		return TypeConflict
	case http.StatusTooManyRequests:
		return TypeTooManyRequests
	case statusClientClosedRequest:
		return TypeCanceled
	case http.StatusNotImplemented:
		return TypeUnsupported
	case http.StatusServiceUnavailable:
		return TypeUnavailable
	case http.StatusGatewayTimeout:
		return TypeTimeout
	}

	if status >= http.StatusInternalServerError {
		return TypeInternal
	}

	return TypeUnknown
}
