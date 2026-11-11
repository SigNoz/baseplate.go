package kyaml

import (
	"bytes"

	kyyaml "sigs.k8s.io/kustomize/kyaml/yaml"

	"github.com/SigNoz/baseplate.go/errors"
)

// Set returns y with the scalar at path set to value, preserving every
// other byte of the document: field order, comments, quoting, indentation,
// and line endings. It locates the scalar through the parsed node tree,
// then rewrites just that value's source bytes in place. So, unlike
// [ToJSON] and [FromJSON] (which round-trip through JSON and reformat), the
// result differs from y by only the one value. Use it to change a single
// field of a hand-authored manifest without reformatting the rest.
//
// Path is a kustomize kyaml lookup path; a sequence element is selected
// with a "[field=value]" segment, e.g.
//
//	Set(overlay, "40Gi",
//	    "spec", "templates", "volumeClaimTemplates", "[name=data-0]",
//	    "spec", "resources", "requests", "storage")
//
// Set edits in place only when it can do so without changing the
// document's meaning, so its contract is deliberately narrow. value must be
// a safe plain scalar: it has to start with an ASCII letter or digit and
// hold only those plus "._+/=-", which fits a quantity, size, or image tag.
// Such a value can never introduce a comment, key, quote, or newline; any
// other value is refused with [CodeUnsafeValue]. The target must be a plain
// or double-quoted scalar whose text appears verbatim at its node position.
// A single-quoted, block, folded, or otherwise escaped scalar is refused
// with [CodeNotScalar] rather than edited by searching the line, which
// could match a comment.
//
// Set operates on a single-document stream. It returns a TypeInvalidInput
// error with [CodeYAMLInvalid] or [CodeYAMLEmpty] for invalid or empty
// input, [CodeMultipleDocuments] for a multi-document stream,
// [CodePathNotFound] when path does not resolve, [CodeNotScalar] when path
// does not resolve to an in-place-editable scalar, and [CodeUnsafeValue]
// when value is not a safe plain scalar.
func Set(y YAML, value string, path ...string) (YAML, error) {
	// Reuse ToJSON only to validate and count content documents (it reports
	// empty/invalid the same way the other entry points do); the edit itself
	// is a byte rewrite driven by the node tree, not a JSON round-trip.
	docs, err := ToJSON(y)
	if err != nil {
		return nil, err
	}

	if len(docs) != 1 {
		return nil, errors.Newf(errors.TypeInvalidInput, CodeMultipleDocuments, "set operates on a single document, got %d", len(docs))
	}

	if !isSafePlainScalar(value) {
		return nil, errors.Newf(errors.TypeInvalidInput, CodeUnsafeValue, "value %q is not a safe plain scalar", value)
	}

	rn, err := kyyaml.Parse(string(y))
	if err != nil {
		return nil, errors.Wrap(err, errors.TypeInvalidInput, CodeYAMLInvalid, "yaml is invalid")
	}

	node, err := rn.Pipe(kyyaml.Lookup(path...))
	if err != nil {
		return nil, errors.Wrapf(err, errors.TypeInvalidInput, CodePathNotFound, "could not resolve path %v", path)
	}

	if node == nil {
		return nil, errors.Newf(errors.TypeInvalidInput, CodePathNotFound, "path %v not found", path)
	}

	yn := node.YNode()
	if yn.Kind != kyyaml.ScalarNode || yn.Value == "" {
		return nil, errors.Newf(errors.TypeInvalidInput, CodeNotScalar, "path %v is not a non-empty scalar", path)
	}

	return rewriteScalar(y, yn, value)
}

// isSafePlainScalar reports whether s can stand as a plain YAML scalar with
// no quoting: non-empty, starting with an ASCII letter or digit, and
// otherwise only those plus "._+/=-". This is the conservative subset Set
// accepts as a replacement value, so a value can never carry a comment,
// key, quote, or newline into the document.
func isSafePlainScalar(s string) bool {
	if s == "" {
		return false
	}

	for i, r := range s {
		alnum := r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9'
		switch {
		case alnum:
		case i > 0 && (r == '.' || r == '_' || r == '+' || r == '/' || r == '=' || r == '-'):
		default:
			return false
		}
	}

	return true
}

// rewriteScalar replaces the scalar yn's value with value, editing only the
// scalar's own source bytes on its line so comments, quoting, spacing, and
// indentation survive. It anchors at the node's line and column rather than
// searching the line, so a comment that happens to contain the decoded
// value is never touched. A scalar whose source is not present verbatim at
// that position, such as a single-quoted value using doubled-quote escapes
// or a block or folded value, is reported as [CodeNotScalar].
func rewriteScalar(src YAML, yn *kyyaml.Node, value string) (YAML, error) {
	lines := bytes.Split(src, []byte("\n"))

	i := yn.Line - 1
	if i < 0 || i >= len(lines) {
		return nil, errors.Newf(errors.TypeInvalidInput, CodeYAMLInvalid, "scalar is on line %d, out of range", yn.Line)
	}

	// Column is a 1-based rune index; convert it to a byte offset.
	runes := []rune(string(lines[i]))
	if yn.Column < 1 || yn.Column-1 > len(runes) {
		return nil, errors.Newf(errors.TypeInvalidInput, CodeYAMLInvalid, "scalar column %d is out of range on line %d", yn.Column, yn.Line)
	}

	start := len(string(runes[:yn.Column-1]))

	offset, oldLen, ok := scalarSpan(lines[i][start:], yn)
	if !ok {
		return nil, errors.Newf(errors.TypeInvalidInput, CodeNotScalar, "scalar at line %d column %d is not editable in place", yn.Line, yn.Column)
	}

	at := start + offset
	edited := make([]byte, 0, len(lines[i])-oldLen+len(value))
	edited = append(edited, lines[i][:at]...)
	edited = append(edited, value...)
	edited = append(edited, lines[i][at+oldLen:]...)
	lines[i] = edited

	return YAML(bytes.Join(lines, []byte("\n"))), nil
}

// scalarSpan returns the byte offset and length, within tail, of the
// editable value of scalar yn, where tail is the source line starting at
// yn's column. It handles a plain scalar (its value is the leading bytes of
// tail) and a double-quoted scalar whose value needs no escaping (the bytes
// between the quotes equal the decoded value). For any other style, or when
// the source does not match verbatim, it reports ok false.
func scalarSpan(tail []byte, yn *kyyaml.Node) (offset, length int, ok bool) {
	v := []byte(yn.Value)

	switch yn.Style {
	case 0:
		if bytes.HasPrefix(tail, v) {
			return 0, len(v), true
		}
	case kyyaml.DoubleQuotedStyle:
		if len(tail) > len(v)+1 && tail[0] == '"' && tail[len(v)+1] == '"' && bytes.HasPrefix(tail[1:], v) {
			return 1, len(v), true
		}
	}

	return 0, 0, false
}
