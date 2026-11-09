package errors

import (
	"errors"
	"fmt"
	"slices"
	"time"
)

type base struct {
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
func (b *base) Error() string {
	switch {
	case b.cause == nil:
		return b.message
	case b.message == "":
		return b.cause.Error()
	}

	return b.message + ": " + b.cause.Error()
}

func (b *base) Type() typ {
	return b.t
}

func (b *base) Code() Code {
	return b.c
}

// Message is the error's own message without its cause. Without one, it is the message of the first typed error
// in the cause's chain, or the cause's text when there is none.
func (b *base) Message() string {
	if b.message != "" || b.cause == nil {
		return b.message
	}

	if inner, ok := errors.AsType[*base](b.cause); ok {
		return inner.Message()
	}

	return b.cause.Error()
}

func (b *base) Unwrap() error {
	return b.cause
}

func (b *base) URL() string {
	return b.url
}

func (b *base) Additionals() []Additional {
	return slices.Clone(b.additionals)
}

// Suggestions relate to the error as a whole, while an Additional's suggestions relate to that detail.
func (b *base) Suggestions() []string {
	return slices.Clone(b.suggestions)
}

// RetryAfter is zero unless WithRetryAfter set it.
func (b *base) RetryAfter() time.Duration {
	return b.retryAfter
}

// Stacktrace is where the error was created, unless WithStacktrace replaced it.
func (b *base) Stacktrace() string {
	return b.s.String()
}

func (b *base) WithURL(url string) *base {
	copied := *b
	copied.url = url

	return &copied
}

func (b *base) WithAdditional(messages ...string) *base {
	copied := *b
	copied.additionals = slices.Clone(b.additionals)

	for _, message := range messages {
		copied.additionals = append(copied.additionals, Additional{Message: message})
	}

	return &copied
}

func (b *base) WithSuggestiveAdditional(message string, suggestions ...string) *base {
	copied := *b
	copied.additionals = append(slices.Clone(b.additionals), Additional{Message: message, Suggestions: slices.Clone(suggestions)})

	return &copied
}

// WithSuggestions replaces the error's suggestions.
func (b *base) WithSuggestions(suggestions ...string) *base {
	copied := *b
	copied.suggestions = slices.Clone(suggestions)

	return &copied
}

func (b *base) WithRetryAfter(delay time.Duration) *base {
	copied := *b
	copied.retryAfter = delay

	return &copied
}

// WithStacktrace replaces the captured stack trace, for example with a recovered panic's runtime.Stack.
func (b *base) WithStacktrace(stacktrace string) *base {
	copied := *b
	copied.s = rawStacktrace(stacktrace)

	return &copied
}
