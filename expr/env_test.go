package expr

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/SigNoz/baseplate.go/errors"
	"github.com/SigNoz/baseplate.go/expr/ast"
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

type tCertificate struct {
	SubjectCN string        `expr:"subject_cn"`
	SANs      []string      `expr:"sans"`
	Remaining time.Duration `expr:"remaining"`
	NotAfter  time.Time     `expr:"not_after"`
}

type tTLS struct {
	Used        bool         `expr:"used"`
	Version     string       `expr:"version"`
	Certificate tCertificate `expr:"certificate"`
}

type tPhases struct {
	Wait time.Duration `expr:"wait"`
}

type tHTTP struct {
	StatusCode   int               `expr:"status_code"`
	HTTPVersion  string            `expr:"http_version"`
	Headers      map[string]string `expr:"headers"`
	Body         string            `expr:"body"`
	BodySize     int               `expr:"body_size"`
	JSON         any               `expr:"json"`
	Redirects    int               `expr:"redirects"`
	ResponseTime time.Duration     `expr:"response_time"`
	Phases       tPhases           `expr:"phases"`
	TLS          tTLS              `expr:"tls"`
}

func tHTTPEnv() Env {
	return TypedStatic(tHTTP{}).LimitNesting(1).LimitPattern(256)
}

func TestTypedStatic_Assertions(t *testing.T) {
	testCases := []struct {
		name string
		src  string
	}{
		{name: "StatusInSet", src: `status_code IN [200, 204]`},
		{name: "HTTPVersionInSet", src: `http_version IN ["1.1", "2"]`},
		{name: "MustBeTLS", src: `tls.used`},
		{name: "MustNotBeTLS", src: `NOT tls.used`},
		{name: "BodyMustNotMatch", src: `NOT (body REGEXP "maintenance")`},
		{name: "BodyContains", src: `body CONTAINS "ok"`},
		{name: "BodyDoesNotContain", src: `NOT (body CONTAINS "error")`},
		{name: "BodyMustMatch", src: `body REGEXP "version: [0-9]+"`},
		{name: "JSONFieldEquals", src: `json.data.state = "ready"`},
		{name: "EveryItemHealthy", src: `all(json.items, .ok)`},
		{name: "SomeItemHasField", src: `any(json.items, .id = "x")`},
		{name: "HeaderMustNotMatch_MissingAllowed", src: `NOT (headers["server"] REGEXP "nginx/1\\.1")`},
		{name: "HeaderMustNotMatch_MissingFails", src: `"server" IN headers AND NOT (headers["server"] REGEXP "nginx/1\\.1")`},
		{name: "HeaderMustMatch", src: `headers["content-type"] startsWith "application/json"`},
		{name: "BodySizeRange", src: `body_size >= 10 AND body_size <= 5000`},
		{name: "Latency", src: `response_time < duration("800ms")`},
		{name: "PhaseLatency", src: `phases.wait < duration("300ms")`},
		{name: "CertificateExpiry", src: `tls.certificate.remaining > duration("336h")`},
		{name: "TLSVersion", src: `tls.version = "1.3"`},
		{name: "Redirects", src: `redirects <= 1`},
		{name: "ContentLengthFromHeader", src: `int(headers["content-length"]) > 0`},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := Compile(testCase.src, tHTTPEnv())
			assert.NoError(t, err)
		})
	}
}

func TestTypedStatic(t *testing.T) {
	vars := tHTTP{
		StatusCode:  200,
		HTTPVersion: "2",
		Headers: map[string]string{
			"content-type": "application/json",
		},
		Body: "ok version: 3",
		JSON: map[string]any{
			"state": "ok",
			"items": []any{
				map[string]any{"ok": true},
			},
		},
		ResponseTime: 120 * time.Millisecond,
		TLS: tTLS{
			Used:    true,
			Version: "1.3",
			Certificate: tCertificate{
				Remaining: 400 * time.Hour,
			},
		},
	}

	testCases := []struct {
		name          string
		src           string
		expected      bool
		expectedError *tError
	}{
		{
			name:     "DeclaredFields_Match",
			src:      `status_code = 200 AND tls.version = "1.3"`,
			expected: true,
		},
		{
			name: "StatusAsString_TypeInvalid",
			src:  `status_code = "200"`,
			expectedError: &tError{
				code:    CodeTypeInvalid,
				message: "invalid operation: = (mismatched types int and string)",
				from:    12,
				to:      13,
				line:    1,
				column:  13,
			},
		},
		{
			name: "MisspelledVar_NameUnknown",
			src:  `stauts_code = 200`,
			expectedError: &tError{
				code:        CodeNameUnknown,
				message:     "unknown var stauts_code; did you mean status_code?",
				suggestions: []string{"status_code"},
				from:        0,
				to:          11,
				line:        1,
				column:      1,
			},
		},
		{
			name: "MisspelledField_NameUnknown",
			src:  `tls.versoin = "1.3"`,
			expectedError: &tError{
				code:        CodeNameUnknown,
				message:     "unknown field tls.versoin; did you mean tls.version?",
				suggestions: []string{"tls.version"},
				from:        4,
				to:          11,
				line:        1,
				column:      5,
			},
		},
		{
			name: "UnknownVarNoCloseName_NameUnknown",
			src:  `latency < duration("1s")`,
			expectedError: &tError{
				code:    CodeNameUnknown,
				message: "unknown var latency",
				from:    0,
				to:      7,
				line:    1,
				column:  1,
			},
		},
		{
			name: "MisspelledFunction_FunctionUnknown",
			src:  `uppr(body) = "OK"`,
			expectedError: &tError{
				code:        CodeFunctionUnknown,
				message:     "unknown function uppr; did you mean upper?",
				suggestions: []string{"upper"},
				from:        0,
				to:          4,
				line:        1,
				column:      1,
			},
		},
		{
			name:     "LetName_NotAField",
			src:      `let limit = 300; body_size < limit`,
			expected: true,
		},
		{
			name: "StructAgainstString_TypeInvalid",
			src:  `tls = "1.3"`,
			expectedError: &tError{
				code:    CodeTypeInvalid,
				message: "invalid operation: = (mismatched types expr.tTLS and string)",
				from:    4,
				to:      5,
				line:    1,
				column:  5,
			},
		},
		{
			name: "DurationAgainstInt_TypeInvalid",
			src:  `response_time < 500`,
			expectedError: &tError{
				code:    CodeTypeInvalid,
				message: "invalid operation: < (mismatched types time.Duration and int)",
				from:    14,
				to:      15,
				line:    1,
				column:  15,
			},
		},
		{
			name:     "DurationAgainstDuration_Matches",
			src:      `response_time < duration("500ms")`,
			expected: true,
		},
		{
			name:     "JSONField_Matches",
			src:      `json.state = "ok"`,
			expected: true,
		},
		{
			name:     "MissingJSONField_IsNil",
			src:      `json.missing = nil`,
			expected: true,
		},
		{
			name:     "MissingJSONParent_NotEqual_Matches",
			src:      `json.a.b != "x"`,
			expected: true,
		},
		{
			name:     "MissingHeader_IsNil",
			src:      `headers["server"] = nil`,
			expected: true,
		},
		{
			name: "MissingHeader_NotIn",
			src:  `"server" IN headers`,
		},
		{
			name: "MissingHeader_RegexpFalse",
			src:  `headers["server"] REGEXP "nginx"`,
		},
		{
			name: "MissingJSONField_ContainsFalse",
			src:  `json.missing CONTAINS "a"`,
		},
		{
			name: "IndexPastEnd_RunFailed",
			src:  `json.items[5].ok`,
			expectedError: &tError{
				code:    CodeRunFailed,
				message: "index out of range: 5 (array length is 1)",
				from:    10,
				to:      11,
				line:    1,
				column:  11,
			},
		},
		{
			name: "MissingJSONField_ResultNotBool",
			src:  `json.missing`,
			expectedError: &tError{
				code: CodeResultNotBool,
			},
		},
		{
			name:     "NestedStructField_Matches",
			src:      `tls.certificate.remaining > duration("336h")`,
			expected: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			program, err := Compile(testCase.src, tHTTPEnv())
			if err == nil {
				var got bool

				got, err = program.Bool(vars)
				if err == nil {
					require.Nil(t, testCase.expectedError)
					assert.Equal(t, testCase.expected, got)

					return
				}
			}

			require.NotNil(t, testCase.expectedError, "unexpected error: %v", err)
			tAssertError(t, testCase.expectedError, err)
		})
	}
}

func TestTypedStatic_RunVariables(t *testing.T) {
	program, err := Compile(`status_code = 200`, tHTTPEnv())
	require.NoError(t, err)

	testCases := []struct {
		name          string
		vars          any
		expected      bool
		expectedError *tError
	}{
		{
			name:     "DeclaredType_Matches",
			vars:     tHTTP{StatusCode: 200},
			expected: true,
		},
		{
			name: "OtherType_RunFailed",
			vars: map[string]any{"status_code": 200},
			expectedError: &tError{
				code:    CodeRunFailed,
				message: "vars are map[string]interface {}, expected expr.tHTTP",
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := program.Bool(testCase.vars)
			if testCase.expectedError != nil {
				tAssertError(t, testCase.expectedError, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, testCase.expected, got)
		})
	}
}

func TestEnv_LimitNodes(t *testing.T) {
	testCases := []struct {
		name         string
		env          Env
		src          string
		expectedCode errors.Code
	}{
		{
			name:         "Nodes_OverLimit_NodesExceeded",
			env:          tHTTPEnv().LimitNodes(2),
			src:          `status_code > 0 AND redirects > 0`,
			expectedCode: CodeNodesExceeded,
		},
		{
			name:         "Nodes_OverLimit_UntypedExpressionTooLarge",
			env:          Untyped().LimitNodes(2),
			src:          `a > 0 AND b > 0`,
			expectedCode: CodeNodesExceeded,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := Compile(testCase.src, testCase.env)
			if testCase.expectedCode == (errors.Code{}) {
				require.NoError(t, err)

				return
			}

			require.Error(t, err)
			assert.Equal(t, testCase.expectedCode, errors.CodeOf(err))
		})
	}
}

func TestEnv_LimitMemory(t *testing.T) {
	testCases := []struct {
		name         string
		env          Env
		expectedCode errors.Code
	}{
		{
			name: "Unset_DefaultBudget",
			env:  TypedStatic(tHTTP{}),
		},
		{
			name:         "Small_MemoryExceeded",
			env:          TypedStatic(tHTTP{}).LimitMemory(100),
			expectedCode: CodeMemoryExceeded,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			program, err := Compile(`len(map(1..1000, # * 2)) > 0`, testCase.env)
			require.NoError(t, err)

			_, err = program.Bool(tHTTP{})
			if testCase.expectedCode == (errors.Code{}) {
				require.NoError(t, err)

				return
			}

			require.Error(t, err)
			assert.Equal(t, testCase.expectedCode, errors.CodeOf(err))
		})
	}
}
