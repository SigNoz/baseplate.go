// Package filters holds the [sigs.k8s.io/kustomize/kyaml/kio.Filter]
// implementations that [github.com/SigNoz/baseplate.go/kyaml.Format] runs
// after kustomize's formatter.
//
// A filter receives the documents of a YAML stream as
// [sigs.k8s.io/kustomize/kyaml/yaml.RNode] values and returns the
// documents to write. Run one in a
// [sigs.k8s.io/kustomize/kyaml/kio.Pipeline]:
//
//	err := kio.Pipeline{
//		Inputs:  []kio.Reader{&kio.ByteReader{Reader: bytes.NewReader(input)}},
//		Filters: []kio.Filter{kustomizefilters.FormatFilter{}, filters.KustomizationOrder{}},
//		Outputs: []kio.Writer{kio.ByteWriter{Writer: &out}},
//	}.Execute()
//
// [KustomizationOrder] keeps apiVersion and kind at the top of a
// Kustomization, where kustomize's formatter would put a top-level
// namespace first.
package filters
