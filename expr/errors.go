package expr

import (
	"fmt"

	"github.com/SigNoz/baseplate.go/errors"
)

// The codes of the errors [Compile], [CompileValue], [Program.Bool] and
// [Program.Run] return.
var (
	CodeSyntaxInvalid  = errors.MustNewCode("syntax_invalid")
	CodeTypeInvalid    = errors.MustNewCode("type_invalid")
	CodeResultNotBool  = errors.MustNewCode("result_not_bool")
	CodeNodesExceeded  = errors.MustNewCode("nodes_exceeded")
	CodeRunFailed      = errors.MustNewCode("run_failed")
	CodeNestingTooDeep = errors.MustNewCode("nesting_too_deep")
	CodePatternInvalid = errors.MustNewCode("pattern_invalid")
	CodeMemoryExceeded = errors.MustNewCode("memory_exceeded")
	CodeVarsInvalid    = errors.MustNewCode("vars_invalid")
)

// PositionError is where in an expression's source an error is. It is the
// cause of every error with a position that [Compile], [CompileValue],
// [Program.Bool] and [Program.Run] return; the type, code, message and
// suggestions are on the [github.com/SigNoz/baseplate.go/errors] error that
// wraps it. Get it with [github.com/SigNoz/baseplate.go/errors.AsType].
type PositionError struct {
	From    int `json:"from"`   // Offset of the first code point, from 0.
	To      int `json:"to"`     // Offset after the last code point.
	Line    int `json:"line"`   // From 1.
	Column  int `json:"column"` // From 1, in code points.
	snippet string
}

// Error returns the line and column, followed by the source line with a
// caret under the error.
func (e *PositionError) Error() string {
	return fmt.Sprintf("(%d:%d)%s", e.Line, e.Column, e.snippet)
}
