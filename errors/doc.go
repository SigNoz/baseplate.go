// Package errors creates and inspects typed errors. It replaces the standard
// library's errors package and [fmt.Errorf] in a codebase: [Is], [As], [AsType]
// and [Join] call the standard library, so callers import only this package.
//
// Every error has a [Type], which says what the caller can do about it, a
// [Code], which identifies the failure, and a message. It records the stack
// where it was created.
//
//	if id == "" {
//		return errors.New(errors.TypeInvalidInput, CodeRunIDMissing, "id is required")
//	}
//
//	rows, err := db.QueryContext(ctx, query)
//	if err != nil {
//		return errors.Wrapf(err, errors.TypeInternal, CodeRunsListFailed, "list runs of %s", orgID)
//	}
//
// # Types and codes
//
// Pick the type for what the caller can do: TypeInvalidInput for bad input,
// TypeNotFound for something that doesn't exist, TypeInternal for a dependency
// that failed, and so on. A type carries the HTTP status and retryability
// that [HTTPStatus] and [IsRetryable] report. A package declares types of its
// own with [MustNewType], and the codes of its errors with [MustNewCode]:
//
//	var (
//		TypeLicenseUnavailable = errors.MustNewType("license-unavailable", http.StatusUnavailableForLegalReasons, false)
//		CodeLicenseExpired     = errors.MustNewCode("license_expired")
//	)
//
// Don't declare sentinel errors. An error created at init has a useless stack
// trace. Return a fresh error and branch on its type or code instead, with
// [TypeOf], [CodeOf], [Ast], [Asc], [IsNotFound] and the other Is functions.
//
// # Wrapping
//
// [Wrap] and [Wrapf] add a type, a code and a short context to an error from a
// dependency. The text of the result is "context: cause", and [Is] and [As]
// still reach the cause. Messages say "expected", never "want", and never
// contain a secret such as a token or a password.
//
// Lookups use the first typed error in the chain, so an error wrapped by
// another library or joined with [Join] keeps its type. An error without one
// is untyped: its type is TypeUnknown and its code CodeUnknown.
//
// # Details for API responses
//
// [New], [Newf], [Wrap], [Wrapf] and [WithAdditionalf] return an error with
// methods that add details for an API response. Each returns a copy and leaves
// the original unchanged:
//
//   - WithURL sets a documentation URL.
//   - WithAdditional adds a detail, such as one per invalid field, with
//     suggestions of its own.
//   - WithSuggestions sets suggestions for the error as a whole.
//   - WithRetryAfter sets how long the client should wait before retrying.
//   - WithStacktrace replaces the recorded stack, such as with the stack of a
//     recovered panic.
//
// [AsException] reads them back for an API response, and the error's methods
// Type, Code, Message, URL, Suggestions, RetryAfter and Stacktrace read the
// rest. Wrapping keeps the cause's additionals, suggestions and retry delay.
//
// # Logging
//
// [Attr] adds an error to a [log/slog] record as exception.type,
// exception.code, exception.message and exception.stacktrace:
//
//	logger.ErrorContext(ctx, "cannot send notification", errors.Attr(err))
//
// # HTTP
//
// [AsException] converts an error to the body of an API response,
// [HTTPStatus] picks its status code, and [RetryAfterOf] gives the Retry-After
// delay. [TypeFromHTTPStatus] goes the other way, for errors read from
// another service's response.
//
// # Linting
//
// To keep the standard library's errors package and [fmt.Errorf] out of a
// codebase, add this to .golangci.yml:
//
//	linters:
//	  enable:
//	    - depguard
//	    - forbidigo
//	  settings:
//	    depguard:
//	      rules:
//	        noerrors:
//	          files:
//	            - $all
//	          deny:
//	            - pkg: errors
//	              desc: Use github.com/SigNoz/baseplate.go/errors instead.
//	    forbidigo:
//	      forbid:
//	        - pattern: fmt.Errorf
//	          msg: Use github.com/SigNoz/baseplate.go/errors instead.
package errors
