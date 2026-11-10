package kyaml

import (
	"bytes"

	"sigs.k8s.io/kustomize/kyaml/kio"
	kustomizefilters "sigs.k8s.io/kustomize/kyaml/kio/filters"

	"github.com/SigNoz/baseplate.go/errors"
	"github.com/SigNoz/baseplate.go/kyaml/filters"
)

// Format returns y with its fields in canonical Kubernetes order and its
// layout normalized. It runs kustomize's
// [sigs.k8s.io/kustomize/kyaml/kio/filters.FormatFilter] and then
// [filters.KustomizationOrder] on every document. Comments are kept, a List
// stays one document with its items formatted inside it, and documents
// without content, such as a leading "---" or a document of only comments,
// are dropped. Formatting formatted YAML returns it unchanged.
//
// It returns a [github.com/SigNoz/baseplate.go/errors.TypeInvalidInput]
// error with [CodeYAMLInvalid] when y isn't valid YAML.
func Format(y YAML) (YAML, error) {
	var out bytes.Buffer

	err := kio.Pipeline{
		Inputs:  []kio.Reader{&kio.ByteReader{Reader: bytes.NewReader(y), DisableUnwrapping: true}},
		Filters: []kio.Filter{kustomizefilters.FormatFilter{}, filters.KustomizationOrder{}},
		Outputs: []kio.Writer{kio.ByteWriter{Writer: &out}},
	}.Execute()
	if err != nil {
		return nil, errors.Wrap(err, errors.TypeInvalidInput, CodeYAMLInvalid, "yaml is invalid")
	}

	return out.Bytes(), nil
}
