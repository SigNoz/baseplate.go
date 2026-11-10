package filters

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestKustomizationOrder(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Kustomization_NamespaceFirst_ApiVersionAndKindMovedUp",
			input:    "namespace: alpha\nresources:\n- bravo.yaml\nkind: Kustomization\napiVersion: kustomize.config.k8s.io/v1beta1\n",
			expected: "apiVersion: kustomize.config.k8s.io/v1beta1\nkind: Kustomization\nnamespace: alpha\nresources:\n- bravo.yaml\n",
		},
		{
			name:     "Kustomization_AlreadyOrdered_Unchanged",
			input:    "apiVersion: kustomize.config.k8s.io/v1beta1\nkind: Kustomization\nnamespace: charlie\n",
			expected: "apiVersion: kustomize.config.k8s.io/v1beta1\nkind: Kustomization\nnamespace: charlie\n",
		},
		{
			name:     "Kustomization_NoApiVersion_KindMovedUp",
			input:    "namespace: delta\nkind: Kustomization\n",
			expected: "kind: Kustomization\nnamespace: delta\n",
		},
		{
			name:     "Deployment_NamespaceFirst_Unchanged",
			input:    "namespace: echo\nkind: Deployment\napiVersion: apps/v1\n",
			expected: "namespace: echo\nkind: Deployment\napiVersion: apps/v1\n",
		},
		{
			name:     "Sequence_NotAMapping_Unchanged",
			input:    "- kind\n- Kustomization\n",
			expected: "- kind\n- Kustomization\n",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			node, err := yaml.Parse(testCase.input)
			require.NoError(t, err)

			nodes, err := KustomizationOrder{}.Filter([]*yaml.RNode{node})
			require.NoError(t, err)
			require.Len(t, nodes, 1)

			out, err := nodes[0].String()
			require.NoError(t, err)
			assert.Equal(t, testCase.expected, out)
		})
	}
}
