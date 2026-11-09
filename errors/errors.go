package errors

import (
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"
)

// New returns an error of type t and code c with message, and records the
// stack where it is called.
func New(t typ, c Code, message string) *base {
	return &base{t: t, c: c, message: message, s: newStacktrace()}
}

// Newf is [New] with a message formatted as in [fmt.Sprintf].
func Newf(t typ, c Code, format string, args ...any) *base {
	return &base{t: t, c: c, message: fmt.Sprintf(format, args...), s: newStacktrace()}
}

// Wrap returns an error of type t and code c that wraps cause with message,
// and records the stack where it is called. Its text is "message: cause". It
// keeps the details, suggestions and retry delay of the first typed error in
// cause's chain, but not its URL.
func Wrap(cause error, t typ, c Code, message string) *base {
	return wrap(&base{t: t, c: c, message: message, cause: cause, s: newStacktrace()})
}

// Wrapf is [Wrap] with a message formatted as in [fmt.Sprintf].
func Wrapf(cause error, t typ, c Code, format string, args ...any) *base {
	return wrap(&base{t: t, c: c, message: fmt.Sprintf(format, args...), cause: cause, s: newStacktrace()})
}

// WithAdditionalf wraps err with a detail formatted as in [fmt.Sprintf],
// leaving its text and message unchanged. It keeps the type, code, URL and
// stack trace of the first typed error in err's chain. An untyped err becomes
// TypeInternal and CodeInternal, with the stack where WithAdditionalf is
// called.
func WithAdditionalf(err error, format string, args ...any) *base {
	b := &base{t: TypeInternal, c: CodeInternal, cause: err, s: newStacktrace()}
	if inner, ok := errors.AsType[*base](err); ok {
		b.t = inner.t
		b.c = inner.c
		b.url = inner.url
		b.s = inner.s
	}

	return wrap(b).WithAdditional(fmt.Sprintf(format, args...))
}

// TypeOf returns the type of the first typed error in err's chain, or
// TypeUnknown when there is none or err is nil.
func TypeOf(err error) typ {
	if b, ok := errors.AsType[*base](err); ok {
		return b.t
	}

	return TypeUnknown
}

// CodeOf returns the code of the first typed error in err's chain, or
// CodeUnknown when there is none or err is nil.
func CodeOf(err error) Code {
	if b, ok := errors.AsType[*base](err); ok {
		return b.c
	}

	return CodeUnknown
}

// MessageOf returns the message, without its cause, of the first typed error
// in err's chain. It returns err's full text when err is untyped, and "" when
// err is nil.
func MessageOf(err error) string {
	if err == nil {
		return ""
	}

	if b, ok := errors.AsType[*base](err); ok {
		return b.Message()
	}

	return err.Error()
}

// RetryAfterOf returns the retry delay of the first typed error in err's
// chain, or zero when there is none or err is nil.
func RetryAfterOf(err error) time.Duration {
	if b, ok := errors.AsType[*base](err); ok {
		return b.retryAfter
	}

	return 0
}

// Ast reports whether the type of err, as returned by [TypeOf], is t.
func Ast(err error, t typ) bool {
	return TypeOf(err) == t
}

// Asc reports whether the code of err, as returned by [CodeOf], is c.
func Asc(err error, c Code) bool {
	return CodeOf(err) == c
}

// IsNotFound reports whether the type of err is TypeNotFound.
func IsNotFound(err error) bool {
	return Ast(err, TypeNotFound)
}

// IsConflict reports whether the type of err is TypeConflict.
func IsConflict(err error) bool {
	return Ast(err, TypeConflict)
}

// IsUnauthenticated reports whether the type of err is TypeUnauthenticated.
func IsUnauthenticated(err error) bool {
	return Ast(err, TypeUnauthenticated)
}

// IsForbidden reports whether the type of err is TypeForbidden.
func IsForbidden(err error) bool {
	return Ast(err, TypeForbidden)
}

// IsUnavailable reports whether the type of err is TypeUnavailable.
func IsUnavailable(err error) bool {
	return Ast(err, TypeUnavailable)
}

// IsRetryable reports whether retrying the operation that returned err may
// succeed. It is false for nil and for the types that need a change to the
// request, credentials or system first: TypeInvalidInput, TypeUnauthenticated,
// TypeForbidden, TypeMethodNotAllowed, TypeConflict, TypeLicenseUnavailable,
// TypeCanceled and TypeUnsupported. An untyped error is retryable, because its
// outcome is unknown.
func IsRetryable(err error) bool {
	return err != nil && TypeOf(err).retryable()
}

// Is calls [errors.Is] from the standard library.
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// As calls [errors.As] from the standard library.
func As(err error, target any) bool {
	return errors.As(err, target)
}

// AsType calls [errors.AsType] from the standard library.
func AsType[E error](err error) (E, bool) {
	return errors.AsType[E](err)
}

// Join calls [errors.Join] from the standard library.
func Join(errs ...error) error {
	return errors.Join(errs...)
}

// Attr returns err as an "exception" group for [log/slog] with type, code,
// message and stacktrace attributes, which a text handler writes as
// exception.type=not-found and so on. The message is err's full text. An
// untyped error gives its Go type and message only, and a nil err gives an
// empty group, which handlers drop.
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
