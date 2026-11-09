package errors

var (
	// TypeUnknown is the type of an untyped error.
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

func (t typ) String() string {
	return t.s
}

// A retry can't fix these without a change to the request, the credentials or the system.
func (t typ) retryable() bool {
	switch t {
	case TypeInvalidInput, TypeUnauthenticated, TypeForbidden, TypeMethodNotAllowed, TypeConflict, TypeLicenseUnavailable, TypeCanceled, TypeUnsupported:
		return false
	}

	return true
}
