package errors

import (
	"errors"
	"fmt"
	"slices"
	"time"
)

type base struct {
	typ         Type
	code        Code
	message     string
	cause       error
	url         string
	additionals []Additional
	suggestions []string
	retryAfter  time.Duration
	stacktrace  fmt.Stringer
}

// Additional is one detail of an error, such as one invalid field of a
// request.
type Additional struct {
	Message     string
	Suggestions []string // Relate to this detail alone, such as "did you mean: service".
}

// Error returns the error's message followed by ": " and its cause, or the
// message alone when there is no cause. An error from [WithAdditionalf] has no
// message of its own and returns its cause's text.
func (b *base) Error() string {
	switch {
	case b.cause == nil:
		return b.message
	case b.message == "":
		return b.cause.Error()
	}

	return b.message + ": " + b.cause.Error()
}

// Type returns the error's type.
func (b *base) Type() Type {
	return b.typ
}

// Code returns the error's code.
func (b *base) Code() Code {
	return b.code
}

// Message returns the error's own message without its cause, which is what an
// API response shows. An error with no message of its own returns the message
// of the first typed error in its cause's chain, or the cause's text when there
// is none.
func (b *base) Message() string {
	if b.message != "" || b.cause == nil {
		return b.message
	}

	if inner, ok := errors.AsType[*base](b.cause); ok {
		return inner.Message()
	}

	return b.cause.Error()
}

// Unwrap returns the error's cause, or nil when it has none.
func (b *base) Unwrap() error {
	return b.cause
}

// URL returns the documentation URL set by WithURL, or "" when there is none.
func (b *base) URL() string {
	return b.url
}

// Additionals returns a copy of the error's details, in the order they were
// added.
func (b *base) Additionals() []Additional {
	additionals := make([]Additional, len(b.additionals))
	for i, additional := range b.additionals {
		additionals[i] = Additional{Message: additional.Message, Suggestions: slices.Clone(additional.Suggestions)}
	}

	return additionals
}

// Suggestions returns a copy of the suggestions that relate to the error as a
// whole.
func (b *base) Suggestions() []string {
	return slices.Clone(b.suggestions)
}

// RetryAfter returns the delay set by WithRetryAfter, or zero when there is
// none.
func (b *base) RetryAfter() time.Duration {
	return b.retryAfter
}

// Stacktrace returns the stack where the error was created, one function and
// file:line pair per frame, or the stack set by WithStacktrace.
func (b *base) Stacktrace() string {
	return b.stacktrace.String()
}

// WithURL returns a copy of the error with a documentation URL for the
// failure.
func (b *base) WithURL(url string) *base {
	copied := *b
	copied.url = url

	return &copied
}

// WithAdditional returns a copy of the error with a detail added, along with
// the suggestions that relate to that detail alone.
func (b *base) WithAdditional(message string, suggestions ...string) *base {
	copied := *b
	copied.additionals = append(slices.Clone(b.additionals), Additional{Message: message, Suggestions: slices.Clone(suggestions)})

	return &copied
}

// WithSuggestions returns a copy of the error whose suggestions for the error
// as a whole are replaced by suggestions.
func (b *base) WithSuggestions(suggestions ...string) *base {
	copied := *b
	copied.suggestions = slices.Clone(suggestions)

	return &copied
}

// WithRetryAfter returns a copy of the error that tells the caller to retry
// after delay. [AsException] includes it, and an HTTP handler can send it
// as the Retry-After header.
func (b *base) WithRetryAfter(delay time.Duration) *base {
	copied := *b
	copied.retryAfter = delay

	return &copied
}

// WithStacktrace returns a copy of the error whose stack trace is replaced by
// stacktrace, such as the stack of a recovered panic from [runtime.Stack].
func (b *base) WithStacktrace(stacktrace string) *base {
	copied := *b
	copied.stacktrace = rawStacktrace(stacktrace)

	return &copied
}
