package kyaml

import (
	"bytes"

	kyyaml "sigs.k8s.io/kustomize/kyaml/yaml"

	"github.com/SigNoz/baseplate.go/errors"
)

// Set returns y with the scalar at path set to value, preserving every
// other byte of the document: field order, comments, quoting, indentation,
// and line endings. It finds the scalar through the parsed node tree, then
// rewrites only that value in place. So, unlike [ToJSON] and [FromJSON]
// (which round-trip through JSON and reformat), the result differs from y
// by just the one value. Use it to change a single field of a hand-authored
// manifest without reformatting the rest.
//
// Path is a kustomize kyaml lookup path; a sequence element is selected
// with a "[field=value]" segment, e.g.
//
//	Set(overlay, "40Gi",
//	    "spec", "templates", "volumeClaimTemplates", "[name=data-0]",
//	    "spec", "resources", "requests", "storage")
//
// Set operates on a single-document stream. It returns a TypeInvalidInput
// error with [CodeYAMLInvalid] or [CodeYAMLEmpty] for invalid or empty
// input, [CodeMultipleDocuments] for a multi-document stream,
// [CodePathNotFound] when path does not resolve, and [CodeNotScalar] when
// path resolves to a mapping, a sequence, an empty scalar, or a scalar Set
// cannot edit in place (a block, folded, or multi-line value).
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

	return rewriteScalar(y, yn.Line, yn.Column, yn.Value, value)
}

// rewriteScalar replaces old with value at the scalar's 1-based line and
// column in src, leaving every other byte unchanged (so comments, quoting,
// spacing and indentation survive). It errors rather than guess if old is
// not at that position, e.g. a block or multi-line scalar.
func rewriteScalar(src YAML, line, column int, old, value string) (YAML, error) {
	lines := bytes.Split(src, []byte("\n"))

	i := line - 1
	if i < 0 || i >= len(lines) {
		return nil, errors.Newf(errors.TypeInvalidInput, CodeYAMLInvalid, "value is on line %d, out of range", line)
	}

	// Column is a 1-based rune index; convert it to a byte offset into the line.
	runes := []rune(string(lines[i]))
	if column < 1 || column-1 > len(runes) {
		return nil, errors.Newf(errors.TypeInvalidInput, CodeYAMLInvalid, "value column %d is out of range on line %d", column, line)
	}

	start := len(string(runes[:column-1]))

	rel := bytes.Index(lines[i][start:], []byte(old))
	if rel < 0 {
		// A block, folded, line-wrapped, or escaped scalar does not appear verbatim here: it is a
		// valid scalar Set cannot edit in place, not invalid input.
		return nil, errors.Newf(errors.TypeInvalidInput, CodeNotScalar, "value %q is not an inline scalar at line %d column %d", old, line, column)
	}

	at := start + rel
	edited := make([]byte, 0, len(lines[i])-len(old)+len(value))
	edited = append(edited, lines[i][:at]...)
	edited = append(edited, value...)
	edited = append(edited, lines[i][at+len(old):]...)
	lines[i] = edited

	return YAML(bytes.Join(lines, []byte("\n"))), nil
}
