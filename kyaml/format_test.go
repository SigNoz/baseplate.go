package kyaml

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/SigNoz/baseplate.go/errors"
)

// tRead returns the contents of testdata/<dir>/<name>.
func tRead(t *testing.T, dir, name string) []byte {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("testdata", dir, name))
	require.NoError(t, err)

	return data
}

// The expected files are fixed, so a mismatch is a change in behavior.
func TestFormat(t *testing.T) {
	testCases := []struct {
		name string
		file string
	}{
		{name: "Kustomization_NamespaceAboveApiVersion_ApiVersionAndKindFirst", file: "kustomization_namespace"},
		{name: "Deployment_UnorderedFields_CanonicalOrderAndCommentsKept", file: "deployment_unordered"},
		{name: "ConfigMap_EmbeddedConfigWithLongLine_Unchanged", file: "configmap_embedded_config"},
		{name: "CustomResource_NestedSequencesAndBlockComments_CanonicalOrder", file: "custom_resource"},
		{name: "HelmValues_NoKind_Unchanged", file: "helm_values"},
		{name: "MultiDocument_EmptyAndCommentOnlyDocuments_Dropped", file: "multi_document"},
		{name: "Quoting_AmbiguousStrings_QuotesKept", file: "quoting"},
		{name: "Service_AlreadyFormatted_Unchanged", file: "already_formatted"},
		{name: "Sequence_WideIndent_Compact", file: "sequence_indent"},
		{name: "Ingress_UnorderedFields_CanonicalOrder", file: "ingress"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			input := tRead(t, "format", testCase.file+".input.yaml")
			expected := tRead(t, "format", testCase.file+".expected.yaml")

			formatted, err := Format(input)
			require.NoError(t, err)
			assert.Equal(t, string(expected), string(formatted))

			again, err := Format(formatted)
			require.NoError(t, err)
			assert.Equal(t, string(formatted), string(again))
		})
	}
}

func TestFormat_Invalid(t *testing.T) {
	testCases := []struct {
		name  string
		input YAML
	}{
		{name: "UnclosedFlowSequence_YAMLInvalid", input: YAML("kind: [ConfigMap\n")},
		{name: "BadIndentation_YAMLInvalid", input: YAML("metadata:\n  name: mike\n bad: november\n")},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := Format(testCase.input)
			require.Error(t, err)
			assert.Equal(t, errors.TypeInvalidInput, errors.TypeOf(err))
			assert.Equal(t, CodeYAMLInvalid, errors.CodeOf(err))
		})
	}
}
