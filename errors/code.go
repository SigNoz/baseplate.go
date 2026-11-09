package errors

import (
	"regexp"
)

// The generic codes, one for each type. Use them when a caller has nothing
// finer to tell apart.
var (
	// CodeUnknown is what [CodeOf] returns for an untyped error.
	CodeUnknown            = Code{"unknown"}
	CodeInvalidInput       = Code{"invalid_input"}
	CodeUnauthenticated    = Code{"unauthenticated"}
	CodeForbidden          = Code{"forbidden"}
	CodeNotFound           = Code{"not_found"}
	CodeMethodNotAllowed   = Code{"method_not_allowed"}
	CodeConflict           = Code{"conflict"}
	CodeTooManyRequests    = Code{"too_many_requests"}
	CodeLicenseUnavailable = Code{"license_unavailable"}
	CodeCanceled           = Code{"canceled"}
	CodeTimeout            = Code{"timeout"}
	CodeUnsupported        = Code{"unsupported"}
	CodeUnavailable        = Code{"unavailable"}
	CodeInternal           = Code{"internal"}
	CodeFatal              = Code{"fatal"}
)

var (
	codeRegexp = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)
)

// Code identifies one failure for machines, more finely than its type, such as
// "license_unavailable" or "SecretNotFound". Build one with [NewCode] or
// [MustNewCode], and compare codes with ==.
type Code struct{ s string }

// NewCode returns the code s. It accepts a letter followed by letters, digits
// and underscores, so both snake_case and CamelCase work, and returns a
// TypeInvalidInput error for anything else.
func NewCode(s string) (Code, error) {
	if !codeRegexp.MatchString(s) {
		return Code{}, Newf(TypeInvalidInput, CodeInvalidInput, "code %q is invalid, expected a letter followed by letters, digits or underscores", s)
	}

	return Code{s: s}, nil
}

// MustNewCode is [NewCode] that panics on an invalid code, for declaring codes
// in package-level vars.
func MustNewCode(s string) Code {
	code, err := NewCode(s)
	if err != nil {
		panic(err)
	}

	return code
}

// String returns the code as it was declared.
func (c Code) String() string {
	return c.s
}

// MarshalText returns the code, as String does.
func (c Code) MarshalText() ([]byte, error) {
	return []byte(c.s), nil
}

// UnmarshalText sets the code from text, which must be valid for [NewCode].
func (c *Code) UnmarshalText(text []byte) error {
	parsed, err := NewCode(string(text))
	if err != nil {
		return err
	}

	*c = parsed

	return nil
}
