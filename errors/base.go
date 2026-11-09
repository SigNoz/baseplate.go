package errors

import (
	"errors"
	"fmt"
	"slices"
	"time"
)

type Base struct {
	t           typ
	c           Code
	message     string
	cause       error
	url         string
	additionals []Additional
	suggestions []string
	retryAfter  time.Duration
	s           fmt.Stringer
}

// Additional is one detail of an error, with the suggestions that relate to it alone.
type Additional struct {
	Message     string
	Suggestions []string
}

// An empty message, which only WithAdditionalf creates, shows the cause alone.
func (b *Base) Error() string {
	switch {
	case b.cause == nil:
		return b.message
	case b.message == "":
		return b.cause.Error()
	}

	return b.message + ": " + b.cause.Error()
}

func (b *Base) Type() typ {
	return b.t
}

func (b *Base) Code() Code {
	return b.c
}

// Message is the error's own message without its cause. Without one, it is the message of the first Base
// in the cause's chain, or the cause's text when there is none.
func (b *Base) Message() string {
	if b.message != "" || b.cause == nil {
		return b.message
	}

	if inner, ok := errors.AsType[*Base](b.cause); ok {
		return inner.Message()
	}

	return b.cause.Error()
}

func (b *Base) Unwrap() error {
	return b.cause
}

func (b *Base) URL() string {
	return b.url
}

func (b *Base) Additionals() []Additional {
	return slices.Clone(b.additionals)
}

// Suggestions relate to the error as a whole, while an Additional's suggestions relate to that detail.
func (b *Base) Suggestions() []string {
	return slices.Clone(b.suggestions)
}

// RetryAfter is zero unless WithRetryAfter set it.
func (b *Base) RetryAfter() time.Duration {
	return b.retryAfter
}

// Stacktrace is where the error was created, unless WithStacktrace replaced it.
func (b *Base) Stacktrace() string {
	return b.s.String()
}

func (b *Base) WithURL(url string) *Base {
	copied := *b
	copied.url = url

	return &copied
}

func (b *Base) WithAdditional(messages ...string) *Base {
	copied := *b
	copied.additionals = slices.Clone(b.additionals)

	for _, message := range messages {
		copied.additionals = append(copied.additionals, Additional{Message: message})
	}

	return &copied
}

func (b *Base) WithSuggestiveAdditional(message string, suggestions ...string) *Base {
	copied := *b
	copied.additionals = append(slices.Clone(b.additionals), Additional{Message: message, Suggestions: slices.Clone(suggestions)})

	return &copied
}

// WithSuggestions replaces the error's suggestions.
func (b *Base) WithSuggestions(suggestions ...string) *Base {
	copied := *b
	copied.suggestions = slices.Clone(suggestions)

	return &copied
}

func (b *Base) WithRetryAfter(delay time.Duration) *Base {
	copied := *b
	copied.retryAfter = delay

	return &copied
}

// WithStacktrace replaces the captured stack trace, for example with a recovered panic's runtime.Stack.
func (b *Base) WithStacktrace(stacktrace string) *Base {
	copied := *b
	copied.s = rawStacktrace(stacktrace)

	return &copied
}
