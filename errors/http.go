package errors

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"slices"
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
	Type        string                    `json:"type" required:"true" nullable:"false"`
	Code        string                    `json:"code" required:"true" nullable:"false"`
	Message     string                    `json:"message" required:"true" nullable:"false"`
	URL         string                    `json:"url,omitempty" required:"false"`
	Errors      []responseerroradditional `json:"errors" required:"true" nullable:"false"` // The error's additionals.
	Retry       *responseretryjson        `json:"retry,omitempty" required:"false"`        // Set only when the error has a retry delay.
	Suggestions []string                  `json:"suggestions" required:"true" nullable:"false"`
}

// The nested types keep SigNoz's names, so the schemas a generator derives from
// them don't change.
type responseerroradditional struct {
	Message     string   `json:"message" required:"true" nullable:"false"`
	Suggestions []string `json:"suggestions" required:"true" nullable:"false"`
}

type responseretryjson struct {
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
			Errors:      []responseerroradditional{},
			Suggestions: []string{},
		}
	}

	errs := make([]responseerroradditional, len(b.additionals))
	for i, additional := range b.additionals {
		errs[i] = responseerroradditional{Message: additional.Message, Suggestions: cloneNonNil(additional.Suggestions)}
	}

	var retry *responseretryjson
	if b.retryAfter > 0 {
		retry = &responseretryjson{Delay: b.retryAfter}
	}

	return &JSON{
		Type:        b.typ.String(),
		Code:        b.code.String(),
		Message:     b.Message(),
		URL:         b.url,
		Errors:      errs,
		Retry:       retry,
		Suggestions: cloneNonNil(b.suggestions),
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

func cloneNonNil(s []string) []string {
	if s == nil {
		return []string{}
	}

	return slices.Clone(s)
}
