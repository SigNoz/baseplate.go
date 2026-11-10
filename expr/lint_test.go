package expr

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/SigNoz/baseplate.go/errors"
)

func TestLint(t *testing.T) {
	testCases := []struct {
		name         string
		env          Env
		src          string
		expectedCode errors.Code
	}{
		{
			name: "Nesting_OneDeep_Accepted",
			env:  tHTTPEnv(),
			src:  `all(json.items, .ok)`,
		},
		{
			name:         "Nesting_TwoDeep_NestingTooDeep",
			env:          tHTTPEnv(),
			src:          `all(json.items, any(.tags, # = "x"))`,
			expectedCode: CodeNestingTooDeep,
		},
		{
			name:         "Nesting_InTernary_NestingTooDeep",
			env:          tHTTPEnv(),
			src:          `tls.used ? all(json.items, all(.tags, # != "")) : false`,
			expectedCode: CodeNestingTooDeep,
		},
		{
			name: "Nesting_Unset_Accepted",
			env:  TypedStatic(tHTTP{}),
			src:  `all(json.items, any(.tags, # = "x"))`,
		},
		{
			name:         "Pattern_NotLiteral_PatternInvalid",
			env:          tHTTPEnv(),
			src:          `body REGEXP http_version`,
			expectedCode: CodePatternInvalid,
		},
		{
			name:         "Pattern_Invalid_PatternInvalid",
			env:          tHTTPEnv(),
			src:          `body REGEXP "[a"`,
			expectedCode: CodePatternInvalid,
		},
		{
			name:         "Pattern_TooLong_PatternInvalid",
			env:          tHTTPEnv(),
			src:          `body REGEXP "` + strings.Repeat("a", 257) + `"`,
			expectedCode: CodePatternInvalid,
		},
		{
			name: "Pattern_AtLimit_Accepted",
			env:  tHTTPEnv(),
			src:  `body REGEXP "` + strings.Repeat("a", 256) + `"`,
		},
		{
			name: "Pattern_Unset_NotLiteralAccepted",
			env:  TypedStatic(tHTTP{}),
			src:  `body REGEXP http_version`,
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
