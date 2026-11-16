package expr

import (
	"strings"

	"github.com/expr-lang/expr/file"
	"github.com/expr-lang/expr/parser/lexer"

	"github.com/SigNoz/baseplate.go/errors"
)

// aliases maps the language's spellings that expr-lang/expr doesn't have to
// the expr-lang ones.
var aliases = map[string]string{
	"AND":      "and",
	"OR":       "or",
	"NOT":      "not",
	"IN":       "in",
	"CONTAINS": "contains",
	"REGEXP":   "matches",
	"regexp":   "matches",
}

// source is an expression as the user wrote it and as expr-lang/expr reads
// it.
type source struct {
	text      string
	exprlang  string
	respelled []respelledToken // In source order.
}

// respelledToken is an alias in the source and the expr-lang spelling it
// became, such as AND as and or = as ==. Offsets are in code points.
type respelledToken struct {
	from         int // From and to are the token's offsets in the source.
	to           int
	exprlangFrom int // ExprlangFrom and exprlangTo are its offsets in the expr-lang text.
	exprlangTo   int
	text         string // Text and exprlangText are its spellings in the source and expr-lang.
	exprlangText string
}

// newSource respells the language's aliases on expr-lang's own tokens, so
// quoted text is never touched. An identifier after "." or "?." is a member
// name and stays as written, and so does the "=" of "let name =".
func newSource(text string) (*source, error) {
	s := &source{text: text}

	tokens, err := lexer.Lex(file.NewSource(text))
	if err != nil {
		return nil, s.positionError(err, CodeSyntaxInvalid)
	}

	runes := []rune(text)
	exprlang := make([]rune, 0, len(runes))

	last, shift := 0, 0
	for i, token := range tokens {
		spelling := exprlangSpelling(tokens, i)
		if spelling == "" {
			continue
		}

		exprlang = append(exprlang, runes[last:token.From]...)
		exprlang = append(exprlang, []rune(spelling)...)

		exprlangFrom := token.From + shift
		shift += len([]rune(spelling)) - (token.To - token.From)
		s.respelled = append(s.respelled, respelledToken{
			from:         token.From,
			to:           token.To,
			exprlangFrom: exprlangFrom,
			exprlangTo:   token.To + shift,
			text:         token.Value,
			exprlangText: spelling,
		})
		last = token.To
	}

	exprlang = append(exprlang, runes[last:]...)
	s.exprlang = string(exprlang)

	return s, nil
}

// offset maps an offset in the expr-lang text to the source. An offset
// inside a respelled token maps to the same character of the source token,
// or to its last one when the source token is shorter; an end offset maps
// at most to the token's end.
func (s *source) offset(at int, end bool) int {
	shift := 0
	for _, r := range s.respelled {
		switch {
		case at <= r.exprlangFrom:
			return at + shift
		case at < r.exprlangTo:
			last := r.to - r.from - 1
			if end {
				last++
			}

			return r.from + min(at-r.exprlangFrom, last)
		}

		shift = r.to - r.exprlangTo
	}

	return at + shift
}

// positionError turns an error from expr-lang, whose positions are on the
// expr-lang text, into an error on the source with code.
func (s *source) positionError(err error, code errors.Code) error {
	exprlangErr, ok := errors.AsType[*file.Error](err)
	if !ok {
		return errors.New(errors.TypeInvalidInput, code, err.Error())
	}

	from, to := s.offset(exprlangErr.From, false), s.offset(exprlangErr.To, true)

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

	return errors.Wrap(position, errors.TypeInvalidInput, code, s.respell(exprlangErr.Message, exprlangErr.From, exprlangErr.To))
}

// respell puts back the source spelling of a respelled token that an error
// at from-to on the expr-lang text names, as in "invalid operation: =" for
// "==".
func (s *source) respell(message string, from, to int) string {
	to = max(to, from+1)

	for _, r := range s.respelled {
		if r.exprlangFrom >= to || from >= r.exprlangTo {
			continue
		}

		message = strings.Replace(message, `("`+r.exprlangText+`")`, `("`+r.text+`")`, 1)
		message = strings.Replace(message, "operation: "+r.exprlangText+" (", "operation: "+r.text+" (", 1)
	}

	return message
}

// exprlangSpelling returns the expr-lang spelling of tokens[i], or "" when
// it stays as written.
func exprlangSpelling(tokens []lexer.Token, i int) string {
	token := tokens[i]

	switch {
	case token.Kind == lexer.Identifier:
		if i > 0 && tokens[i-1].Kind == lexer.Operator && (tokens[i-1].Value == "." || tokens[i-1].Value == "?.") {
			return ""
		}

		return aliases[token.Value]
	case token.Kind == lexer.Operator && token.Value == "=":
		if i > 1 && tokens[i-1].Kind == lexer.Identifier && tokens[i-2].Kind == lexer.Operator && tokens[i-2].Value == "let" {
			return ""
		}

		return "=="
	}

	return ""
}
