package errors

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"time"
)

const (
	// statusClientClosedRequest is nginx's non-standard status for a client that went away before the response.
	statusClientClosedRequest int = 499
)

// JSON is the error body of an HTTP API response.
type JSON struct {
	Type        string           `json:"type" required:"true" nullable:"false"`
	Code        string           `json:"code" required:"true" nullable:"false"`
	Message     string           `json:"message" required:"true" nullable:"false"`
	URL         string           `json:"url,omitempty" required:"false"`
	Errors      []JSONAdditional `json:"errors" required:"true" nullable:"false"`
	Retry       *JSONRetry       `json:"retry,omitempty" required:"false"`
	Suggestions []string         `json:"suggestions" required:"true" nullable:"false"`
}

type JSONAdditional struct {
	Message     string   `json:"message" required:"true" nullable:"false"`
	Suggestions []string `json:"suggestions" required:"true" nullable:"false"`
}

type JSONRetry struct {
	Delay time.Duration `json:"delay" required:"true" nullable:"false"` // In nanoseconds.
}

// AsJSON describes the first typed error in err's chain. An untyped error is TypeUnknown and CodeUnknown, with its full text as the message.
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

// AsURLValues carries the code, message, url and the additionals as JSON in errors, for a redirect's query string.
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

// HTTPStatus maps the type of the first typed error in err's chain to a status code. Anything unmapped is 500.
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

// TypeFromHTTPStatus is HTTPStatus in reverse, for errors read from a response. An unmapped 5xx is TypeInternal, anything else unmapped TypeUnknown.
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
