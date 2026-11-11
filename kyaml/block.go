package kyaml

import (
	"bytes"

	kyyaml "sigs.k8s.io/kustomize/kyaml/yaml"

	"github.com/SigNoz/baseplate.go/errors"
)

// EditBlockScalar edits the text held in the block scalar at outerPath by
// handing its dedented content to edit and splicing the result back into
// the document, re-indented to the block's indentation. Every byte outside
// the block body is preserved, and inside it only what edit changes
// changes. So a change to one field of a config embedded in a manifest
// (such as a collector config kept in a ConfigMap) yields a diff of just
// that field. The edit callback usually composes one or more [Set] calls on
// the inner document.
//
// outerPath is a kustomize kyaml lookup path to the block scalar, a "|" or
// ">" value; a sequence element is selected with a "[field=value]" segment.
// The content indentation is read from the source lines, so an explicit
// block indentation indicator is handled.
//
// EditBlockScalar operates on a single-document stream. It returns a
// TypeInvalidInput error with [CodeYAMLInvalid] or [CodeYAMLEmpty] for
// invalid or empty input, [CodeMultipleDocuments] for a multi-document
// stream, [CodePathNotFound] when outerPath does not resolve, and
// [CodeNotBlockScalar] when outerPath does not resolve to a non-empty block
// scalar. It passes through any error edit returns, and returns
// [CodeYAMLInvalid] if the spliced document no longer parses.
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
	if yn.Kind != kyyaml.ScalarNode || yn.Style != kyyaml.LiteralStyle && yn.Style != kyyaml.FoldedStyle {
		return nil, errors.Newf(errors.TypeInvalidInput, CodeNotBlockScalar, "path %v is not a block scalar", outerPath)
	}

	return editBlock(y, yn, edit)
}

// editBlock splices the block scalar whose header is on yn.Line: it finds
// the body's line range, dedents it, hands it to edit, and re-indents the
// result back into the document. It leaves every byte outside the body
// untouched.
func editBlock(src YAML, yn *kyyaml.Node, edit func(YAML) (YAML, error)) (YAML, error) {
	lines := bytes.Split(src, []byte("\n"))

	headerIdx := yn.Line - 1
	if headerIdx < 0 || headerIdx >= len(lines) {
		return nil, errors.Newf(errors.TypeInvalidInput, CodeYAMLInvalid, "block header is on line %d, out of range", yn.Line)
	}

	// The body is the run of following lines indented deeper than the header's
	// key; it ends at the first non-blank line indented no deeper. bodyIndent
	// is the least indentation among the body's non-blank lines, the amount to
	// strip so the inner document starts at column zero.
	keyIndent := leadingSpaces(lines[headerIdx])
	bodyStart := headerIdx + 1
	lastContent := -1
	bodyIndent := -1

	for i := bodyStart; i < len(lines); i++ {
		if len(bytes.TrimSpace(lines[i])) == 0 {
			continue
		}

		indent := leadingSpaces(lines[i])
		if indent <= keyIndent {
			break
		}

		lastContent = i
		if bodyIndent == -1 || indent < bodyIndent {
			bodyIndent = indent
		}
	}

	if lastContent == -1 {
		return nil, errors.Newf(errors.TypeInvalidInput, CodeNotBlockScalar, "block scalar at line %d has no content to edit", yn.Line)
	}

	inner := dedent(lines[bodyStart:lastContent+1], bodyIndent)

	edited, err := edit(YAML(inner))
	if err != nil {
		return nil, err
	}

	reindented := indentLines(bytes.Split([]byte(edited), []byte("\n")), bodyIndent)

	out := make([][]byte, 0, len(lines))
	out = append(out, lines[:bodyStart]...)
	out = append(out, reindented...)
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

// dedent strips up to n leading spaces from each line (a blank line keeps
// what it has, which is fewer) and joins the result with newlines.
func dedent(lines [][]byte, n int) []byte {
	out := make([][]byte, len(lines))
	for i, line := range lines {
		strip := n
		if sp := leadingSpaces(line); sp < strip {
			strip = sp
		}

		out[i] = line[strip:]
	}

	return bytes.Join(out, []byte("\n"))
}

// indentLines prepends n spaces to each non-empty line, the inverse of
// dedent for the lines edit did not change. A whitespace-only line is
// rendered empty.
func indentLines(lines [][]byte, n int) [][]byte {
	pad := bytes.Repeat([]byte(" "), n)
	out := make([][]byte, len(lines))

	for i, line := range lines {
		if len(bytes.TrimSpace(line)) == 0 {
			out[i] = nil
			continue
		}

		out[i] = append(append([]byte{}, pad...), line...)
	}

	return out
}
