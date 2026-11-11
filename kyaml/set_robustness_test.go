package kyaml

import (
	"encoding/json"
	"fmt"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tLeaves flattens the single document in y to a map of JSON path to scalar
// value, so two documents can be compared leaf by leaf regardless of field
// order or formatting. It fails the test if y is not one valid document.
func tLeaves(t *testing.T, y string) map[string]string {
	t.Helper()

	docs, err := ToJSON(YAML(y))
	require.NoError(t, err, "output must be valid YAML")
	require.Len(t, docs, 1)

	var tree any
	require.NoError(t, json.Unmarshal(docs[0], &tree))

	leaves := map[string]string{}
	tFlatten("", tree, leaves)

	return leaves
}

func tFlatten(prefix string, node any, out map[string]string) {
	switch n := node.(type) {
	case map[string]any:
		keys := make([]string, 0, len(n))
		for k := range n {
			keys = append(keys, k)
		}

		sort.Strings(keys)
		for _, k := range keys {
			tFlatten(prefix+"/"+k, n[k], out)
		}
	case []any:
		for i, v := range n {
			tFlatten(fmt.Sprintf("%s/%d", prefix, i), v, out)
		}
	default:
		out[prefix] = fmt.Sprint(node)
	}
}

// tDiffKeys returns the keys whose value differs between before and after,
// counting a key present in only one side as changed.
func tDiffKeys(before, after map[string]string) []string {
	var changed []string
	for k, bv := range before {
		if av, ok := after[k]; !ok || av != bv {
			changed = append(changed, k)
		}
	}

	for k := range after {
		if _, ok := before[k]; !ok {
			changed = append(changed, k)
		}
	}

	sort.Strings(changed)

	return changed
}

// TestSet_ChangesExactlyOneLeaf proves the defining property: a successful
// Set leaves the document valid YAML that differs from the input in exactly
// one leaf, and that leaf holds the new value. It covers the layouts a plain
// line-level diff cannot distinguish from a mis-anchored edit — a comment
// repeating the old value, a double-quoted scalar, multi-byte runes before
// the value, non-canonical order, and a value shared by two siblings.
func TestSet_ChangesExactlyOneLeaf(t *testing.T) {
	testCases := []struct {
		name         string
		input        string
		path         []string
		newValue     string
		expectedLeaf string // The JSON path expected to carry newValue.
	}{
		{
			name:         "TwoSiblingsShareOldValue_OnlyTargetChanges",
			input:        "spec:\n  templates:\n    volumeClaimTemplates:\n    - name: data-0\n      spec:\n        resources:\n          requests:\n            storage: 30Gi\n    - name: default\n      spec:\n        resources:\n          requests:\n            storage: 30Gi\n",
			path:         tStoragePath("data-0"),
			newValue:     "60Gi",
			expectedLeaf: "/spec/templates/volumeClaimTemplates/0/spec/resources/requests/storage",
		},
		{
			name:         "CommentRepeatsOldValue_CommentIsNotData",
			input:        "spec:\n  a: 30Gi # was 30Gi, grew from 30Gi\n  b: keep\n",
			path:         []string{"spec", "a"},
			newValue:     "60Gi",
			expectedLeaf: "/spec/a",
		},
		{
			name:         "DoubleQuoted_InnerChanges",
			input:        "spec:\n  a: \"30Gi\"\n  b: plain\n",
			path:         []string{"spec", "a"},
			newValue:     "60Gi",
			expectedLeaf: "/spec/a",
		},
		{
			name:         "MultiByteBeforeValue_RightLeaf",
			input:        "café: 30Gi # ☕\nother: keep\n",
			path:         []string{"café"},
			newValue:     "60Gi",
			expectedLeaf: "/café",
		},
		{
			name:         "ValueEqualToAKeyName_KeyUntouched",
			input:        "spec:\n  storage: small\n  size: storage\n",
			path:         []string{"spec", "size"},
			newValue:     "large",
			expectedLeaf: "/spec/size",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			before := tLeaves(t, testCase.input)

			out, err := Set(YAML(testCase.input), testCase.newValue, testCase.path...)
			require.NoError(t, err)

			after := tLeaves(t, string(out))

			changed := tDiffKeys(before, after)
			require.Equal(t, []string{testCase.expectedLeaf}, changed, "exactly the target leaf may change in meaning")
			assert.Equal(t, testCase.newValue, after[testCase.expectedLeaf])
		})
	}
}

// TestSet_NeverPanics feeds hostile inputs, paths, and values and only
// requires that Set returns (error or not) without panicking, and that any
// success is still valid YAML. A format-preserving byte editor driven by node
// positions must never index out of range or emit garbage.
func TestSet_NeverPanics(t *testing.T) {
	inputs := []string{
		"",
		"\n\n",
		"---\n---\n",
		"a: b\n",
		"a: 'it''s' # it's\n",
		"a: |\n  block\n  scalar\n",
		"a: >\n  folded\n",
		"a: [1, 2, 3]\n",
		"a: {b: c}\n",
		"list:\n- 30Gi\n- 30Gi\n",
		"deep:\n  a:\n    b:\n      c: 30Gi\n",
		"café: 30Gi # ☕ over here\n",
		"a: \"escaped \\\" quote\"\n",
		"a:\n",
		"{",
	}

	paths := [][]string{
		{"a"},
		{"a", "b"},
		{"list", "0"},
		{"deep", "a", "b", "c"},
		{"café"},
		{"missing"},
		{},
	}

	values := []string{"60Gi", "", "x y", "has # hash", "ok", "-dash", "a\nb"}

	for i, in := range inputs {
		for j, p := range paths {
			for k, v := range values {
				t.Run(fmt.Sprintf("input%d/path%d/value%d", i, j, k), func(t *testing.T) {
					out, err := Set(YAML(in), v, p...)
					if err == nil {
						_, jerr := ToJSON(out)
						assert.NoError(t, jerr, "input=%q path=%v value=%q produced invalid YAML: %q", in, p, v, string(out))
					}
				})
			}
		}
	}
}
