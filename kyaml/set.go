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
// path resolves to a mapping or sequence rather than a scalar.
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
	if yn.Kind != kyyaml.ScalarNode {
		return nil, errors.Newf(errors.TypeInvalidInput, CodeNotScalar, "path %v is not a scalar", path)
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

	start := column - 1
	if start < 0 || start > len(lines[i]) {
		return nil, errors.Newf(errors.TypeInvalidInput, CodeYAMLInvalid, "value column %d is out of range on line %d", column, line)
	}

	rel := bytes.Index(lines[i][start:], []byte(old))
	if rel < 0 {
		return nil, errors.Newf(errors.TypeInvalidInput, CodeYAMLInvalid, "value %q not found at line %d column %d", old, line, column)
	}

	at := start + rel
	edited := make([]byte, 0, len(lines[i])-len(old)+len(value))
	edited = append(edited, lines[i][:at]...)
	edited = append(edited, value...)
	edited = append(edited, lines[i][at+len(old):]...)
	lines[i] = edited

	return YAML(bytes.Join(lines, []byte("\n"))), nil
}
