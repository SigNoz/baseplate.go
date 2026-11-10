package kyaml

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/SigNoz/baseplate.go/errors"
)

// The expected files are fixed, so a mismatch is a change in behavior.
func TestToJSON_FromJSON(t *testing.T) {
	testCases := []struct {
		name string
		file string
	}{
		{name: "Deployment_Comments_DroppedAndCanonicalOrder", file: "deployment_comments"},
		{name: "ConfigMap_LongLine_NotWrapped", file: "configmap_long_line"},
		{name: "CustomResource_NestedSequences_CanonicalOrder", file: "custom_resource"},
		{name: "HelmValues_NoKind_SortedKeys", file: "helm_values"},
		{name: "Kustomization_Namespace_ApiVersionAndKindFirst", file: "kustomization_namespace"},
		{name: "MultiDocument_EmptyDocuments_Skipped", file: "multi_document"},
		{name: "Quoting_AmbiguousStrings_Yaml11Quoting", file: "quoting"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			input := tRead(t, "json", testCase.file+".input.yaml")
			expectedJSON := bytes.Split(bytes.TrimSuffix(tRead(t, "json", testCase.file+".expected.jsonl"), []byte("\n")), []byte("\n"))
			expectedYAML := tRead(t, "json", testCase.file+".expected.yaml")

			docs, err := ToJSON(input)
			require.NoError(t, err)
			require.Len(t, docs, len(expectedJSON))

			for i, doc := range docs {
				assert.Equal(t, string(expectedJSON[i]), string(doc))
			}

			out, err := FromJSON(docs...)
			require.NoError(t, err)
			assert.Equal(t, string(expectedYAML), string(out))

			formatted, err := Format(out)
			require.NoError(t, err)
			assert.Equal(t, string(out), string(formatted))
		})
	}
}

// Documents that start with "{" are converted like any other document,
// whether they are YAML flow mappings or JSON.
func TestToJSON_FlowStyle(t *testing.T) {
	testCases := []struct {
		name     string
		input    YAML
		expected []string
	}{
		{
			name:     "FlowMapping_ConvertedToJSON",
			input:    YAML("{apiVersion: v1, kind: ConfigMap, metadata: {name: quebec}, data: {mode: debug}}\n"),
			expected: []string{`{"apiVersion":"v1","data":{"mode":"debug"},"kind":"ConfigMap","metadata":{"name":"quebec"}}`},
		},
		{
			name:     "JSONDocument_Compacted",
			input:    YAML("{\"metadata\": {\"name\": \"romeo\"}, \"kind\": \"Namespace\", \"apiVersion\": \"v1\"}\n"),
			expected: []string{`{"apiVersion":"v1","kind":"Namespace","metadata":{"name":"romeo"}}`},
		},
		{
			name:     "FlowMappingAfterBlockDocument_BothConverted",
			input:    YAML("kind: ServiceAccount\n---\n{kind: Secret}\n"),
			expected: []string{`{"kind":"ServiceAccount"}`, `{"kind":"Secret"}`},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			docs, err := ToJSON(testCase.input)
			require.NoError(t, err)
			require.Len(t, docs, len(testCase.expected))

			for i, doc := range docs {
				assert.Equal(t, testCase.expected[i], string(doc))
			}
		})
	}
}

func TestToJSON_Invalid(t *testing.T) {
	testCases := []struct {
		name         string
		input        YAML
		expectedCode errors.Code
	}{
		{name: "UnclosedFlowSequence_YAMLInvalid", input: YAML("kind: [ConfigMap\n"), expectedCode: CodeYAMLInvalid},
		{name: "Empty_YAMLEmpty", input: YAML(""), expectedCode: CodeYAMLEmpty},
		{name: "SeparatorsOnly_YAMLEmpty", input: YAML("---\n---\n"), expectedCode: CodeYAMLEmpty},
		{name: "CommentsOnly_YAMLEmpty", input: YAML("# oscar\n---\n# papa\n"), expectedCode: CodeYAMLEmpty},
		{name: "BrokenFlowMapping_YAMLInvalid", input: YAML("{apiVersion: v1, kind: ConfigMap,, }\n"), expectedCode: CodeYAMLInvalid},
		{name: "BrokenJSON_YAMLInvalid", input: YAML(`{"kind": "ConfigMap",, }` + "\n"), expectedCode: CodeYAMLInvalid},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := ToJSON(testCase.input)
			require.Error(t, err)
			assert.Equal(t, errors.TypeInvalidInput, errors.TypeOf(err))
			assert.Equal(t, testCase.expectedCode, errors.CodeOf(err))
		})
	}
}

func TestFromJSON(t *testing.T) {
	testCases := []struct {
		name         string
		docs         []json.RawMessage
		expected     YAML
		expectedCode errors.Code
	}{
		{name: "NoDocuments_Empty", docs: nil, expected: YAML{}},
		{name: "TwoDocuments_JoinedWithSeparator", docs: []json.RawMessage{json.RawMessage(`{"kind":"Namespace","apiVersion":"v1","metadata":{"name":"quebec"}}`), json.RawMessage(`{"romeo":"on"}`)}, expected: YAML("apiVersion: v1\nkind: Namespace\nmetadata:\n  name: quebec\n---\nromeo: \"on\"\n")},
		{name: "List_WrapperKept", docs: []json.RawMessage{json.RawMessage(`{"kind":"List","apiVersion":"v1","items":[{"kind":"ConfigMap","apiVersion":"v1","metadata":{"name":"oscar"}}]}`)}, expected: YAML("apiVersion: v1\nkind: List\nitems:\n- apiVersion: v1\n  kind: ConfigMap\n  metadata:\n    name: oscar\n")},
		{name: "TruncatedObject_JSONInvalid", docs: []json.RawMessage{json.RawMessage(`{"sierra":`)}, expectedCode: CodeJSONInvalid},
		{name: "SecondDocumentInvalid_JSONInvalid", docs: []json.RawMessage{json.RawMessage(`{"tango":1}`), json.RawMessage(`[uniform`)}, expectedCode: CodeJSONInvalid},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			out, err := FromJSON(testCase.docs...)
			if testCase.expectedCode != (errors.Code{}) {
				require.Error(t, err)
				assert.Equal(t, errors.TypeInvalidInput, errors.TypeOf(err))
				assert.Equal(t, testCase.expectedCode, errors.CodeOf(err))

				return
			}

			require.NoError(t, err)
			assert.Equal(t, string(testCase.expected), string(out))
		})
	}
}
