package kyaml

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	kyyaml "sigs.k8s.io/kustomize/kyaml/yaml"

	"github.com/SigNoz/baseplate.go/errors"
)

// tBlockInner returns the decoded content of the block scalar at path, so a
// test can inspect the embedded document on its own.
func tBlockInner(t *testing.T, y string, path ...string) string {
	t.Helper()

	rn, err := kyyaml.Parse(y)
	require.NoError(t, err)

	node, err := rn.Pipe(kyyaml.Lookup(path...))
	require.NoError(t, err)
	require.NotNil(t, node)

	return node.YNode().Value
}

func TestEditBlockScalar(t *testing.T) {
	testCases := []struct {
		name            string
		input           string
		outerPath       []string
		edit            func(inner YAML) (YAML, error)
		expectedChanged []string // Lines of the outer document expected to change.
		expectedKept    []string // Substrings that must survive unchanged.
	}{
		{
			name:      "CollectorFields_OnlyTargetedFieldsChange",
			input:     "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: signoz-otel-collector\ndata:\n  otel-collector-config.yaml: |-\n    processors:\n      batch:\n        send_batch_max_size: 25000\n        send_batch_size: 20000\n        timeout: 1s\n      batch/meter:\n        send_batch_max_size: 25000\n        send_batch_size: 20000\n        timeout: 1s\n      memory_limiter:\n        check_interval: 1s\n        limit_mib: 512\n        spike_limit_mib: 128\n  otel-collector-opamp-config.yaml: keep\n",
			outerPath: []string{"data", "otel-collector-config.yaml"},
			edit: func(inner YAML) (YAML, error) {
				var err error

				inner, err = Set(inner, "55000", "processors", "batch", "send_batch_max_size")
				if err != nil {
					return nil, err
				}

				inner, err = Set(inner, "50000", "processors", "batch", "send_batch_size")
				if err != nil {
					return nil, err
				}

				return Set(inner, "8192", "processors", "memory_limiter", "limit_mib")
			},
			expectedChanged: []string{
				"        send_batch_max_size: 55000",
				"        send_batch_size: 50000",
				"        limit_mib: 8192",
			},
			// The batch/meter processor shares the old values and must not move;
			// the outer keys and the sibling data key must survive.
			expectedKept: []string{
				"      batch/meter:\n        send_batch_max_size: 25000\n        send_batch_size: 20000",
				"kind: ConfigMap",
				"  name: signoz-otel-collector",
				"  otel-collector-opamp-config.yaml: keep",
				"otel-collector-config.yaml: |-",
			},
		},
		{
			name:      "LiteralBlockAtTopLevelKey_IndentPreserved",
			input:     "config: |\n  processors:\n    memory_limiter:\n      limit_mib: 512\ntrailing: keep\n",
			outerPath: []string{"config"},
			edit: func(inner YAML) (YAML, error) {
				return Set(inner, "2048", "processors", "memory_limiter", "limit_mib")
			},
			expectedChanged: []string{"      limit_mib: 2048"},
			expectedKept:    []string{"config: |", "trailing: keep"},
		},
		{
			name:      "BlockScalarNestedUnderSequence",
			input:     "spec:\n  containers:\n  - name: collector\n    config: |-\n      processors:\n        memory_limiter:\n          limit_mib: 512\n  - name: sidecar\n    config: keep\n",
			outerPath: []string{"spec", "containers", "[name=collector]", "config"},
			edit: func(inner YAML) (YAML, error) {
				return Set(inner, "2048", "processors", "memory_limiter", "limit_mib")
			},
			expectedChanged: []string{"          limit_mib: 2048"},
			expectedKept:    []string{"- name: sidecar\n    config: keep"},
		},
		{
			// The block header is on the sequence-dash line, so the body
			// boundary must come from the content indentation, not the dash
			// line's leading spaces, or a less-indented sibling leaks in.
			name:      "BlockHeaderOnDashLine_SiblingExcluded",
			input:     "items:\n- config: |-\n    a: b\n  other: keep\n",
			outerPath: []string{"items", "0", "config"},
			edit: func(inner YAML) (YAML, error) {
				return Set(inner, "updated", "a")
			},
			expectedChanged: []string{"    a: updated"},
			expectedKept:    []string{"  other: keep"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			out, err := EditBlockScalar(YAML(testCase.input), testCase.outerPath, testCase.edit)
			require.NoError(t, err)

			changed := tChangedLines(t, testCase.input, string(out))
			assert.ElementsMatch(t, testCase.expectedChanged, changed)

			for _, kept := range testCase.expectedKept {
				assert.Contains(t, string(out), kept)
			}
		})
	}
}

// TestEditBlockScalar_ChangesExactlyIntendedLeaves proves the embedded
// document changes in exactly the targeted leaves and nothing else — in
// particular the second batch processor that shares the old values is
// untouched — and that the outer document changes only in the one
// block-scalar leaf.
func TestEditBlockScalar_ChangesExactlyIntendedLeaves(t *testing.T) {
	input := "apiVersion: v1\nkind: ConfigMap\ndata:\n  config.yaml: |-\n    processors:\n      batch:\n        send_batch_max_size: 25000\n        send_batch_size: 20000\n      batch/meter:\n        send_batch_max_size: 25000\n        send_batch_size: 20000\n      memory_limiter:\n        limit_mib: 512\n  other: untouched\n"
	outerPath := []string{"data", "config.yaml"}

	innerBefore := tLeaves(t, tBlockInner(t, input, outerPath...))
	outerBefore := tLeaves(t, input)

	out, err := EditBlockScalar(YAML(input), outerPath, func(inner YAML) (YAML, error) {
		var err error

		inner, err = Set(inner, "55000", "processors", "batch", "send_batch_max_size")
		if err != nil {
			return nil, err
		}

		inner, err = Set(inner, "50000", "processors", "batch", "send_batch_size")
		if err != nil {
			return nil, err
		}

		return Set(inner, "8192", "processors", "memory_limiter", "limit_mib")
	})
	require.NoError(t, err)

	innerAfter := tLeaves(t, tBlockInner(t, string(out), outerPath...))
	innerChanged := tDiffKeys(innerBefore, innerAfter)
	assert.ElementsMatch(t, []string{
		"/processors/batch/send_batch_max_size",
		"/processors/batch/send_batch_size",
		"/processors/memory_limiter/limit_mib",
	}, innerChanged, "only the three targeted embedded fields may change")

	assert.Equal(t, "55000", innerAfter["/processors/batch/send_batch_max_size"])
	assert.Equal(t, "8192", innerAfter["/processors/memory_limiter/limit_mib"])
	assert.Equal(t, "25000", innerAfter["/processors/batch/meter/send_batch_max_size"], "batch/meter must be untouched")

	outerAfter := tLeaves(t, string(out))
	assert.Equal(t, []string{"/data/config.yaml"}, tDiffKeys(outerBefore, outerAfter), "only the block-scalar leaf may change in the outer document")
}

func TestEditBlockScalar_Errors(t *testing.T) {
	setLimit := func(inner YAML) (YAML, error) {
		return Set(inner, "2048", "processors", "memory_limiter", "limit_mib")
	}

	testCases := []struct {
		name         string
		input        string
		outerPath    []string
		edit         func(inner YAML) (YAML, error)
		expectedCode errors.Code
	}{
		{name: "PlainScalarTarget_NotBlockScalar", input: "data:\n  config: plain\n", outerPath: []string{"data", "config"}, edit: setLimit, expectedCode: CodeNotBlockScalar},
		{name: "MappingTarget_NotBlockScalar", input: "data:\n  config:\n    a: b\n", outerPath: []string{"data", "config"}, edit: setLimit, expectedCode: CodeNotBlockScalar},
		{name: "FoldedScalarTarget_NotBlockScalar", input: "data:\n  config: >-\n    message: hello\n    world\n", outerPath: []string{"data", "config"}, edit: setLimit, expectedCode: CodeNotBlockScalar},
		{name: "UnknownPath_PathNotFound", input: "data:\n  config: |\n    a: b\n", outerPath: []string{"data", "missing"}, edit: setLimit, expectedCode: CodePathNotFound},
		{name: "MultipleDocuments", input: "data:\n  config: |\n    a: b\n---\nkind: X\n", outerPath: []string{"data", "config"}, edit: setLimit, expectedCode: CodeMultipleDocuments},
		{name: "EmptyInput_YAMLEmpty", input: "", outerPath: []string{"data", "config"}, edit: setLimit, expectedCode: CodeYAMLEmpty},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := EditBlockScalar(YAML(testCase.input), testCase.outerPath, testCase.edit)
			require.Error(t, err)
			assert.Equal(t, errors.TypeInvalidInput, errors.TypeOf(err))
			assert.Equal(t, testCase.expectedCode, errors.CodeOf(err))
		})
	}
}

// TestEditBlockScalar_PropagatesEditError confirms an error from the
// callback is returned verbatim, not wrapped or swallowed.
func TestEditBlockScalar_PropagatesEditError(t *testing.T) {
	sentinel := errors.New(errors.TypeInvalidInput, CodeUnsafeValue, "boom")

	_, err := EditBlockScalar(YAML("data:\n  config: |\n    a: b\n"), []string{"data", "config"}, func(_ YAML) (YAML, error) {
		return nil, sentinel
	})
	require.ErrorIs(t, err, sentinel)
}

// TestEditBlockScalar_DoesNotWrapLongLines guards against the failure mode
// of a JSON round-trip emitter (SigNoz/platform-pod#3363): re-serializing
// the embedded collector config hard-wrapped a ~90-column OTTL expression
// at 80 columns, splitting it across two physical lines. Because
// EditBlockScalar byte-rewrites only the changed scalar and restores every
// other line from its original bytes, a long untouched line survives
// verbatim.
func TestEditBlockScalar_DoesNotWrapLongLines(t *testing.T) {
	longOTTL := `          - time_unix_nano == 0 and observed_time_unix_nano != 0 and observed_time < Now() - Duration("360h")`
	input := "apiVersion: v1\nkind: ConfigMap\ndata:\n  otel-collector-config.yaml: |-\n    processors:\n      filter/drop_stale:\n        logs:\n          log_record:\n" + longOTTL + "\n      memory_limiter:\n        limit_mib: 512\n"

	out, err := EditBlockScalar(YAML(input), []string{"data", "otel-collector-config.yaml"}, func(inner YAML) (YAML, error) {
		return Set(inner, "8192", "processors", "memory_limiter", "limit_mib")
	})
	require.NoError(t, err)

	changed := tChangedLines(t, input, string(out))
	assert.Equal(t, []string{"        limit_mib: 8192"}, changed, "only limit_mib may change")
	assert.Contains(t, string(out), longOTTL, "the long OTTL expression must survive on one line")
	assert.NotContains(t, string(out), "Now()\n", "the OTTL expression must not be wrapped after Now()")
}

// TestEditBlockScalar_PreservesUntouchedBytes checks that a line the
// callback does not change keeps its exact source bytes, including a
// whitespace-only interior line whose indentation differs from the block's,
// so an identity edit round-trips the document unchanged.
func TestEditBlockScalar_PreservesUntouchedBytes(t *testing.T) {
	// The block body is indented 4; the interior line has 6 spaces and no
	// content, and a trailing-comment line carries bytes dedent/reindent could
	// drop.
	input := "data:\n  config: |\n    a: 1\n      \n    b: 2 # note\ntrailing: keep\n"
	outerPath := []string{"data", "config"}

	identity, err := EditBlockScalar(YAML(input), outerPath, func(inner YAML) (YAML, error) {
		return inner, nil
	})
	require.NoError(t, err)
	assert.Equal(t, input, string(identity), "an identity edit must round-trip byte for byte")

	changed, err := EditBlockScalar(YAML(input), outerPath, func(inner YAML) (YAML, error) {
		return Set(inner, "3", "b")
	})
	require.NoError(t, err)

	got := tChangedLines(t, input, string(changed))
	assert.Equal(t, []string{"    b: 3 # note"}, got, "only the targeted line may change; the comment and blank line survive")
	assert.Contains(t, string(changed), "    a: 1\n      \n", "the whitespace-only interior line keeps its spaces")
}

// TestEditBlockScalar_NeverPanics feeds hostile inputs, paths, and
// callbacks and only requires that EditBlockScalar returns without
// panicking and that any success is still valid YAML.
func TestEditBlockScalar_NeverPanics(t *testing.T) {
	inputs := []string{
		"",
		"data:\n  config: |\n    a: b\n",
		"data:\n  config: |-\n    a: b\n    c: d\n",
		"data:\n  config: |\n\n    a: b\n\n    c: d\ntrailing: x\n",
		"data:\n  config: plain\n",
		"data:\n  config: |\n",
		"config: |\n  deep:\n    nested:\n      v: 1\n",
		"{",
	}

	paths := [][]string{
		{"data", "config"},
		{"config"},
		{"missing"},
		{},
	}

	edits := []func(YAML) (YAML, error){
		func(inner YAML) (YAML, error) { return Set(inner, "60", "a") },
		func(inner YAML) (YAML, error) { return inner, nil },
		func(_ YAML) (YAML, error) { return YAML("broken: ["), nil },
		func(_ YAML) (YAML, error) { return nil, errors.New(errors.TypeInvalidInput, CodeYAMLInvalid, "x") },
	}

	for i, in := range inputs {
		for j, p := range paths {
			for k, e := range edits {
				t.Run(fmt.Sprintf("input%d/path%d/edit%d", i, j, k), func(t *testing.T) {
					out, err := EditBlockScalar(YAML(in), p, e)
					if err == nil {
						_, jerr := ToJSON(out)
						assert.NoError(t, jerr, "input=%q path=%v produced invalid YAML: %q", in, p, string(out))
					}
				})
			}
		}
	}
}
