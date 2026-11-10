package filters_test

import (
	"bytes"
	"fmt"

	"sigs.k8s.io/kustomize/kyaml/kio"
	kustomizefilters "sigs.k8s.io/kustomize/kyaml/kio/filters"

	"github.com/SigNoz/baseplate.go/kyaml/filters"
)

// Without KustomizationOrder, the formatter would write namespace first.
func ExampleKustomizationOrder() {
	input := []byte("resources:\n- myapp.yaml\nnamespace: myapp\nkind: Kustomization\napiVersion: kustomize.config.k8s.io/v1beta1\n")

	var out bytes.Buffer

	err := kio.Pipeline{
		Inputs:  []kio.Reader{&kio.ByteReader{Reader: bytes.NewReader(input)}},
		Filters: []kio.Filter{kustomizefilters.FormatFilter{}, filters.KustomizationOrder{}},
		Outputs: []kio.Writer{kio.ByteWriter{Writer: &out}},
	}.Execute()
	if err != nil {
		_, _ = fmt.Println(err)

		return
	}

	_, _ = fmt.Print(out.String())
	// Output:
	// apiVersion: kustomize.config.k8s.io/v1beta1
	// kind: Kustomization
	// namespace: myapp
	// resources:
	// - myapp.yaml
}
