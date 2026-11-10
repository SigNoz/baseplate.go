package expr

import (
	"github.com/expr-lang/expr/file"
	"github.com/expr-lang/expr/parser/lexer"
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
	text     string
	upstream string
	rewrites []rewrite // In source order.
}

// rewrite is one token the shim respelled. Offsets are in code points.
type rewrite struct {
	from   int // From and to are offsets in the source.
	to     int
	upFrom int // UpFrom and upTo are offsets in the upstream text.
	upTo   int
	text   string // Text and up are the token's spellings in the source and upstream.
	up     string
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
		up := respelling(tokens, i)
		if up == "" {
			continue
		}

		upstream = append(upstream, runes[last:token.From]...)
		upstream = append(upstream, []rune(up)...)

		upFrom := token.From + shift
		shift += len([]rune(up)) - (token.To - token.From)
		s.rewrites = append(s.rewrites, rewrite{
			from:   token.From,
			to:     token.To,
			upFrom: upFrom,
			upTo:   token.To + shift,
			text:   token.Value,
			up:     up,
		})
		last = token.To
	}

	upstream = append(upstream, runes[last:]...)
	s.upstream = string(upstream)

	return s, nil
}

// respelling returns the upstream spelling of tokens[i], or "" when it stays
// as written.
func respelling(tokens []lexer.Token, i int) string {
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

// offset maps an offset in the upstream text to the source. An offset inside
// a rewritten token maps to the same character of the source token, or to
// its last one when the source token is shorter; an end offset maps at most
// to the token's end.
func (s *source) offset(up int, end bool) int {
	shift := 0
	for _, r := range s.rewrites {
		switch {
		case up <= r.upFrom:
			return up + shift
		case up < r.upTo:
			last := r.to - r.from - 1
			if end {
				last++
			}

			return r.from + min(up-r.upFrom, last)
		}

		shift = r.to - r.upTo
	}

	return up + shift
}
