package errors

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"time"
)

const (
	// The statusClientClosedRequest code is nginx's non-standard status for a
	// client that went away before the response.
	statusClientClosedRequest int = 499
)

// JSON is the error body of an HTTP API response, built by [AsJSON]. Errors
// and Suggestions are never null.
type JSON struct {
	Type        string           `json:"type" required:"true" nullable:"false"`
	Code        string           `json:"code" required:"true" nullable:"false"`
	Message     string           `json:"message" required:"true" nullable:"false"`
	URL         string           `json:"url,omitempty" required:"false"`
	Errors      []JSONAdditional `json:"errors" required:"true" nullable:"false"`
	Retry       *JSONRetry       `json:"retry,omitempty" required:"false"`
	Suggestions []string         `json:"suggestions" required:"true" nullable:"false"`
}

// JSONAdditional is one detail of a [JSON] error body.
type JSONAdditional struct {
	Message     string   `json:"message" required:"true" nullable:"false"`
	Suggestions []string `json:"suggestions" required:"true" nullable:"false"`
}

// JSONRetry tells the client when to retry, and is set only when the error has
// a retry delay.
type JSONRetry struct {
	Delay time.Duration `json:"delay" required:"true" nullable:"false"` // In nanoseconds.
}

// AsJSON returns the error body for err, built from the first typed error in
// its chain. The message is that error's own message without its cause, so
// wrapped causes don't reach the client. An untyped err has TypeUnknown,
// CodeUnknown and its full text as the message. Don't pass a nil err.
func AsJSON(err error) *JSON {
	b, ok := errors.AsType[*base](err)
	if !ok {
		return &JSON{
			Type:        TypeUnknown.String(),
			Code:        CodeUnknown.String(),
			Message:     err.Error(),
			Errors:      []JSONAdditional{},
			Suggestions: []string{},
		}
	}

	errs := make([]JSONAdditional, len(b.additionals))
	for i, additional := range b.additionals {
		errs[i] = JSONAdditional{Message: additional.Message, Suggestions: nonNil(additional.Suggestions)}
	}

	var retry *JSONRetry
	if b.retryAfter > 0 {
		retry = &JSONRetry{Delay: b.retryAfter}
	}

	return &JSON{
		Type:        b.t.String(),
		Code:        b.c.String(),
		Message:     b.Message(),
		URL:         b.url,
		Errors:      errs,
		Retry:       retry,
		Suggestions: nonNil(b.suggestions),
	}
}

// AsURLValues returns err's code, message and url, and its details as a JSON
// array under errors, for the query string of a redirect that reports an
// error. It reads err the way [AsJSON] does.
func AsURLValues(err error) url.Values {
	body := AsJSON(err)

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

// HTTPStatus returns the HTTP status code for the type of the first typed
// error in err's chain:
//
//	TypeInvalidInput        400 Bad Request
//	TypeUnauthenticated     401 Unauthorized
//	TypeForbidden           403 Forbidden
//	TypeNotFound            404 Not Found
//	TypeMethodNotAllowed    405 Method Not Allowed
//	TypeConflict            409 Conflict
//	TypeTooManyRequests     429 Too Many Requests
//	TypeLicenseUnavailable  451 Unavailable For Legal Reasons
//	TypeCanceled            499 Client Closed Request (nginx)
//	TypeUnsupported         501 Not Implemented
//	TypeUnavailable         503 Service Unavailable
//	TypeTimeout             504 Gateway Timeout
//
// Every other type, and an untyped error, is 500 Internal Server Error.
func HTTPStatus(err error) int {
	switch TypeOf(err) {
	case TypeInvalidInput:
		return http.StatusBadRequest
	case TypeUnauthenticated:
		return http.StatusUnauthorized
	case TypeForbidden:
		return http.StatusForbidden
	case TypeNotFound:
		return http.StatusNotFound
	case TypeMethodNotAllowed:
		return http.StatusMethodNotAllowed
	case TypeConflict:
		return http.StatusConflict
	case TypeTooManyRequests:
		return http.StatusTooManyRequests
	case TypeLicenseUnavailable:
		return http.StatusUnavailableForLegalReasons
	case TypeCanceled:
		return statusClientClosedRequest
	case TypeUnsupported:
		return http.StatusNotImplemented
	case TypeUnavailable:
		return http.StatusServiceUnavailable
	case TypeTimeout:
		return http.StatusGatewayTimeout
	}

	return http.StatusInternalServerError
}

// TypeFromHTTPStatus returns the type for an HTTP status code, the reverse of
// [HTTPStatus], for building an error from another service's response. An
// unmapped 5xx status is TypeInternal, and any other unmapped status is
// TypeUnknown.
func TypeFromHTTPStatus(status int) typ {
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
	case http.StatusUnavailableForLegalReasons:
		return TypeLicenseUnavailable
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

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}

	return s
}
