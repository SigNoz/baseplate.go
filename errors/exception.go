package errors

import (
	"errors"
	"slices"
	"time"
)

// Exception is the public form of an error, as written in the error body of
// an API response. Errors and Suggestions are never null.
type Exception struct {
	Type        string                `json:"type" required:"true" nullable:"false"`
	Code        string                `json:"code" required:"true" nullable:"false"`
	Message     string                `json:"message" required:"true" nullable:"false"`
	URL         string                `json:"url,omitempty" required:"false"`
	Errors      []ExceptionAdditional `json:"errors" required:"true" nullable:"false"` // The error's additionals.
	Retry       *ExceptionRetry       `json:"retry,omitempty" required:"false"`
	Suggestions []string              `json:"suggestions" required:"true" nullable:"false"`
}

// ExceptionAdditional is one additional of an [Exception].
type ExceptionAdditional struct {
	Message     string   `json:"message" required:"true" nullable:"false"`
	Suggestions []string `json:"suggestions" required:"true" nullable:"false"`
}

// ExceptionRetry tells the client when to retry, and is set on an [Exception]
// only when the error has a retry delay.
type ExceptionRetry struct {
	Delay time.Duration `json:"delay" required:"true" nullable:"false"` // In nanoseconds.
}

// AsException converts err to its public form, built from the first typed
// error in its chain. The message is that error's own message without its
// cause, so wrapped causes don't reach the client. An untyped err has
// TypeUnknown, CodeUnknown and its full text as the message. Don't pass a nil
// err.
func AsException(err error) *Exception {
	b, ok := errors.AsType[*base](err)
	if !ok {
		return &Exception{
			Type:        TypeUnknown.String(),
			Code:        CodeUnknown.String(),
			Message:     err.Error(),
			Errors:      []ExceptionAdditional{},
			Suggestions: []string{},
		}
	}

	additionals := make([]ExceptionAdditional, len(b.additionals))
	for i, additional := range b.additionals {
		additionals[i] = ExceptionAdditional{Message: additional.Message, Suggestions: cloneNonNil(additional.Suggestions)}
	}

	var retry *ExceptionRetry
	if b.retryAfter > 0 {
		retry = &ExceptionRetry{Delay: b.retryAfter}
	}

	return &Exception{
		Type:        b.typ.String(),
		Code:        b.code.String(),
		Message:     b.Message(),
		URL:         b.url,
		Errors:      additionals,
		Retry:       retry,
		Suggestions: cloneNonNil(b.suggestions),
	}
}

func cloneNonNil(s []string) []string {
	if s == nil {
		return []string{}
	}

	return slices.Clone(s)
}
