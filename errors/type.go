package errors

import (
	"net/http"
	"regexp"
)

// The base types, one per thing a caller can do about an error. A package
// declares more with [NewType] or [MustNewType].
var (
	// TypeUnknown is what [TypeOf] returns for an untyped error. Don't create
	// errors with it.
	TypeUnknown          = Type{name: "unknown", status: http.StatusInternalServerError, retryable: true}
	TypeInvalidInput     = Type{name: "invalid-input", status: http.StatusBadRequest, retryable: false}
	TypeUnauthenticated  = Type{name: "unauthenticated", status: http.StatusUnauthorized, retryable: false}
	TypeForbidden        = Type{name: "forbidden", status: http.StatusForbidden, retryable: false}
	TypeNotFound         = Type{name: "not-found", status: http.StatusNotFound, retryable: true}
	TypeMethodNotAllowed = Type{name: "method-not-allowed", status: http.StatusMethodNotAllowed, retryable: false}
	TypeConflict         = Type{name: "conflict", status: http.StatusConflict, retryable: false}
	TypeTooManyRequests  = Type{name: "too-many-requests", status: http.StatusTooManyRequests, retryable: true}
	// TypeCanceled maps to nginx's non-standard 499 Client Closed Request.
	TypeCanceled    = Type{name: "canceled", status: statusClientClosedRequest, retryable: false}
	TypeTimeout     = Type{name: "timeout", status: http.StatusGatewayTimeout, retryable: true}
	TypeUnsupported = Type{name: "unsupported", status: http.StatusNotImplemented, retryable: false}
	TypeUnavailable = Type{name: "unavailable", status: http.StatusServiceUnavailable, retryable: true}
	TypeInternal    = Type{name: "internal", status: http.StatusInternalServerError, retryable: true}
	// TypeFatal is an unrecoverable failure, such as a recovered panic.
	TypeFatal = Type{name: "fatal", status: http.StatusInternalServerError, retryable: true}
)

var (
	typeNameRegexp  = regexp.MustCompile(`^[a-z]+(-[a-z]+)*$`)
	codeInvalidType = Code{"invalid_type"}
)

// Type says what a caller can do about an error. It carries the HTTP status
// that [HTTPStatus] returns and whether [IsRetryable] reports that a retry
// may succeed. Compare types with ==. A type marshals to its name but can't be
// decoded from it, because the name doesn't carry the status or retryability.
type Type struct {
	name      string
	status    int
	retryable bool
}

// NewType returns a type for a package's own errors, such as
// "license-unavailable". The name must be kebab-case, and status an HTTP
// error status from 400 to 599. Retryable says whether retrying the operation
// that failed may succeed without a change to the request, credentials or
// system. It returns a TypeInvalidInput error for an invalid name or status.
func NewType(name string, status int, retryable bool) (Type, error) {
	if !typeNameRegexp.MatchString(name) {
		return Type{}, Newf(TypeInvalidInput, codeInvalidType, "type %q is invalid, expected kebab-case such as not-found", name)
	}

	if status < http.StatusBadRequest || status > 599 {
		return Type{}, Newf(TypeInvalidInput, codeInvalidType, "status %d of type %q is invalid, expected 400 to 599", status, name)
	}

	return Type{name: name, status: status, retryable: retryable}, nil
}

// MustNewType is [NewType] that panics on an invalid type, for declaring
// types in package-level vars.
func MustNewType(name string, status int, retryable bool) Type {
	t, err := NewType(name, status, retryable)
	if err != nil {
		panic(err)
	}

	return t
}

// String returns the type's name, such as "not-found".
func (t Type) String() string {
	return t.name
}

// MarshalText returns the type's name, as String does.
func (t Type) MarshalText() ([]byte, error) {
	return []byte(t.name), nil
}
