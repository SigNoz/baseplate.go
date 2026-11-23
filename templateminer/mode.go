package templateminer

import (
	"github.com/SigNoz/baseplate.go/errors"
)

// The modes a [Miner] runs in.
var (
	// ModeStream mines lines one at a time with Drain.
	ModeStream = Mode{"stream"}
)

// The codes of the errors for a mode.
var (
	CodeModeInvalid = errors.MustNewCode("mode_invalid") // The mode isn't declared.
)

// Mode is the algorithm a [Miner] uses. Compare modes with ==.
type Mode struct{ s string }

// NewMode returns the mode named s, or a TypeInvalidInput error with
// [CodeModeInvalid] for a name that isn't declared, including "".
func NewMode(s string) (Mode, error) {
	switch s {
	case ModeStream.s:
		return Mode{s}, nil
	}

	return Mode{}, errors.Newf(errors.TypeInvalidInput, CodeModeInvalid, "mode %q is invalid, expected %s", s, ModeStream)
}

// String returns the mode's name, or "" for the zero Mode.
func (mode Mode) String() string {
	return mode.s
}

// MarshalText returns the mode's name.
func (mode Mode) MarshalText() ([]byte, error) {
	return []byte(mode.s), nil
}

// UnmarshalText sets the mode from its name, and returns the error of
// [NewMode] for a name that isn't declared.
func (mode *Mode) UnmarshalText(text []byte) error {
	parsed, err := NewMode(string(text))
	if err != nil {
		return err
	}

	*mode = parsed

	return nil
}
