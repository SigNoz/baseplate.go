package expr

import (
	"strings"

	"github.com/expr-lang/expr/file"
	"github.com/expr-lang/expr/parser/lexer"

	"github.com/SigNoz/baseplate.go/errors"
)

// aliases maps the language's spellings that upstream expr doesn't have to
// the upstream ones.
var aliases = map[string]string{
	"AND":      "and",
	"OR":       "or",
	"NOT":      "not",
	"IN":       "in",
	"CONTAINS": "contains",
	"REGEXP":   "matches",
	"regexp":   "matches",
}

// source is an expression as the user wrote it and as upstream expr reads
// it.
type source struct {
	text      string
	upstream  string
	respelled []respelledToken // In source order.
}

// respelledToken is an alias in the source and the upstream spelling it
// became, such as AND as and or = as ==. Offsets are in code points.
type respelledToken struct {
	from         int // From and to are the token's offsets in the source.
	to           int
	upstreamFrom int // UpstreamFrom and upstreamTo are its offsets in the upstream text.
	upstreamTo   int
	text         string // Text and upstreamText are its spellings in the source and upstream.
	upstreamText string
}

// newSource respells the language's aliases on upstream's own tokens, so
// quoted text is never touched. An identifier after "." or "?." is a member
// name and stays as written, and so does the "=" of "let name =".
func newSource(text string) (*source, error) {
	s := &source{text: text}

	tokens, err := lexer.Lex(file.NewSource(text))
	if err != nil {
		return nil, s.positionError(err, CodeSyntaxInvalid)
	}

	runes := []rune(text)
	upstream := make([]rune, 0, len(runes))

	last, shift := 0, 0
	for i, token := range tokens {
		spelling := upstreamSpelling(tokens, i)
		if spelling == "" {
			continue
		}

		upstream = append(upstream, runes[last:token.From]...)
		upstream = append(upstream, []rune(spelling)...)

		upstreamFrom := token.From + shift
		shift += len([]rune(spelling)) - (token.To - token.From)
		s.respelled = append(s.respelled, respelledToken{
			from:         token.From,
			to:           token.To,
			upstreamFrom: upstreamFrom,
			upstreamTo:   token.To + shift,
			text:         token.Value,
			upstreamText: spelling,
		})
		last = token.To
	}

	upstream = append(upstream, runes[last:]...)
	s.upstream = string(upstream)

	return s, nil
}

// offset maps an offset in the upstream text to the source. An offset inside
// a respelled token maps to the same character of the source token, or to
// its last one when the source token is shorter; an end offset maps at most
// to the token's end.
func (s *source) offset(up int, end bool) int {
	shift := 0
	for _, r := range s.respelled {
		switch {
		case up <= r.upstreamFrom:
			return up + shift
		case up < r.upstreamTo:
			last := r.to - r.from - 1
			if end {
				last++
			}

			return r.from + min(up-r.upstreamFrom, last)
		}

		shift = r.to - r.upstreamTo
	}

	return up + shift
}

// positionError turns an error from expr, whose positions are on the
// upstream text, into an error on the source with code.
func (s *source) positionError(err error, code errors.Code) error {
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

	return errors.Wrap(position, errors.TypeInvalidInput, code, s.respell(upstream.Message, upstream.From, upstream.To))
}

// respell puts back the source spelling of a respelled token that an error
// at from-to on the upstream text names, as in "invalid operation: =" for
// "==".
func (s *source) respell(message string, from, to int) string {
	to = max(to, from+1)

	for _, r := range s.respelled {
		if r.upstreamFrom >= to || from >= r.upstreamTo {
			continue
		}

		message = strings.Replace(message, `("`+r.upstreamText+`")`, `("`+r.text+`")`, 1)
		message = strings.Replace(message, "operation: "+r.upstreamText+" (", "operation: "+r.text+" (", 1)
	}

	return message
}

// upstreamSpelling returns the upstream spelling of tokens[i], or "" when it
// stays as written.
func upstreamSpelling(tokens []lexer.Token, i int) string {
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
