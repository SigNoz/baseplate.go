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

var (
	// CodeLicenseExpired is a code a package declares for itself.
	CodeLicenseExpired = errors.MustNewCode("license_expired")
)

func Example() {
	err := errors.Wrapf(fs.ErrNotExist, errors.TypeNotFound, errors.CodeNotFound, "no backup of %s", "db_1")

	_, _ = fmt.Println(err)
	_, _ = fmt.Println(errors.TypeOf(err), errors.CodeOf(err))
	_, _ = fmt.Println(errors.IsNotFound(err), errors.Is(err, fs.ErrNotExist))
	// Output:
	// no backup of db_1: file does not exist
	// not-found not_found
	// true true
}

func ExampleNew() {
	err := errors.New(errors.TypeLicenseUnavailable, CodeLicenseExpired, "license expired on 2026-10-01")

	_, _ = fmt.Println(err)
	_, _ = fmt.Println(errors.Asc(err, CodeLicenseExpired))
	// Output:
	// license expired on 2026-10-01
	// true
}

func ExampleNew_details() {
	err := errors.New(errors.TypeInvalidInput, errors.CodeInvalidInput, "invalid filter").
		WithURL("https://signoz.io/docs/filters").
		WithAdditional("operator ~ is unsupported").
		WithSuggestiveAdditional("key servce is unknown", "did you mean: service").
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

func ExampleWrap() {
	cause := errors.New(errors.TypeTooManyRequests, errors.CodeTooManyRequests, "rate limited").WithRetryAfter(2 * time.Second)

	err := errors.Wrap(cause, errors.TypeUnavailable, errors.CodeUnavailable, "cannot query logs")

	_, _ = fmt.Println(err)
	_, _ = fmt.Println(err.Message())
	_, _ = fmt.Println(errors.TypeOf(err), err.RetryAfter())
	// Output:
	// cannot query logs: rate limited
	// cannot query logs
	// unavailable 2s
}

// An untyped error becomes TypeInternal, and its text stays the same.
func ExampleWithAdditionalf() {
	err := errors.WithAdditionalf(fs.ErrPermission, "cannot delete user %s", "u_1")

	_, _ = fmt.Println(err)
	_, _ = fmt.Println(errors.TypeOf(err))
	_, _ = fmt.Println(err.Additionals())
	// Output:
	// permission denied
	// internal
	// [{cannot delete user u_1 []}]
}

func ExampleTypeOf() {
	typed := errors.Join(context.Canceled, errors.New(errors.TypeConflict, errors.CodeConflict, "topic exists"))
	untyped := context.DeadlineExceeded

	_, _ = fmt.Println(errors.TypeOf(typed))
	_, _ = fmt.Println(errors.TypeOf(untyped))
	// Output:
	// conflict
	// unknown
}

func ExampleMessageOf() {
	err := errors.Wrap(fs.ErrClosed, errors.TypeInternal, errors.CodeInternal, "cannot read user")

	_, _ = fmt.Println(errors.MessageOf(err))
	_, _ = fmt.Println(errors.MessageOf(fs.ErrClosed))
	// Output:
	// cannot read user
	// file already closed
}

func ExampleIsRetryable() {
	_, _ = fmt.Println(errors.IsRetryable(errors.New(errors.TypeUnavailable, errors.CodeUnavailable, "endpoint unreachable")))
	_, _ = fmt.Println(errors.IsRetryable(errors.New(errors.TypeInvalidInput, errors.CodeInvalidInput, "bad id")))
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

	logger.Error("cannot restore", errors.Attr(errors.New(errors.TypeNotFound, errors.CodeNotFound, "no backup of db_1")))
	// Output:
	// level=ERROR msg="cannot restore" exception.type=not-found exception.code=not_found exception.message="no backup of db_1"
}

func ExampleAsJSON() {
	err := errors.Wrap(context.Canceled, errors.TypeCanceled, errors.CodeCanceled, "query canceled").
		WithRetryAfter(time.Second)

	body, marshalErr := json.Marshal(errors.AsJSON(err))
	if marshalErr != nil {
		panic(marshalErr)
	}

	_, _ = fmt.Println(string(body))
	// Output:
	// {"type":"canceled","code":"canceled","message":"query canceled","errors":[],"retry":{"delay":1000000000},"suggestions":[]}
}

func ExampleHTTPStatus() {
	_, _ = fmt.Println(errors.HTTPStatus(errors.New(errors.TypeNotFound, errors.CodeNotFound, "dashboard not found")))
	_, _ = fmt.Println(errors.HTTPStatus(context.DeadlineExceeded))
	// Output:
	// 404
	// 500
}

func ExampleTypeFromHTTPStatus() {
	_, _ = fmt.Println(errors.TypeFromHTTPStatus(http.StatusTooManyRequests))
	_, _ = fmt.Println(errors.TypeFromHTTPStatus(http.StatusBadGateway))
	// Output:
	// too-many-requests
	// internal
}
