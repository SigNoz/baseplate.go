package expr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// The cases are SigNoz's ConvertLabelSetToEnv tests, plus cases for trimming
// and nesting. The expected variables and conflict flags were generated
// once by running ConvertLabelSetToEnv on every case. TestTyped runs Nest
// on every label set of SigNoz's EvalExpression tests as well.
func TestNest(t *testing.T) {
	testCases := []struct {
		name             string
		flat             map[string]string
		expectedVars     map[string]any
		expectedConflict bool
	}{
		{
			name:             "DeeperDottedKey_WinsOverShallow",
			flat:             map[string]string{"foo.bar": "shallow", "foo.bar.baz": "deep"},
			expectedVars:     map[string]any{"foo": map[string]any{"bar": map[string]any{"baz": "deep"}}},
			expectedConflict: true,
		},
		{
			name:         "DottedKeys_Nested",
			flat:         map[string]string{"foo.bar": "value1", "foo.baz": "value2"},
			expectedVars: map[string]any{"foo": map[string]any{"bar": "value1", "baz": "value2"}},
		},
		{
			name:             "NestedMap_WinsOverPlainKey",
			flat:             map[string]string{"foo": "ignored", "foo.bar": "value"},
			expectedVars:     map[string]any{"foo": map[string]any{"bar": "value"}},
			expectedConflict: true,
		},
		{
			name:         "PlainKeys_Kept",
			flat:         map[string]string{"key1": "value1", "key2": "value2"},
			expectedVars: map[string]any{"key1": "value1", "key2": "value2"},
		},
		{
			name:         "KeyWithSpaces_Trimmed",
			flat:         map[string]string{" env ": "prod"},
			expectedVars: map[string]any{"env": "prod"},
		},
		{
			name:         "SharedPrefix_Merged",
			flat:         map[string]string{"k8s.namespace": "alpha", "k8s.pod.name": "api-7f9c"},
			expectedVars: map[string]any{"k8s": map[string]any{"namespace": "alpha", "pod": map[string]any{"name": "api-7f9c"}}},
		},
		{
			name:         "DottedSegmentsWithSpaces_Trimmed",
			flat:         map[string]string{" a . b ": "x"},
			expectedVars: map[string]any{"a": map[string]any{"b": "x"}},
		},
		{
			name:         "DeepAndShallowSiblings_Merged",
			flat:         map[string]string{"a.b.c": "deep", "a.d": "side"},
			expectedVars: map[string]any{"a": map[string]any{"b": map[string]any{"c": "deep"}, "d": "side"}},
		},
		{
			name:         "Empty_EmptyVars",
			flat:         map[string]string{},
			expectedVars: map[string]any{},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			vars, conflict := Nest(testCase.flat)
			assert.Equal(t, testCase.expectedVars, vars)
			assert.Equal(t, testCase.expectedConflict, conflict)
		})
	}
}

func TestNest_NamedStringTypes(t *testing.T) {
	type tKey string

	type tValue string

	testCases := []struct {
		name         string
		flat         map[tKey]tValue
		expectedVars map[string]any
	}{
		{
			name:         "DottedKey_Nested",
			flat:         map[tKey]tValue{"service.name": "checkout"},
			expectedVars: map[string]any{"service": map[string]any{"name": "checkout"}},
		},
		{
			name:         "PlainKey_Kept",
			flat:         map[tKey]tValue{"tier": "gold"},
			expectedVars: map[string]any{"tier": "gold"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			vars, conflict := Nest(testCase.flat)
			assert.Equal(t, testCase.expectedVars, vars)
			assert.False(t, conflict)
		})
	}
}
