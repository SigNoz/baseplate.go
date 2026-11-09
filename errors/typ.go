package errors

import (
	"strings"
)

// The types of an error, one per thing a caller can do about it. [HTTPStatus]
// maps each to a status code.
var (
	// TypeUnknown is what [TypeOf] returns for an untyped error. Don't create
	// errors with it.
	TypeUnknown            = typ{"unknown"}
	TypeInvalidInput       = typ{"invalid-input"}
	TypeUnauthenticated    = typ{"unauthenticated"}
	TypeForbidden          = typ{"forbidden"}
	TypeNotFound           = typ{"not-found"}
	TypeMethodNotAllowed   = typ{"method-not-allowed"}
	TypeConflict           = typ{"conflict"}
	TypeTooManyRequests    = typ{"too-many-requests"}
	TypeLicenseUnavailable = typ{"license-unavailable"}
	TypeCanceled           = typ{"canceled"}
	TypeTimeout            = typ{"timeout"}
	TypeUnsupported        = typ{"unsupported"}
	TypeUnavailable        = typ{"unavailable"}
	TypeInternal           = typ{"internal"}
	// TypeFatal is an unrecoverable failure, such as a recovered panic.
	TypeFatal = typ{"fatal"}
)

var (
	typs = []typ{
		TypeUnknown,
		TypeInvalidInput,
		TypeUnauthenticated,
		TypeForbidden,
		TypeNotFound,
		TypeMethodNotAllowed,
		TypeConflict,
		TypeTooManyRequests,
		TypeLicenseUnavailable,
		TypeCanceled,
		TypeTimeout,
		TypeUnsupported,
		TypeUnavailable,
		TypeInternal,
		TypeFatal,
	}
)

type typ struct{ s string }

func newTyp(s string) (typ, error) {
	names := make([]string, len(typs))
	for i, t := range typs {
		if t.s == s {
			return t, nil
		}

		names[i] = t.s
	}

	return typ{}, Newf(TypeInvalidInput, CodeInvalidInput, "type %q is invalid, expected one of %s", s, strings.Join(names, ", "))
}

// String returns the type's name in kebab-case, such as "not-found".
func (t typ) String() string {
	return t.s
}

// MarshalText returns the type's name, as String does.
func (t typ) MarshalText() ([]byte, error) {
	return []byte(t.s), nil
}

// UnmarshalText sets the type from its name, such as "not-found". It returns a
// TypeInvalidInput error for a name that isn't one of the declared types.
func (t *typ) UnmarshalText(text []byte) error {
	parsed, err := newTyp(string(text))
	if err != nil {
		return err
	}

	*t = parsed

	return nil
}

// A retry can't fix these without a change to the request, the credentials or
// the system.
func (t typ) retryable() bool {
	switch t {
	case TypeInvalidInput, TypeUnauthenticated, TypeForbidden, TypeMethodNotAllowed, TypeConflict, TypeLicenseUnavailable, TypeCanceled, TypeUnsupported:
		return false
	}

	return true
}
