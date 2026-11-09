package errors

import (
	"bytes"
	"context"
	"io/fs"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type tWrapper struct{ cause error }

func (w *tWrapper) Error() string {
	return "wrapped by a library: " + w.cause.Error()
}

func (w *tWrapper) Unwrap() error {
	return w.cause
}

func TestError(t *testing.T) {
	testCases := []struct {
		name            string
		err             *base
		expectedError   string
		expectedMessage string
		expectedType    typ
		expectedCode    Code
	}{
		{
			name:            "New",
			err:             New(TypeNotFound, CodeNotFound, "run not found"),
			expectedError:   "run not found",
			expectedMessage: "run not found",
			expectedType:    TypeNotFound,
			expectedCode:    CodeNotFound,
		},
		{
			name:            "Newf",
			err:             Newf(TypeConflict, MustNewCode("session_busy"), "session %q is busy", "sess_1"),
			expectedError:   `session "sess_1" is busy`,
			expectedMessage: `session "sess_1" is busy`,
			expectedType:    TypeConflict,
			expectedCode:    MustNewCode("session_busy"),
		},
		{
			name:            "Wrap_AppendsCause",
			err:             Wrap(New(TypeInternal, CodeInternal, "connection reset"), TypeUnavailable, CodeUnavailable, "cannot store event"),
			expectedError:   "cannot store event: connection reset",
			expectedMessage: "cannot store event",
			expectedType:    TypeUnavailable,
			expectedCode:    CodeUnavailable,
		},
		{
			name:            "Wrapf_AppendsCause",
			err:             Wrapf(fs.ErrNotExist, TypeInvalidInput, MustNewCode("SpecInvalid"), "spec %q", "spec.auth"),
			expectedError:   `spec "spec.auth": file does not exist`,
			expectedMessage: `spec "spec.auth"`,
			expectedType:    TypeInvalidInput,
			expectedCode:    MustNewCode("SpecInvalid"),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expectedError, testCase.err.Error())
			assert.Equal(t, testCase.expectedMessage, testCase.err.Message())
			assert.Equal(t, testCase.expectedType, testCase.err.Type())
			assert.Equal(t, testCase.expectedCode, testCase.err.Code())
		})
	}
}

func TestTypeOfCodeOfMessageOf(t *testing.T) {
	testCases := []struct {
		name            string
		err             error
		expectedType    typ
		expectedCode    Code
		expectedMessage string
	}{
		{
			name:            "Typed",
			err:             New(TypeForbidden, CodeForbidden, "push to main"),
			expectedType:    TypeForbidden,
			expectedCode:    CodeForbidden,
			expectedMessage: "push to main",
		},
		{
			name:            "Wrapped_OutermostWins",
			err:             Wrap(New(TypeNotFound, CodeNotFound, "no such branch"), TypeInternal, CodeInternal, "cannot clone"),
			expectedType:    TypeInternal,
			expectedCode:    CodeInternal,
			expectedMessage: "cannot clone",
		},
		{
			name:            "ForeignWrapper_BaseInside",
			err:             &tWrapper{cause: New(TypeTimeout, CodeTimeout, "query took too long")},
			expectedType:    TypeTimeout,
			expectedCode:    CodeTimeout,
			expectedMessage: "query took too long",
		},
		{
			name:            "Joined_FirstBase",
			err:             Join(context.Canceled, New(TypeTooManyRequests, CodeTooManyRequests, "rate limited")),
			expectedType:    TypeTooManyRequests,
			expectedCode:    CodeTooManyRequests,
			expectedMessage: "rate limited",
		},
		{
			name:            "Foreign_Unknown",
			err:             context.DeadlineExceeded,
			expectedType:    TypeUnknown,
			expectedCode:    CodeUnknown,
			expectedMessage: "context deadline exceeded",
		},
		{
			name:            "Nil_Unknown",
			err:             nil,
			expectedType:    TypeUnknown,
			expectedCode:    CodeUnknown,
			expectedMessage: "",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expectedType, TypeOf(testCase.err))
			assert.Equal(t, testCase.expectedCode, CodeOf(testCase.err))
			assert.Equal(t, testCase.expectedMessage, MessageOf(testCase.err))
			assert.True(t, Ast(testCase.err, testCase.expectedType))
			assert.True(t, Asc(testCase.err, testCase.expectedCode))
		})
	}
}

func TestPredicates(t *testing.T) {
	testCases := []struct {
		name      string
		predicate func(error) bool
		match     typ
		other     typ
	}{
		{name: "IsNotFound", predicate: IsNotFound, match: TypeNotFound, other: TypeInternal},
		{name: "IsConflict", predicate: IsConflict, match: TypeConflict, other: TypeInvalidInput},
		{name: "IsUnauthenticated", predicate: IsUnauthenticated, match: TypeUnauthenticated, other: TypeForbidden},
		{name: "IsForbidden", predicate: IsForbidden, match: TypeForbidden, other: TypeUnauthenticated},
		{name: "IsUnavailable", predicate: IsUnavailable, match: TypeUnavailable, other: TypeTimeout},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.True(t, testCase.predicate(New(testCase.match, CodeUnknown, "matching")))
			assert.False(t, testCase.predicate(New(testCase.other, CodeUnknown, "other")))
			assert.False(t, testCase.predicate(nil))
		})
	}
}

func TestIsRetryable(t *testing.T) {
	testCases := []struct {
		name              string
		err               error
		expectedRetryable bool
	}{
		{name: "InvalidInput_No", err: New(TypeInvalidInput, CodeInvalidInput, "bad id"), expectedRetryable: false},
		{name: "Unauthenticated_No", err: New(TypeUnauthenticated, CodeUnauthenticated, "no token"), expectedRetryable: false},
		{name: "Forbidden_No", err: New(TypeForbidden, CodeForbidden, "read-only key"), expectedRetryable: false},
		{name: "MethodNotAllowed_No", err: New(TypeMethodNotAllowed, CodeMethodNotAllowed, "PATCH on /runs"), expectedRetryable: false},
		{name: "Conflict_No", err: New(TypeConflict, CodeConflict, "topic exists"), expectedRetryable: false},
		{name: "LicenseUnavailable_No", err: New(TypeLicenseUnavailable, CodeLicenseUnavailable, "no license"), expectedRetryable: false},
		{name: "Canceled_No", err: New(TypeCanceled, CodeCanceled, "client went away"), expectedRetryable: false},
		{name: "Unsupported_No", err: New(TypeUnsupported, CodeUnsupported, "no sqlite"), expectedRetryable: false},
		{name: "NotFound_Yes", err: New(TypeNotFound, CodeNotFound, "secret not created yet"), expectedRetryable: true},
		{name: "TooManyRequests_Yes", err: New(TypeTooManyRequests, CodeTooManyRequests, "slow down"), expectedRetryable: true},
		{name: "Timeout_Yes", err: New(TypeTimeout, CodeTimeout, "slow query"), expectedRetryable: true},
		{name: "Unavailable_Yes", err: New(TypeUnavailable, CodeUnavailable, "endpoint unreachable"), expectedRetryable: true},
		{name: "Internal_Yes", err: New(TypeInternal, CodeInternal, "disk full"), expectedRetryable: true},
		{name: "Fatal_Yes", err: New(TypeFatal, CodeFatal, "panic"), expectedRetryable: true},
		{name: "Foreign_Yes", err: context.DeadlineExceeded, expectedRetryable: true},
		{name: "Nil_No", err: nil, expectedRetryable: false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expectedRetryable, IsRetryable(testCase.err))
		})
	}
}

func TestWrap_KeepsDetailsButNotURL(t *testing.T) {
	cause := New(TypeTooManyRequests, CodeTooManyRequests, "rate limited").
		WithURL("https://signoz.io/docs/limits").
		WithSuggestiveAdditional("key servce", "did you mean: service").
		WithSuggestions("retry later").
		WithRetryAfter(time.Second)

	testCases := []struct {
		name string
		err  *base
	}{
		{name: "Wrap", err: Wrap(cause, TypeInternal, CodeInternal, "cannot query")},
		{name: "Wrapf", err: Wrapf(cause, TypeInternal, CodeInternal, "cannot query %s", "logs")},
		{name: "Wrap_ThroughForeignWrapper", err: Wrap(&tWrapper{cause: cause}, TypeInternal, CodeInternal, "cannot query")},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, cause.Additionals(), testCase.err.Additionals())
			assert.Equal(t, []string{"retry later"}, testCase.err.Suggestions())
			assert.Equal(t, time.Second, testCase.err.RetryAfter())
			assert.Equal(t, time.Second, RetryAfterOf(testCase.err))
			assert.Empty(t, testCase.err.URL())
		})
	}
}

func TestWithAdditionalf(t *testing.T) {
	testCases := []struct {
		name                string
		err                 error
		expectedError       string
		expectedMessage     string
		expectedType        typ
		expectedCode        Code
		expectedAdditionals []Additional
	}{
		{
			name:                "Base_AddsDetail",
			err:                 New(TypeForbidden, MustNewCode("root_user"), "user is root").WithAdditional("a root user can't be changed"),
			expectedError:       "user is root",
			expectedMessage:     "user is root",
			expectedType:        TypeForbidden,
			expectedCode:        MustNewCode("root_user"),
			expectedAdditionals: []Additional{{Message: "a root user can't be changed"}, {Message: "cannot delete user u_1"}},
		},
		{
			name:                "WrappedBase_MessageWithoutCause",
			err:                 Wrap(fs.ErrClosed, TypeInternal, MustNewCode("store_closed"), "cannot read user"),
			expectedError:       "cannot read user: file already closed",
			expectedMessage:     "cannot read user",
			expectedType:        TypeInternal,
			expectedCode:        MustNewCode("store_closed"),
			expectedAdditionals: []Additional{{Message: "cannot delete user u_1"}},
		},
		{
			name:                "Foreign_Internal",
			err:                 fs.ErrPermission,
			expectedError:       "permission denied",
			expectedMessage:     "permission denied",
			expectedType:        TypeInternal,
			expectedCode:        CodeInternal,
			expectedAdditionals: []Additional{{Message: "cannot delete user u_1"}},
		},
		{
			name:                "ForeignWrapper_InnerTypeMessageAndDetails",
			err:                 &tWrapper{cause: New(TypeNotFound, CodeNotFound, "user not found").WithAdditional("checked org o_1")},
			expectedError:       "wrapped by a library: user not found",
			expectedMessage:     "user not found",
			expectedType:        TypeNotFound,
			expectedCode:        CodeNotFound,
			expectedAdditionals: []Additional{{Message: "checked org o_1"}, {Message: "cannot delete user u_1"}},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := WithAdditionalf(testCase.err, "cannot delete user %s", "u_1")

			assert.Equal(t, testCase.expectedError, err.Error())
			assert.Equal(t, testCase.expectedMessage, err.Message())
			assert.Equal(t, testCase.expectedType, err.Type())
			assert.Equal(t, testCase.expectedCode, err.Code())
			assert.Equal(t, testCase.expectedAdditionals, err.Additionals())
			assert.Equal(t, testCase.err, err.Unwrap())
		})
	}
}

func TestWithAdditionalf_KeepsURL(t *testing.T) {
	testCases := []struct {
		name string
		err  error
	}{
		{name: "Typed", err: New(TypeForbidden, CodeForbidden, "user is root").WithURL("https://signoz.io/docs/users")},
		{name: "ForeignWrapper", err: &tWrapper{cause: New(TypeForbidden, CodeForbidden, "user is root").WithURL("https://signoz.io/docs/users")}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := WithAdditionalf(testCase.err, "cannot delete user %s", "u_3")

			assert.Equal(t, "https://signoz.io/docs/users", err.URL())
			assert.Equal(t, "https://signoz.io/docs/users", AsJSON(err).URL)
		})
	}
}

func TestWithAdditionalf_Base_KeepsOriginalStacktrace(t *testing.T) {
	original := New(TypeForbidden, MustNewCode("root_user"), "user is root")

	err := WithAdditionalf(&tWrapper{cause: original}, "cannot delete user %s", "u_2")

	assert.Equal(t, original.Stacktrace(), err.Stacktrace())
}

func TestStacktrace_StartsAtCaller(t *testing.T) {
	testCases := []struct {
		name string
		err  *base
	}{
		{name: "New", err: New(TypeInternal, CodeInternal, "sandbox lost")},
		{name: "Newf", err: Newf(TypeInternal, CodeInternal, "sandbox %s lost", "sb_1")},
		{name: "Wrap", err: Wrap(context.Canceled, TypeCanceled, CodeCanceled, "run stopped")},
		{name: "Wrapf", err: Wrapf(context.Canceled, TypeCanceled, CodeCanceled, "run %s stopped", "run_1")},
		{name: "WithAdditionalf_Foreign", err: WithAdditionalf(context.Canceled, "run %s stopped", "run_1")},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.True(t, strings.HasPrefix(testCase.err.Stacktrace(), "github.com/SigNoz/baseplate.go/errors.TestStacktrace_StartsAtCaller\n"), testCase.err.Stacktrace())
		})
	}
}

func TestAttr(t *testing.T) {
	testCases := []struct {
		name             string
		err              error
		expectedContains []string
		expectedMissing  []string
	}{
		{
			name: "Base_TypeCodeMessageStacktrace",
			err:  Wrap(fs.ErrNotExist, TypeNotFound, MustNewCode("backup_missing"), "no backup of db_1"),
			expectedContains: []string{
				"exception.type=not-found",
				"exception.code=backup_missing",
				`exception.message="no backup of db_1: file does not exist"`,
				`exception.stacktrace="github.com/SigNoz/baseplate.go/errors.TestAttr`,
			},
		},
		{
			name:             "Foreign_GoTypeAndMessage",
			err:              context.DeadlineExceeded,
			expectedContains: []string{"exception.type=context.deadlineExceededError", `exception.message="context deadline exceeded"`},
			expectedMissing:  []string{"exception.code", "exception.stacktrace"},
		},
		{
			name:            "Nil_Dropped",
			err:             nil,
			expectedMissing: []string{"exception"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var buf bytes.Buffer

			slog.New(slog.NewTextHandler(&buf, nil)).Error("cannot restore", Attr(testCase.err))

			for _, expected := range testCase.expectedContains {
				assert.Contains(t, buf.String(), expected)
			}

			for _, missing := range testCase.expectedMissing {
				assert.NotContains(t, buf.String(), missing)
			}
		})
	}
}
