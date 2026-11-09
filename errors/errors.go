package errors

import (
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"
)

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

// WithAdditionalf wraps err to add a detail, keeping the type, code and stack trace of the first Base in its chain,
// or TypeInternal, CodeInternal and a new stack trace when there is none.
func WithAdditionalf(err error, format string, args ...any) *Base {
	base := &Base{t: TypeInternal, c: CodeInternal, cause: err, s: newStacktrace()}
	if inner, ok := errors.AsType[*Base](err); ok {
		base.t = inner.t
		base.c = inner.c
		base.s = inner.s
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
