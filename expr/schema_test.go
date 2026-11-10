package expr

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnv_Schema(t *testing.T) {
	type tCheck struct {
		StatusCode int               `expr:"status_code"`
		Headers    map[string]string `expr:"headers"`
		JSON       any               `expr:"json"`
		Latency    time.Duration     `expr:"latency"`
		TLS        struct {
			Version string   `expr:"version"`
			SANs    []string `expr:"sans"`
		} `expr:"tls"`
	}

	testCases := []struct {
		name         string
		env          Env
		expectedJSON string
	}{
		{
			name:         "Untyped_Open",
			env:          Untyped(),
			expectedJSON: `{"open":true,"limits":{}}`,
		},
		{
			name:         "TypedDynamic_OpenWithLimits",
			env:          TypedDynamic().LimitNodes(500),
			expectedJSON: `{"open":true,"limits":{"nodes":500}}`,
		},
		{
			name: "TypedStatic_FieldsAndLimits",
			env:  TypedStatic(tCheck{}).LimitNesting(1).LimitPattern(256).LimitMemory(1000),
			expectedJSON: `{"vars":[` +
				`{"name":"status_code","type":"int"},` +
				`{"name":"headers","type":"map","elem":{"type":"string"}},` +
				`{"name":"json","type":"any"},` +
				`{"name":"latency","type":"duration"},` +
				`{"name":"tls","type":"object","fields":[{"name":"version","type":"string"},{"name":"sans","type":"list","elem":{"type":"string"}}]}` +
				`],"limits":{"nesting":1,"pattern":256}}`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			schema, err := testCase.env.Schema()
			require.NoError(t, err)

			encoded, err := json.Marshal(schema)
			require.NoError(t, err)
			assert.JSONEq(t, testCase.expectedJSON, string(encoded))

			var decoded Schema
			require.NoError(t, json.Unmarshal(encoded, &decoded))
			assert.Equal(t, schema, decoded)
		})
	}
}

func TestSchemaOf_VarsInvalid(t *testing.T) {
	testCases := []struct {
		name string
		vars any
	}{
		{
			name: "Int64Field",
			vars: struct{ Count int64 }{},
		},
		{
			name: "Nil",
			vars: nil,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := SchemaOf(testCase.vars)
			tAssertError(t, &tError{code: CodeVarsInvalid}, err)

			_, err = TypedStatic(testCase.vars).Schema()
			tAssertError(t, &tError{code: CodeVarsInvalid}, err)
		})
	}
}
