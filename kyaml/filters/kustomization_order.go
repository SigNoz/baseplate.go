package filters

import (
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const kustomizationKind = "Kustomization"

// KustomizationOrder moves apiVersion and kind to the top of every
// Kustomization document. The kustomize formatter,
// [sigs.k8s.io/kustomize/kyaml/kio/filters.FormatFilter], ranks a top-level
// namespace above them, so run KustomizationOrder after it.
type KustomizationOrder struct{}

var _ kio.Filter = KustomizationOrder{}

// Filter moves apiVersion and kind to the front of each Kustomization in
// nodes, in place, and returns nodes. Documents of another kind and
// documents that aren't mappings are left alone. It never returns an error.
func (KustomizationOrder) Filter(nodes []*yaml.RNode) ([]*yaml.RNode, error) {
	for _, node := range nodes {
		ynode := node.YNode()
		if ynode == nil || ynode.Kind != yaml.MappingNode {
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

func topLevelValue(node *yaml.Node, key string) string {
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1].Value
		}
	}

	return ""
}

func moveKeyToFront(node *yaml.Node, key string) {
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value != key {
			continue
		}

		if i == 0 {
			return
		}

		k, v := node.Content[i], node.Content[i+1]
		node.Content = append(node.Content[:i], node.Content[i+2:]...)
		node.Content = append([]*yaml.Node{k, v}, node.Content...)

		return
	}
}
