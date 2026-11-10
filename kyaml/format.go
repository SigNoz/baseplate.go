package kyaml

import (
	"bytes"

	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/filters"
	kustomizeyaml "sigs.k8s.io/kustomize/kyaml/yaml"

	"github.com/SigNoz/baseplate.go/errors"
)

const kustomizationKind = "Kustomization"

// Format returns y with its fields in canonical Kubernetes order and its
// layout normalized. Comments are kept, and documents without content,
// such as a leading "---" or a document of only comments, are dropped.
// Formatting formatted YAML returns it unchanged.
//
// It returns a [github.com/SigNoz/baseplate.go/errors.TypeInvalidInput]
// error with [CodeYAMLInvalid] when y isn't valid YAML.
func Format(y YAML) (YAML, error) {
	var out bytes.Buffer

	err := kio.Pipeline{
		Inputs:  []kio.Reader{&kio.ByteReader{Reader: bytes.NewReader(y)}},
		Filters: []kio.Filter{filters.FormatFilter{}, kio.FilterFunc(orderKustomization)},
		Outputs: []kio.Writer{kio.ByteWriter{Writer: &out}},
	}.Execute()
	if err != nil {
		return nil, errors.Wrap(err, errors.TypeInvalidInput, CodeYAMLInvalid, "yaml is invalid")
	}

	return out.Bytes(), nil
}

// orderKustomization moves apiVersion and kind back to the top of
// Kustomization documents. FormatFilter ranks a top-level namespace above
// them at every level, which would put namespace first in a Kustomization.
func orderKustomization(nodes []*kustomizeyaml.RNode) ([]*kustomizeyaml.RNode, error) {
	for _, node := range nodes {
		ynode := node.YNode()
		if ynode == nil || ynode.Kind != kustomizeyaml.MappingNode {
			continue
		}

		if topLevelValue(ynode, "kind") != kustomizationKind {
			continue
		}

		moveKeyToFront(ynode, "kind")
		moveKeyToFront(ynode, "apiVersion")
	}

	return nodes, nil
}

func topLevelValue(node *kustomizeyaml.Node, key string) string {
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1].Value
		}
	}

	return ""
}

func moveKeyToFront(node *kustomizeyaml.Node, key string) {
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value != key {
			continue
		}

		if i == 0 {
			return
		}

		k, v := node.Content[i], node.Content[i+1]
		node.Content = append(node.Content[:i], node.Content[i+2:]...)
		node.Content = append([]*kustomizeyaml.Node{k, v}, node.Content...)

		return
	}
}
