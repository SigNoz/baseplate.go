package expr

import (
	"fmt"
	"strings"

	"github.com/expr-lang/expr/file"

	"github.com/SigNoz/baseplate.go/errors"
)

// The codes of the errors [Compile], [CompileValue], [Program.Bool] and
// [Program.Run] return.
var (
	CodeSyntaxInvalid   = errors.MustNewCode("syntax_invalid")
	CodeTypeInvalid     = errors.MustNewCode("type_invalid")
	CodeResultNotBool   = errors.MustNewCode("result_not_bool")
	CodeNodesExceeded   = errors.MustNewCode("nodes_exceeded")
	CodeRunFailed       = errors.MustNewCode("run_failed")
	CodeNestingTooDeep  = errors.MustNewCode("nesting_too_deep")
	CodePatternInvalid  = errors.MustNewCode("pattern_invalid")
	CodeMemoryExceeded  = errors.MustNewCode("memory_exceeded")
	CodeVarsInvalid     = errors.MustNewCode("vars_invalid")
	CodeNameUnknown     = errors.MustNewCode("name_unknown")
	CodeFunctionUnknown = errors.MustNewCode("function_unknown")
	CodeVarTypeInvalid  = errors.MustNewCode("var_type_invalid")
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

// positionError turns an error from expr, whose positions are on the
// upstream text, into an error on the source with code and suggestions.
func (s *source) positionError(err error, code errors.Code, suggestions ...string) error {
	upstream, ok := errors.AsType[*file.Error](err)
	if !ok {
		return errors.New(errors.TypeInvalidInput, code, err.Error())
	}

	from, to := s.offset(upstream.From, false), s.offset(upstream.To, true)

	bound := (&file.Error{
		From: from,
		To:   to,
	}).Bind(file.NewSource(s.text))

	position := &PositionError{
		From:    from,
		To:      to,
		Line:    bound.Line,
		Column:  bound.Column + 1,
		snippet: bound.Snippet,
	}

	wrapped := errors.Wrap(position, errors.TypeInvalidInput, code, s.respell(upstream.Message, upstream.From, upstream.To))
	if len(suggestions) > 0 {
		return wrapped.WithSuggestions(suggestions...)
	}

	return wrapped
}

// respell puts back the source spelling of a rewritten token that an error
// at from-to on the upstream text names, as in "invalid operation: =" for
// "==".
func (s *source) respell(message string, from, to int) string {
	to = max(to, from+1)

	for _, r := range s.rewrites {
		if r.upFrom >= to || from >= r.upTo {
			continue
		}

		message = strings.Replace(message, `("`+r.up+`")`, `("`+r.text+`")`, 1)
		message = strings.Replace(message, "operation: "+r.up+" (", "operation: "+r.text+" (", 1)
	}

	return message
}
