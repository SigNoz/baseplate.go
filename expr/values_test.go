package expr

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProgram_References(t *testing.T) {
	testCases := []struct {
		name     string
		src      string
		env      Env
		expected []string
	}{
		{
			name:     "NestedFields_LongestPath",
			src:      `tls.certificate.remaining > duration("336h") AND tls.used`,
			env:      tHTTPEnv(),
			expected: []string{"tls.certificate.remaining", "tls.used"},
		},
		{
			name:     "BracketKey_Quoted",
			src:      `headers["content-type"] startsWith "application/json"`,
			env:      tHTTPEnv(),
			expected: []string{`headers["content-type"]`},
		},
		{
			name:     "IndexedList_StopsAtIndex",
			src:      `json.items[0].ok`,
			env:      tHTTPEnv(),
			expected: []string{"json.items"},
		},
		{
			name:     "FunctionsAndLets_LeftOut",
			src:      `let limit = 300; len(body) < limit`,
			env:      tHTTPEnv(),
			expected: []string{"body"},
		},
		{
			name: "Literal_NoReferences",
			src:  `1 < 2`,
			env:  tHTTPEnv(),
		},
		{
			name:     "Untyped_Attributes",
			src:      `attributes["k8s.pod.name"] = "api" OR severity_text = "ERROR"`,
			env:      Untyped(),
			expected: []string{`attributes["k8s.pod.name"]`, "severity_text"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			program, err := Compile(testCase.src, testCase.env)
			require.NoError(t, err)
			assert.Equal(t, testCase.expected, program.References())
		})
	}
}

// The values are those a test run shows next to a failed assertion
// (Assertions TRD §6).
func TestProgram_Values(t *testing.T) {
	vars := tHTTP{
		StatusCode: 503,
		Headers: map[string]string{
			"content-type": "text/html",
		},
		JSON: map[string]any{
			"status": "degraded",
			"items":  []any{1, 2},
		},
		ResponseTime: 1500 * time.Millisecond,
		TLS: tTLS{
			Certificate: tCertificate{
				SANs: []string{"a.example.com"},
			},
		},
	}

	testCases := []struct {
		name     string
		src      string
		expected map[string]string
	}{
		{
			name: "Scalars_Formatted",
			src:  `status_code = 200 AND response_time < duration("1s")`,
			expected: map[string]string{
				"status_code":   "503",
				"response_time": "1.5s",
			},
		},
		{
			name: "JSONField_AsText",
			src:  `json.status = "ok"`,
			expected: map[string]string{
				"json.status": "degraded",
			},
		},
		{
			name: "MissingField_Nil",
			src:  `json.missing = "x" OR headers["server"] = "nginx"`,
			expected: map[string]string{
				"json.missing":   "nil",
				"headers.server": "nil",
			},
		},
		{
			name: "List_JSON",
			src:  `len(json.items) > 3 AND len(tls.certificate.sans) = 0`,
			expected: map[string]string{
				"json.items":           `[1,2]`,
				"tls.certificate.sans": `["a.example.com"]`,
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			program, err := Compile(testCase.src, tHTTPEnv())
			require.NoError(t, err)
			assert.Equal(t, testCase.expected, program.Values(vars))
		})
	}
}

func TestStringify_Truncates(t *testing.T) {
	testCases := []struct {
		name           string
		value          any
		expectedLength int
	}{
		{
			name:           "Short_Kept",
			value:          "degraded",
			expectedLength: 8,
		},
		{
			name:           "Long_CutTo256",
			value:          string(make([]rune, 300)),
			expectedLength: 256,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Len(t, []rune(stringify(testCase.value)), testCase.expectedLength)
		})
	}
}
