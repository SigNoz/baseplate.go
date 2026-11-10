package expr

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/expr-lang/expr/docgen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnv_Schema(t *testing.T) {
	type tTLSVersion struct {
		Version string `expr:"version"`
	}

	type tCheck struct {
		StatusCode int               `expr:"status_code"`
		Headers    map[string]string `expr:"headers"`
		JSON       any               `expr:"json"`
		Latency    time.Duration     `expr:"latency"`
		TLS        tTLSVersion       `expr:"tls"`
	}

	testCases := []struct {
		name           string
		env            Env
		expectedLimits Limits
		expectedKinds  map[docgen.Identifier]docgen.Kind
	}{
		{
			name: "Untyped_NoVars",
			env:  Untyped(),
		},
		{
			name: "TypedDynamic_NoVarsWithLimits",
			env:  TypedDynamic().LimitNodes(500),
			expectedLimits: Limits{
				Nodes: 500,
			},
		},
		{
			name: "TypedStatic_VarsAndLimits",
			env:  TypedStatic(tCheck{}).LimitNesting(1).LimitPattern(256).LimitMemory(1000),
			expectedLimits: Limits{
				Nesting: 1,
				Pattern: 256,
			},
			expectedKinds: map[docgen.Identifier]docgen.Kind{
				"status_code": "int",
				"headers":     "map",
				"json":        "any",
				"latency":     "struct",
				"tls":         "struct",
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			schema, err := testCase.env.Schema()
			require.NoError(t, err)
			assert.Equal(t, testCase.expectedLimits, schema.Limits)

			if testCase.expectedKinds == nil {
				assert.Nil(t, schema.Vars)

				return
			}

			require.NotNil(t, schema.Vars)

			for name, kind := range testCase.expectedKinds {
				require.Contains(t, schema.Vars.Variables, name)
				assert.Equal(t, kind, schema.Vars.Variables[name].Kind, name)
			}

			encoded, err := json.Marshal(schema)
			require.NoError(t, err)

			var decoded Schema
			require.NoError(t, json.Unmarshal(encoded, &decoded))
			assert.Equal(t, schema.Limits, decoded.Limits)
			assert.Len(t, decoded.Vars.Variables, len(schema.Vars.Variables))
		})
	}
}

func TestEnv_Schema_VarsInvalid(t *testing.T) {
	_, err := TypedStatic(nil).Schema()
	tAssertError(t, &tError{code: CodeVarsInvalid}, err)
}
