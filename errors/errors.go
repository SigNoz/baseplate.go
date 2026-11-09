package errors

import (
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"
)

func New(t typ, c Code, message string) *base {
	return &base{t: t, c: c, message: message, s: newStacktrace()}
}

func Newf(t typ, c Code, format string, args ...any) *base {
	return &base{t: t, c: c, message: fmt.Sprintf(format, args...), s: newStacktrace()}
}

// Wrap keeps the additionals, suggestions and retry delay of the first typed error in cause's chain.
func Wrap(cause error, t typ, c Code, message string) *base {
	return wrap(&base{t: t, c: c, message: message, cause: cause, s: newStacktrace()})
}

// Wrapf keeps the additionals, suggestions and retry delay of the first typed error in cause's chain.
func Wrapf(cause error, t typ, c Code, format string, args ...any) *base {
	return wrap(&base{t: t, c: c, message: fmt.Sprintf(format, args...), cause: cause, s: newStacktrace()})
}

// WithAdditionalf wraps err to add a detail, keeping the type, code and stack trace of the first typed error in its chain,
// or TypeInternal, CodeInternal and a new stack trace when there is none.
func WithAdditionalf(err error, format string, args ...any) *base {
	b := &base{t: TypeInternal, c: CodeInternal, cause: err, s: newStacktrace()}
	if inner, ok := errors.AsType[*base](err); ok {
		b.t = inner.t
		b.c = inner.c
		b.s = inner.s
	}

	return wrap(b).WithAdditional(fmt.Sprintf(format, args...))
}

// TypeOf returns the type of the first typed error in err's chain, or TypeUnknown when there is none.
func TypeOf(err error) typ {
	if b, ok := errors.AsType[*base](err); ok {
		return b.t
	}

	return TypeUnknown
}

// CodeOf returns the code of the first typed error in err's chain, or CodeUnknown when there is none.
func CodeOf(err error) Code {
	if b, ok := errors.AsType[*base](err); ok {
		return b.c
	}

	return CodeUnknown
}

// RetryAfterOf returns the retry delay of the first typed error in err's chain, or zero when there is none.
func RetryAfterOf(err error) time.Duration {
	if b, ok := errors.AsType[*base](err); ok {
		return b.retryAfter
	}

	return 0
}

// MessageOf returns the message, without its cause, of the first typed error in err's chain, or err's full text when it is untyped.
func MessageOf(err error) string {
	if err == nil {
		return ""
	}

	if b, ok := errors.AsType[*base](err); ok {
		return b.Message()
	}

	return err.Error()
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

// IsRetryable reports whether a retry may fix err. An untyped error is retryable, because its outcome is unknown.
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
// An untyped error logs its Go type and message only.
func Attr(err error) slog.Attr {
	if err == nil {
		return slog.GroupAttrs("exception")
	}

	b, ok := errors.AsType[*base](err)
	if !ok {
		return slog.GroupAttrs("exception", slog.String("type", fmt.Sprintf("%T", err)), slog.String("message", err.Error()))
	}

	return slog.GroupAttrs(
		"exception",
		slog.String("type", b.t.String()),
		slog.String("code", b.c.String()),
		slog.String("message", err.Error()),
		slog.String("stacktrace", b.Stacktrace()),
	)
}

func wrap(b *base) *base {
	if inner, ok := errors.AsType[*base](b.cause); ok {
		b.additionals = slices.Clone(inner.additionals)
		b.suggestions = slices.Clone(inner.suggestions)
		b.retryAfter = inner.retryAfter
	}

	return b
}
