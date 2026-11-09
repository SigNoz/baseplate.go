package errors

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

type typ struct{ s string }

// String returns the type's name in kebab-case, such as "not-found".
func (t typ) String() string {
	return t.s
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
