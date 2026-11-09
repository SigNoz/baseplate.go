package errors

import (
	"regexp"
)

var (
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

// Code identifies one failure for machines, more finely than its type.
type Code struct{ s string }

// NewCode accepts snake_case and CamelCase, so each repo keeps its own convention.
func NewCode(s string) (Code, error) {
	if !codeRegexp.MatchString(s) {
		return Code{}, Newf(TypeInvalidInput, CodeInvalidInput, "code %q is invalid, expected a letter followed by letters, digits or underscores", s)
	}

	return Code{s: s}, nil
}

// MustNewCode panics on an invalid code, so it belongs in package-level vars.
func MustNewCode(s string) Code {
	code, err := NewCode(s)
	if err != nil {
		panic(err)
	}

	return code
}

func (c Code) String() string {
	return c.s
}
