package errors

import (
	"regexp"
)

var (
	// CodeUnknown is what [CodeOf] returns for an untyped error, and the code
	// [WithAdditionalf] gives one. Declare a package's own codes for the errors
	// it creates.
	CodeUnknown = Code{"unknown"}
)

var (
	codeRegexp      = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)
	codeInvalidCode = Code{"invalid_code"}
)

// Code identifies one failure for machines, more finely than its type, such as
// "license_expired" or "SecretNotFound". Each package declares its own codes
// with [NewCode] or [MustNewCode]. Compare codes with ==.
type Code struct{ s string }

// NewCode returns the code s. It accepts a letter followed by letters, digits
// and underscores, so both snake_case and CamelCase work, and returns a
// TypeInvalidInput error for anything else.
func NewCode(s string) (Code, error) {
	if !codeRegexp.MatchString(s) {
		return Code{}, Newf(TypeInvalidInput, codeInvalidCode, "code %q is invalid, expected a letter followed by letters, digits or underscores", s)
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
