package kyaml

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/SigNoz/baseplate.go/errors"
)

// nonCanonical is deliberately NOT in canonical field order
// (volumeClaimTemplates sorts above podTemplates under Format) and has a
// comment, so it proves Set preserves order and comments where ToJSON and
// FromJSON would reorder and strip them.
const nonCanonical = `apiVersion: clickhouse.altinity.com/v1
kind: ClickHouseInstallation
metadata:
  name: clickhouse # legend: managed by zeus
spec:
  templates:
    podTemplates:
    - name: default
    volumeClaimTemplates:
    - name: data-0
      spec:
        resources:
          requests:
            storage: 30Gi
    - name: default
      spec:
        resources:
          requests:
            storage: 15Gi
`

func TestSet_PreservesEverythingButTheValue(t *testing.T) {
	out, err := Set(YAML(nonCanonical), "40Gi",
		"spec", "templates", "volumeClaimTemplates", "[name=data-0]",
		"spec", "resources", "requests", "storage")
	require.NoError(t, err)

	in := strings.Split(nonCanonical, "\n")
	got := strings.Split(string(out), "\n")
	require.Equal(t, len(in), len(got), "line count must not change")

	changed := 0
	for i := range in {
		if in[i] != got[i] {
			changed++
			assert.Contains(t, got[i], "40Gi")
			assert.Contains(t, in[i], "storage")
		}
	}
	assert.Equal(t, 1, changed, "only the data-0 storage line may change")

	// The comment survives and the sibling is untouched (ToJSON/FromJSON would lose both).
	assert.Contains(t, string(out), "# legend: managed by zeus")
	assert.Contains(t, string(out), "storage: 15Gi")
}

func TestSet_Errors(t *testing.T) {
	testCases := []struct {
		name         string
		input        YAML
		path         []string
		expectedCode errors.Code
	}{
		{name: "PathNotFound", input: YAML(nonCanonical), path: []string{"spec", "nope"}, expectedCode: CodePathNotFound},
		{name: "VCTMissing", input: YAML(nonCanonical), path: []string{"spec", "templates", "volumeClaimTemplates", "[name=absent]", "spec", "resources", "requests", "storage"}, expectedCode: CodePathNotFound},
		{name: "NotScalar", input: YAML(nonCanonical), path: []string{"spec", "templates"}, expectedCode: CodeNotScalar},
		{name: "MultipleDocuments", input: YAML("kind: A\n---\nkind: B\n"), path: []string{"kind"}, expectedCode: CodeMultipleDocuments},
		{name: "Invalid", input: YAML("kind: [ConfigMap\n"), path: []string{"kind"}, expectedCode: CodeYAMLInvalid},
		{name: "Empty", input: YAML(""), path: []string{"kind"}, expectedCode: CodeYAMLEmpty},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Set(tc.input, "x", tc.path...)
			require.Error(t, err)
			assert.Equal(t, errors.TypeInvalidInput, errors.TypeOf(err))
			assert.Equal(t, tc.expectedCode, errors.CodeOf(err))
		})
	}
}
