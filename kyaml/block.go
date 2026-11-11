package kyaml

import (
	"bytes"
	"strings"

	kyyaml "sigs.k8s.io/kustomize/kyaml/yaml"

	"github.com/SigNoz/baseplate.go/errors"
)

// EditBlockScalar edits the text held in the literal block scalar at
// outerPath by handing its dedented content to edit and splicing the result
// back into the document, re-indented to the block's indentation. Every byte
// outside the block body is preserved, and inside it every line the callback
// leaves unchanged is restored from its original source bytes, so only what
// edit changes changes. A change to one field of a config embedded in a
// manifest (such as a collector config kept in a ConfigMap) yields a diff of
// just that field. The edit callback usually composes one or more [Set]
// calls on the inner document.
//
// The outerPath argument is a kustomize kyaml lookup path to the block
// scalar; a sequence element is selected with a "[field=value]" segment.
// Only a literal ("|") block is accepted: a folded (">") block's content is
// the folded text, not the physical lines, so editing it line by line would
// be wrong. The content indentation is read from the first non-blank body
// line, so an explicit block indentation indicator is handled.
//
// EditBlockScalar operates on a single-document stream. It returns a
// TypeInvalidInput error with [CodeYAMLInvalid] or [CodeYAMLEmpty] for
// invalid or empty input, [CodeMultipleDocuments] for a multi-document
// stream, [CodePathNotFound] when outerPath does not resolve, and
// [CodeNotBlockScalar] when outerPath does not resolve to a non-empty
// literal block scalar. It passes through any error edit returns, and
// returns [CodeYAMLInvalid] if the spliced document no longer parses.
func EditBlockScalar(y YAML, outerPath []string, edit func(inner YAML) (YAML, error)) (YAML, error) {
	docs, err := ToJSON(y)
	if err != nil {
		return nil, err
	}

	if len(docs) != 1 {
		return nil, errors.Newf(errors.TypeInvalidInput, CodeMultipleDocuments, "edit operates on a single document, got %d", len(docs))
	}

	rn, err := kyyaml.Parse(string(y))
	if err != nil {
		return nil, errors.Wrap(err, errors.TypeInvalidInput, CodeYAMLInvalid, "yaml is invalid")
	}

	node, err := rn.Pipe(kyyaml.Lookup(outerPath...))
	if err != nil {
		return nil, errors.Wrapf(err, errors.TypeInvalidInput, CodePathNotFound, "could not resolve path %v", outerPath)
	}

	if node == nil {
		return nil, errors.Newf(errors.TypeInvalidInput, CodePathNotFound, "path %v not found", outerPath)
	}

	yn := node.YNode()
	if yn.Kind != kyyaml.ScalarNode || yn.Style != kyyaml.LiteralStyle {
		return nil, errors.Newf(errors.TypeInvalidInput, CodeNotBlockScalar, "path %v is not a literal block scalar", outerPath)
	}

	return editBlock(y, yn, edit)
}

// editBlock splices the block scalar whose header is on yn.Line: it finds
// the body's line range, dedents it, hands it to edit, and splices the
// result back into the document. It leaves every byte outside the body
// untouched, and inside the body restores unchanged lines from their
// original bytes.
func editBlock(src YAML, yn *kyyaml.Node, edit func(YAML) (YAML, error)) (YAML, error) {
	lines := bytes.Split(src, []byte("\n"))

	headerIdx := yn.Line - 1
	if headerIdx < 0 || headerIdx >= len(lines) {
		return nil, errors.Newf(errors.TypeInvalidInput, CodeYAMLInvalid, "block header is on line %d, out of range", yn.Line)
	}

	// An empty block has no body: bail before the scan, which would otherwise
	// mistake a following sibling line for the body.
	if strings.TrimRight(yn.Value, "\n") == "" {
		return nil, errors.Newf(errors.TypeInvalidInput, CodeNotBlockScalar, "block scalar at line %d is empty", yn.Line)
	}

	// The body runs from the line after the header. The first non-blank line
	// sets the block's content indentation; the body ends at the first
	// non-blank line indented less than that (its parent key, or a sibling,
	// however the header is nested). Interior blank lines stay in the body.
	bodyStart := headerIdx + 1
	bodyIndent := -1
	lastContent := -1

	for i := bodyStart; i < len(lines); i++ {
		if len(bytes.TrimSpace(lines[i])) == 0 {
			continue
		}

		indent := leadingSpaces(lines[i])
		if bodyIndent == -1 {
			bodyIndent = indent
		} else if indent < bodyIndent {
			break
		}

		lastContent = i
	}

	if lastContent == -1 {
		return nil, errors.Newf(errors.TypeInvalidInput, CodeNotBlockScalar, "block scalar at line %d has no content to edit", yn.Line)
	}

	orig := lines[bodyStart : lastContent+1]
	innerLines := dedentLines(orig, bodyIndent)
	inner := bytes.Join(innerLines, []byte("\n"))

	// The dedented body must equal the scalar's decoded value, which yaml.v3
	// has normalized to "\n" line endings, so compare with carriage returns
	// removed. A mismatch means the body boundary was misread — such as an
	// explicit indentation indicator ("|2") the scan does not resolve — so
	// refuse rather than hand the callback the wrong text.
	if strings.TrimRight(strings.ReplaceAll(string(inner), "\r", ""), "\n") != strings.TrimRight(yn.Value, "\n") {
		return nil, errors.Newf(errors.TypeInvalidInput, CodeNotBlockScalar, "could not isolate the block scalar body at line %d", yn.Line)
	}

	edited, err := edit(YAML(inner))
	if err != nil {
		return nil, err
	}

	body := reindentBody(orig, innerLines, bytes.Split([]byte(edited), []byte("\n")), bodyIndent)

	out := make([][]byte, 0, len(lines))
	out = append(out, lines[:bodyStart]...)
	out = append(out, body...)
	out = append(out, lines[lastContent+1:]...)

	result := YAML(bytes.Join(out, []byte("\n")))

	// Guard against a splice, or an edit, that produced a malformed document.
	if _, err := ToJSON(result); err != nil {
		return nil, errors.Wrap(err, errors.TypeInvalidInput, CodeYAMLInvalid, "edited document is not valid yaml")
	}

	return result, nil
}

// leadingSpaces counts the space characters at the start of line.
func leadingSpaces(line []byte) int {
	n := 0
	for n < len(line) && line[n] == ' ' {
		n++
	}

	return n
}

// dedentLines strips up to n leading spaces from each line (a blank line
// keeps what it has, which may be fewer).
func dedentLines(lines [][]byte, n int) [][]byte {
	out := make([][]byte, len(lines))
	for i, line := range lines {
		strip := n
		if sp := leadingSpaces(line); sp < strip {
			strip = sp
		}

		out[i] = line[strip:]
	}

	return out
}

// reindentBody rebuilds the block body. A line the callback left unchanged
// is restored from its original source bytes, so comments, trailing
// whitespace and carriage returns survive; a changed line is re-indented by
// n. If the callback changed the line count the lines can no longer be
// paired, so every line is re-indented.
func reindentBody(orig, innerLines, editedLines [][]byte, n int) [][]byte {
	pad := bytes.Repeat([]byte(" "), n)

	reindent := func(line []byte) []byte {
		if len(bytes.TrimSpace(line)) == 0 {
			return line
		}

		return append(append([]byte{}, pad...), line...)
	}

	out := make([][]byte, len(editedLines))
	paired := len(editedLines) == len(innerLines)

	for i, line := range editedLines {
		if paired && bytes.Equal(line, innerLines[i]) {
			out[i] = orig[i]
			continue
		}

		out[i] = reindent(line)
	}

	return out
}
