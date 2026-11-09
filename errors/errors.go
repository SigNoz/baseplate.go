package errors

import (
	"errors"
	"fmt"
	"log/slog"
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

// Message is the error's own message without its cause, or the cause's text when the error has no message.
func (b *Base) Message() string {
	if b.message == "" && b.cause != nil {
		return b.cause.Error()
	}

	return b.message
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

func New(t typ, c Code, message string) *Base {
	return &Base{t: t, c: c, message: message, s: newStacktrace()}
}

func Newf(t typ, c Code, format string, args ...any) *Base {
	return &Base{t: t, c: c, message: fmt.Sprintf(format, args...), s: newStacktrace()}
}

// Wrap keeps the additionals, suggestions and retry delay of the first Base in cause's chain.
func Wrap(cause error, t typ, c Code, message string) *Base {
	return wrap(&Base{t: t, c: c, message: message, cause: cause, s: newStacktrace()})
}

// Wrapf keeps the additionals, suggestions and retry delay of the first Base in cause's chain.
func Wrapf(cause error, t typ, c Code, format string, args ...any) *Base {
	return wrap(&Base{t: t, c: c, message: fmt.Sprintf(format, args...), cause: cause, s: newStacktrace()})
}

// WithAdditionalf adds a detail to err. A *Base keeps its stack trace. Any other error is wrapped
// with the type and code of the first Base in its chain, or TypeInternal and CodeInternal when there is none.
func WithAdditionalf(err error, format string, args ...any) *Base {
	if base, ok := err.(*Base); ok { //nolint:errorlint // Only err itself keeps its stack trace; anything wrapping a Base is wrapped again below.
		return base.WithAdditional(fmt.Sprintf(format, args...))
	}

	base := &Base{t: TypeInternal, c: CodeInternal, cause: err, s: newStacktrace()}
	if inner, ok := errors.AsType[*Base](err); ok {
		base.t = inner.t
		base.c = inner.c
	}

	return wrap(base).WithAdditional(fmt.Sprintf(format, args...))
}

// TypeOf returns the type of the first Base in err's chain, or TypeUnknown when there is none.
func TypeOf(err error) typ {
	if base, ok := errors.AsType[*Base](err); ok {
		return base.t
	}

	return TypeUnknown
}

// CodeOf returns the code of the first Base in err's chain, or CodeUnknown when there is none.
func CodeOf(err error) Code {
	if base, ok := errors.AsType[*Base](err); ok {
		return base.c
	}

	return CodeUnknown
}

// RetryAfterOf returns the retry delay of the first Base in err's chain, or zero when there is none.
func RetryAfterOf(err error) time.Duration {
	if base, ok := errors.AsType[*Base](err); ok {
		return base.retryAfter
	}

	return 0
}

func Ast(err error, t typ) bool {
	return TypeOf(err) == t
}

func Asc(err error, c Code) bool {
	return CodeOf(err) == c
}

func IsNotFound(err error) bool {
	return Ast(err, TypeNotFound)
}

func IsConflict(err error) bool {
	return Ast(err, TypeConflict)
}

func IsUnauthenticated(err error) bool {
	return Ast(err, TypeUnauthenticated)
}

func IsForbidden(err error) bool {
	return Ast(err, TypeForbidden)
}

func IsUnavailable(err error) bool {
	return Ast(err, TypeUnavailable)
}

// IsRetryable reports whether a retry may fix err. An error without a Base is retryable, because its outcome is unknown.
func IsRetryable(err error) bool {
	return err != nil && TypeOf(err).retryable()
}

func Is(err, target error) bool {
	return errors.Is(err, target)
}

func As(err error, target any) bool {
	return errors.As(err, target)
}

func AsType[E error](err error) (E, bool) {
	return errors.AsType[E](err)
}

func Join(errs ...error) error {
	return errors.Join(errs...)
}

// Attr logs err as exception.type, exception.code, exception.message and exception.stacktrace.
// An error without a Base logs its Go type and message only.
func Attr(err error) slog.Attr {
	if err == nil {
		return slog.GroupAttrs("exception")
	}

	base, ok := errors.AsType[*Base](err)
	if !ok {
		return slog.GroupAttrs("exception", slog.String("type", fmt.Sprintf("%T", err)), slog.String("message", err.Error()))
	}

	return slog.GroupAttrs(
		"exception",
		slog.String("type", base.t.String()),
		slog.String("code", base.c.String()),
		slog.String("message", err.Error()),
		slog.String("stacktrace", base.Stacktrace()),
	)
}

func wrap(base *Base) *Base {
	if inner, ok := errors.AsType[*Base](base.cause); ok {
		base.additionals = slices.Clone(inner.additionals)
		base.suggestions = slices.Clone(inner.suggestions)
		base.retryAfter = inner.retryAfter
	}

	return base
}
