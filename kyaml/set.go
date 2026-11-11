package kyaml

import (
	kyyaml "sigs.k8s.io/kustomize/kyaml/yaml"

	"github.com/SigNoz/baseplate.go/errors"
)

// Set returns y with the scalar at path set to value, preserving the
// document's field order, comments, quoting, and layout. Unlike [ToJSON]
// and [FromJSON] — which round-trip through JSON and so reorder fields and
// drop comments — Set edits the YAML node in place, so the result differs
// from y by only the one value. Use it to change a single field of a
// hand-authored manifest without reformatting the rest (e.g. a PVC's size).
//
// The path is a kustomize kyaml lookup path; a sequence element is selected
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
	// Reuse ToJSON only to validate and count content documents (it reports empty/invalid the same
	// way the other entry points do); the edit itself goes through the node tree, not JSON.
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

	if node.YNode().Kind != kyyaml.ScalarNode {
		return nil, errors.Newf(errors.TypeInvalidInput, CodeNotScalar, "path %v is not a scalar", path)
	}

	node.YNode().Value = value

	out, err := rn.String()
	if err != nil {
		return nil, errors.Wrap(err, errors.TypeInvalidInput, CodeYAMLInvalid, "could not serialize yaml")
	}

	return YAML(out), nil
}
