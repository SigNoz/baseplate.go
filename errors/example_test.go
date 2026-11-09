package errors_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/SigNoz/baseplate.go/errors"
)

// A package declares the codes and the types of its own errors.
var (
	CodeBackupMissing      = errors.MustNewCode("backup_missing")
	CodeFilterInvalid      = errors.MustNewCode("filter_invalid")
	CodeLicenseExpired     = errors.MustNewCode("license_expired")
	CodeLogsQueryFailed    = errors.MustNewCode("logs_query_failed")
	CodeRateLimited        = errors.MustNewCode("rate_limited")
	CodeUserReadFailed     = errors.MustNewCode("user_read_failed")
	TypeLicenseUnavailable = errors.MustNewType("license-unavailable", http.StatusUnavailableForLegalReasons, false)
)

func Example() {
	err := errors.Wrapf(fs.ErrNotExist, errors.TypeNotFound, CodeBackupMissing, "no backup of %s", "db_1")

	_, _ = fmt.Println(err)
	_, _ = fmt.Println(errors.TypeOf(err), errors.CodeOf(err))
	_, _ = fmt.Println(errors.IsNotFound(err), errors.Is(err, fs.ErrNotExist))
	// Output:
	// no backup of db_1: file does not exist
	// not-found backup_missing
	// true true
}

func ExampleNew() {
	err := errors.New(TypeLicenseUnavailable, CodeLicenseExpired, "license expired on 2026-10-01")

	_, _ = fmt.Println(err)
	_, _ = fmt.Println(errors.Asc(err, CodeLicenseExpired), errors.HTTPStatus(err), errors.IsRetryable(err))
	// Output:
	// license expired on 2026-10-01
	// true 451 false
}

func ExampleNew_details() {
	err := errors.New(errors.TypeInvalidInput, CodeFilterInvalid, "invalid filter").
		WithURL("https://signoz.io/docs/filters").
		WithAdditional("operator ~ is unsupported").
		WithAdditional("key servce is unknown", "did you mean: service").
		WithSuggestions("check the filter syntax")

	_, _ = fmt.Println(err.Message())
	_, _ = fmt.Println(err.URL())
	_, _ = fmt.Println(err.Additionals())
	_, _ = fmt.Println(err.Suggestions())
	// Output:
	// invalid filter
	// https://signoz.io/docs/filters
	// [{operator ~ is unsupported []} {key servce is unknown [did you mean: service]}]
	// [check the filter syntax]
}

func ExampleNewType() {
	typ, err := errors.NewType("license-unavailable", http.StatusUnavailableForLegalReasons, false)
	if err != nil {
		panic(err)
	}

	_, _ = fmt.Println(typ)
	// Output:
	// license-unavailable
}

func ExampleWrap() {
	cause := errors.New(errors.TypeTooManyRequests, CodeRateLimited, "rate limited").WithRetryAfter(2 * time.Second)

	err := errors.Wrap(cause, errors.TypeUnavailable, CodeLogsQueryFailed, "cannot query logs")

	_, _ = fmt.Println(err)
	_, _ = fmt.Println(err.Message())
	_, _ = fmt.Println(errors.TypeOf(err), err.RetryAfter())
	// Output:
	// cannot query logs: rate limited
	// cannot query logs
	// unavailable 2s
}

// An untyped error becomes TypeInternal and CodeUnknown, and its text stays the
// same.
func ExampleWithAdditionalf() {
	err := errors.WithAdditionalf(fs.ErrPermission, "cannot delete user %s", "u_1")

	_, _ = fmt.Println(err)
	_, _ = fmt.Println(errors.TypeOf(err), errors.CodeOf(err))
	_, _ = fmt.Println(err.Additionals())
	// Output:
	// permission denied
	// internal unknown
	// [{cannot delete user u_1 []}]
}

func ExampleTypeOf() {
	typed := errors.Join(context.Canceled, errors.New(TypeLicenseUnavailable, CodeLicenseExpired, "license expired"))
	untyped := context.DeadlineExceeded

	_, _ = fmt.Println(errors.TypeOf(typed))
	_, _ = fmt.Println(errors.TypeOf(untyped))
	// Output:
	// license-unavailable
	// unknown
}

func ExampleMessageOf() {
	err := errors.Wrap(fs.ErrClosed, errors.TypeInternal, CodeUserReadFailed, "cannot read user")

	_, _ = fmt.Println(errors.MessageOf(err))
	_, _ = fmt.Println(errors.MessageOf(fs.ErrClosed))
	// Output:
	// cannot read user
	// file already closed
}

func ExampleIsRetryable() {
	_, _ = fmt.Println(errors.IsRetryable(errors.New(errors.TypeTooManyRequests, CodeRateLimited, "rate limited")))
	_, _ = fmt.Println(errors.IsRetryable(errors.New(errors.TypeInvalidInput, CodeFilterInvalid, "invalid filter")))
	_, _ = fmt.Println(errors.IsRetryable(context.DeadlineExceeded))
	// Output:
	// true
	// false
	// true
}

func ExampleNewCode() {
	_, err := errors.NewCode("not-found")

	_, _ = fmt.Println(err)
	// Output:
	// code "not-found" is invalid, expected a letter followed by letters, digits or underscores
}

// The handler drops the time and the stack trace to keep the output stable.
func ExampleAttr() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey || a.Key == "stacktrace" {
				return slog.Attr{}
			}

			return a
		},
	}))

	logger.Error("cannot restore", errors.Attr(errors.New(errors.TypeNotFound, CodeBackupMissing, "no backup of db_1")))
	// Output:
	// level=ERROR msg="cannot restore" exception.type=not-found exception.code=backup_missing exception.message="no backup of db_1"
}

func ExampleAsException() {
	err := errors.Wrap(context.Canceled, errors.TypeCanceled, CodeLogsQueryFailed, "query canceled").
		WithRetryAfter(time.Second)

	body, marshalErr := json.Marshal(errors.AsException(err))
	if marshalErr != nil {
		panic(marshalErr)
	}

	_, _ = fmt.Println(string(body))
	// Output:
	// {"type":"canceled","code":"logs_query_failed","message":"query canceled","errors":[],"retry":{"delay":1000000000},"suggestions":[]}
}

func ExampleHTTPStatus() {
	_, _ = fmt.Println(errors.HTTPStatus(errors.New(errors.TypeNotFound, CodeBackupMissing, "no backup of db_1")))
	_, _ = fmt.Println(errors.HTTPStatus(errors.New(TypeLicenseUnavailable, CodeLicenseExpired, "license expired")))
	_, _ = fmt.Println(errors.HTTPStatus(context.DeadlineExceeded))
	// Output:
	// 404
	// 451
	// 500
}

func ExampleTypeFromHTTPStatus() {
	_, _ = fmt.Println(errors.TypeFromHTTPStatus(http.StatusTooManyRequests))
	_, _ = fmt.Println(errors.TypeFromHTTPStatus(http.StatusBadGateway))
	// Output:
	// too-many-requests
	// internal
}
