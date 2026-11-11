package kyaml

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/SigNoz/baseplate.go/errors"
)

// tStoragePath is the lookup path to a named volumeClaimTemplate's storage.
func tStoragePath(vct string) []string {
	return []string{"spec", "templates", "volumeClaimTemplates", "[name=" + vct + "]", "spec", "resources", "requests", "storage"}
}

// tChangedLines returns the lines that differ between before and after, requiring an equal count.
func tChangedLines(t *testing.T, before, after string) []string {
	t.Helper()

	b := strings.Split(before, "\n")
	a := strings.Split(after, "\n")
	require.Equal(t, len(b), len(a), "line count must not change")

	var changed []string
	for i := range b {
		if b[i] != a[i] {
			changed = append(changed, a[i])
		}
	}

	return changed
}

func TestSet(t *testing.T) {
	testCases := []struct {
		name         string
		input        string
		expectedLine string   // the single line expected after the edit
		expectedKept []string // substrings that must still be present
	}{
		{
			name:         "NonCanonicalOrderWithComment_OrderCommentSiblingKept",
			input:        "apiVersion: clickhouse.altinity.com/v1\nkind: ClickHouseInstallation\nmetadata:\n  name: clickhouse # legend: managed by zeus\nspec:\n  templates:\n    podTemplates:\n    - name: default\n    volumeClaimTemplates:\n    - name: data-0\n      spec:\n        resources:\n          requests:\n            storage: 30Gi\n    - name: default\n      spec:\n        resources:\n          requests:\n            storage: 15Gi\n",
			expectedLine: "            storage: 60Gi",
			expectedKept: []string{"# legend: managed by zeus", "storage: 15Gi", "podTemplates:"},
		},
		{
			name:         "ExtraSpacesAfterColon_SpacingKept",
			input:        "spec:\n  templates:\n    volumeClaimTemplates:\n    - name: data-0\n      spec:\n        resources:\n          requests:\n            storage:    30Gi\n",
			expectedLine: "            storage:    60Gi",
		},
		{
			name:         "WideSequenceIndent_IndentKept",
			input:        "spec:\n  templates:\n    volumeClaimTemplates:\n      - name: data-0\n        spec:\n          resources:\n            requests:\n              storage: 30Gi\n",
			expectedLine: "              storage: 60Gi",
		},
		{
			name:         "LeadingDocumentSeparator_SeparatorKept",
			input:        "---\nspec:\n  templates:\n    volumeClaimTemplates:\n    - name: data-0\n      spec:\n        resources:\n          requests:\n            storage: 30Gi\n",
			expectedLine: "            storage: 60Gi",
			expectedKept: []string{"---"},
		},
		{
			name:         "DoubleQuotedValue_QuotesKept",
			input:        "spec:\n  templates:\n    volumeClaimTemplates:\n    - name: data-0\n      spec:\n        resources:\n          requests:\n            storage: \"30Gi\"\n",
			expectedLine: "            storage: \"60Gi\"",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			out, err := Set(YAML(testCase.input), "60Gi", tStoragePath("data-0")...)
			require.NoError(t, err)

			changed := tChangedLines(t, testCase.input, string(out))
			require.Len(t, changed, 1, "only the targeted value may change")
			assert.Equal(t, testCase.expectedLine, changed[0])

			for _, kept := range testCase.expectedKept {
				assert.Contains(t, string(out), kept)
			}
		})
	}
}

func TestSet_Errors(t *testing.T) {
	testCases := []struct {
		name         string
		input        string
		path         []string
		expectedCode errors.Code
	}{
		{name: "UnknownKey_PathNotFound", input: "spec:\n  a: 1\n", path: []string{"spec", "b"}, expectedCode: CodePathNotFound},
		{name: "AbsentSequenceElement_PathNotFound", input: "spec:\n  templates:\n    volumeClaimTemplates:\n    - name: data-0\n      spec:\n        resources:\n          requests:\n            storage: 30Gi\n", path: tStoragePath("absent"), expectedCode: CodePathNotFound},
		{name: "MappingTarget_NotScalar", input: "spec:\n  templates:\n    a: b\n", path: []string{"spec", "templates"}, expectedCode: CodeNotScalar},
		{name: "TwoContentDocuments_MultipleDocuments", input: "kind: A\n---\nkind: B\n", path: []string{"kind"}, expectedCode: CodeMultipleDocuments},
		{name: "UnclosedFlowSequence_YAMLInvalid", input: "kind: [ConfigMap\n", path: []string{"kind"}, expectedCode: CodeYAMLInvalid},
		{name: "Empty_YAMLEmpty", input: "", path: []string{"kind"}, expectedCode: CodeYAMLEmpty},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := Set(YAML(testCase.input), "x", testCase.path...)
			require.Error(t, err)
			assert.Equal(t, errors.TypeInvalidInput, errors.TypeOf(err))
			assert.Equal(t, testCase.expectedCode, errors.CodeOf(err))
		})
	}
}
