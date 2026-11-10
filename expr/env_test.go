package expr

import (
	"testing"

	"github.com/expr-lang/expr/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/SigNoz/baseplate.go/errors"
)

// tRename renames identifier from to identifier to, and counts the trees it
// has rewritten.
type tRename struct {
	from  string
	to    string
	trees *int
}

func (r tRename) Visit(node *ast.Node) {
	identifier, ok := (*node).(*ast.IdentifierNode)
	if !ok || identifier.Value != r.from {
		return
	}

	identifier.Value = r.to
	*r.trees++
}

func TestEnv_Patch(t *testing.T) {
	testCases := []struct {
		name          string
		env           func(trees *int) Env
		src           string
		vars          map[string]any
		expected      bool
		expectedTrees int
	}{
		{
			name: "Untyped_MemberRoot_Renamed",
			env: func(trees *int) Env {
				return Untyped().Patch(tRename{from: "body", to: "body_map", trees: trees})
			},
			src: `body.level = "error"`,
			vars: map[string]any{
				"body_map": map[string]any{"level": "error"},
			},
			expected:      true,
			expectedTrees: 1,
		},
		{
			name: "TypedDynamic_BeforeNilSafe_Renamed",
			env: func(trees *int) Env {
				return TypedDynamic().Patch(tRename{from: "body", to: "body_map", trees: trees})
			},
			src: `body.level = "error"`,
			vars: map[string]any{
				"body_map": map[string]any{"level": "error"},
			},
			expected:      true,
			expectedTrees: 2,
		},
		{
			name: "Untyped_CalleeRenamed",
			env: func(trees *int) Env {
				return Untyped().Patch(tRename{from: "env", to: "os_env_func", trees: trees})
			},
			src: `env("HOME") = "/home/alpha"`,
			vars: map[string]any{
				"os_env_func": func(name string) string { return "/home/" + "alpha" },
			},
			expected:      true,
			expectedTrees: 1,
		},
		{
			name: "TwoPatches_InOrder",
			env: func(trees *int) Env {
				return Untyped().
					Patch(tRename{from: "tier", to: "plan", trees: trees}).
					Patch(tRename{from: "plan", to: "level", trees: trees})
			},
			src: `tier = "gold"`,
			vars: map[string]any{
				"level": "gold",
			},
			expected:      true,
			expectedTrees: 2,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			trees := 0

			program, err := Compile(testCase.src, testCase.env(&trees))
			require.NoError(t, err)

			for range 3 {
				got, err := program.Bool(testCase.vars)
				require.NoError(t, err)
				assert.Equal(t, testCase.expected, got)
			}

			assert.Equal(t, testCase.expectedTrees, trees)
		})
	}
}

// A patch added to one copy of an Env isn't applied by another.
func TestEnv_Patch_Copies(t *testing.T) {
	trees := 0
	base := Untyped().Patch(tRename{from: "tier", to: "plan", trees: &trees})
	renamed := base.Patch(tRename{from: "plan", to: "level", trees: &trees})

	testCases := []struct {
		name     string
		env      Env
		vars     map[string]any
		expected bool
	}{
		{
			name:     "Base_OnePatch",
			env:      base,
			vars:     map[string]any{"plan": "gold"},
			expected: true,
		},
		{
			name:     "Renamed_TwoPatches",
			env:      renamed,
			vars:     map[string]any{"level": "gold"},
			expected: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			program, err := Compile(`tier = "gold"`, testCase.env)
			require.NoError(t, err)

			got, err := program.Bool(testCase.vars)
			require.NoError(t, err)
			assert.Equal(t, testCase.expected, got)
		})
	}
}

// A TypedDynamic program with a patch is typed against the names the
// patched tree reads, here plan, not only the names it was written with.
func TestEnv_Patch_TypedDynamicShapes(t *testing.T) {
	trees := 0

	program, err := Compile(`tier > 5`, TypedDynamic().Patch(tRename{from: "tier", to: "plan", trees: &trees}))
	require.NoError(t, err)

	testCases := []struct {
		name         string
		vars         map[string]any
		expected     bool
		expectedCode errors.Code
	}{
		{
			name:     "IntPlan_Matches",
			vars:     map[string]any{"plan": 7},
			expected: true,
		},
		{
			name:         "StringPlan_TypeInvalid",
			vars:         map[string]any{"plan": "7"},
			expectedCode: CodeTypeInvalid,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := program.Bool(testCase.vars)
			if testCase.expectedCode != (errors.Code{}) {
				require.Error(t, err)
				assert.Equal(t, testCase.expectedCode, errors.CodeOf(err))

				return
			}

			require.NoError(t, err)
			assert.Equal(t, testCase.expected, got)
		})
	}
}
