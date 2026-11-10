package expr

import (
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/SigNoz/baseplate.go/errors"
)

// The cases are SigNoz's EvalExpression tests, plus cases where the result
// depends on which labels are present. The expected results and errors were
// generated once by running SigNoz's ConvertLabelSetToEnv and EvalExpression
// on the SigNoz/expr fork (v1.17.7-beta) for every case. A program that
// Compile rejects must be one the reference rejects on every label set.
func TestTypedDynamic(t *testing.T) {
	testCases := []struct {
		name          string
		expression    string
		labels        map[string]string
		expected      bool
		expectedError *tError
	}{
		{
			name:       "And_BothMatch",
			expression: `env = "production" AND service = "api"`,
			labels:     map[string]string{"env": "production", "service": "api"},
			expected:   true,
		},
		{
			name:       "And_BothMatch_ExprSyntax",
			expression: `service == "auth" && env == "production"`,
			labels:     map[string]string{"env": "production", "service": "auth"},
			expected:   true,
		},
		{
			name:       "And_MissingDottedLabel_NoMatch",
			expression: "service.name = 'otel-demo' AND http.status_code = '501'",
			labels:     map[string]string{"http.status_code": "501"},
		},
		{
			name:       "And_MissingKey_NoMatch",
			expression: "threshold.name = 'warning' && nonexistent = 'auth'",
			labels:     map[string]string{"threshold.name": "warning"},
		},
		{
			name:       "And_OneMismatch_ExprSyntax",
			expression: `service == "auth" && env == "staging"`,
			labels:     map[string]string{"env": "production", "service": "auth"},
		},
		{
			name:       "And_PartialMatch",
			expression: `env = "production" AND service = "api"`,
			labels:     map[string]string{"env": "production", "service": "worker"},
		},
		{
			name:       "BracketAccessOnDottedLabel_Match",
			expression: `cloud["provider"] = 'aws'`,
			labels:     map[string]string{"cloud.provider": "aws"},
			expected:   true,
		},
		{
			name:       "CaseSensitive_NoMatch_ExprSyntax",
			expression: `service == "Auth"`,
			labels:     map[string]string{"service": "auth"},
		},
		{
			name:       "CaseSensitive_NoMatch_QBSyntax",
			expression: `service = "Auth"`,
			labels:     map[string]string{"service": "auth"},
		},
		{
			name:       "Contains_NoSubstringMatch_Lowercase",
			expression: `host contains "staging"`,
			labels:     map[string]string{"host": "prod-server-01"},
		},
		{
			name:       "Contains_NoSubstringMatch_Uppercase",
			expression: `host CONTAINS "staging"`,
			labels:     map[string]string{"host": "prod-server-01"},
		},
		{
			name:       "Contains_SubstringMatch_Lowercase",
			expression: `host contains "prod"`,
			labels:     map[string]string{"host": "prod-server-01"},
			expected:   true,
		},
		{
			name:       "Contains_SubstringMatch_Uppercase",
			expression: `host CONTAINS "prod"`,
			labels:     map[string]string{"host": "prod-server-01"},
			expected:   true,
		},
		{
			name:       "Equality_Match",
			expression: `env = "production"`,
			labels:     map[string]string{"env": "production"},
			expected:   true,
		},
		{
			name:       "Equality_Match_ExprSyntax",
			expression: "threshold.name == 'auth' && ruleId == 'rule1'",
			labels:     map[string]string{"ruleId": "rule1", "threshold.name": "auth"},
			expected:   true,
		},
		{
			name:       "Equality_Match_QBSyntax",
			expression: "threshold.name = 'auth' AND ruleId = 'rule1'",
			labels:     map[string]string{"ruleId": "rule1", "threshold.name": "auth"},
			expected:   true,
		},
		{
			name:       "Equality_NoMatch",
			expression: `env = "production"`,
			labels:     map[string]string{"env": "staging"},
		},
		{
			name:       "Equality_NoMatch_ExprSyntax",
			expression: `service == "payment"`,
			labels:     map[string]string{"env": "production", "service": "auth"},
		},
		{
			name:       "InList_Absent",
			expression: `env in ["production", "staging"]`,
			labels:     map[string]string{"env": "development"},
		},
		{
			name:       "InList_Present",
			expression: `env in ["production", "staging"]`,
			labels:     map[string]string{"env": "production"},
			expected:   true,
		},
		{
			name:       "In_ValueInList_Uppercase",
			expression: `service IN ["auth", "payment", "notification"]`,
			labels:     map[string]string{"service": "auth"},
			expected:   true,
		},
		{
			name:       "In_ValueNotInList_Uppercase",
			expression: `service IN ["payment", "notification"]`,
			labels:     map[string]string{"service": "auth"},
		},
		{
			name:       "Inequality_Match",
			expression: `env != "production"`,
			labels:     map[string]string{"env": "staging"},
			expected:   true,
		},
		{
			name:       "InvalidExpression_ReturnsError",
			expression: "env =",
			labels:     map[string]string{"env": "production"},
			expectedError: &tError{
				code:    CodeSyntaxInvalid,
				message: "unexpected token EOF",
				from:    4,
				to:      5,
				line:    1,
				column:  5,
			},
		},
		{
			name:       "KubernetesNamespace_Match_ExprSyntax",
			expression: `k8s.namespace.name == "auth" && service in ["auth", "payment"]`,
			labels:     map[string]string{"host": "k8s-node-1", "k8s.namespace.name": "auth", "service": "auth"},
			expected:   true,
		},
		{
			name:       "KubernetesNamespace_Match_MixedSyntax",
			expression: `k8s.namespace.name = "auth" && service IN ["auth", "payment"]`,
			labels:     map[string]string{"host": "k8s-node-1", "k8s.namespace.name": "auth", "service": "auth"},
			expected:   true,
		},
		{
			name:       "MigrationFormat_Match_MixedSyntax",
			expression: `threshold.name = "HighCPUUsage" && ruleId = "rule-uuid-123"`,
			labels:     map[string]string{"ruleId": "rule-uuid-123", "severity": "warning", "threshold.name": "HighCPUUsage"},
			expected:   true,
		},
		{
			name:       "MissingDeepDottedLabel_ReturnsFalse",
			expression: `k8s.namespace.name = "api"`,
			labels:     map[string]string{"env": "production"},
		},
		{
			name:       "MissingDottedLabel_ReturnsFalse",
			expression: `service.name = "api"`,
			labels:     map[string]string{"env": "production"},
		},
		{
			name:       "MissingLabelKey_NoMatch_ExprSyntax",
			expression: `"missing_key" == "value"`,
			labels:     map[string]string{"service": "auth"},
		},
		{
			name:       "MissingLabelKey_NoMatch_QBSyntax",
			expression: `"missing_key" = "value"`,
			labels:     map[string]string{"service": "auth"},
		},
		{
			name:       "MissingLeafUnderExistingPrefix_NoMatch",
			expression: "k8s.pod.name = 'checkout'",
			labels:     map[string]string{"k8s.node.name": "node-1"},
		},
		{
			name:          "NonBoolExpression_ReturnsError",
			expression:    "env",
			labels:        map[string]string{"env": "production"},
			expectedError: &tError{code: CodeResultNotBool},
		},
		{
			name:       "OperatorPrecedence_Match_ExprSyntax",
			expression: `service == "auth" && env == "prod" || critical == "true"`,
			labels:     map[string]string{"critical": "true", "env": "staging", "service": "payment"},
			expected:   true,
		},
		{
			name:       "OperatorPrecedence_Match_QBSyntax",
			expression: `service = "auth" AND env = "prod" OR critical = "true"`,
			labels:     map[string]string{"critical": "true", "env": "staging", "service": "payment"},
			expected:   true,
		},
		{
			name:       "Or_FirstFails_MissingDottedLabelSecond",
			expression: `env = "production" OR service.name = "api"`,
			labels:     map[string]string{"env": "staging"},
		},
		{
			name:       "Or_FirstMatches",
			expression: `env = "production" OR env = "staging"`,
			labels:     map[string]string{"env": "production"},
			expected:   true,
		},
		{
			name:       "Or_MissingDottedLabelFirst_SecondMatches",
			expression: `service.name = "api" OR http.status_code = "501"`,
			labels:     map[string]string{"http.status_code": "501"},
			expected:   true,
		},
		{
			name:       "Or_MissingDottedLabelSecond_OtherSideMatches",
			expression: "http.status_code = '501' OR service.name = 'otel-demo'",
			labels:     map[string]string{"http.status_code": "501"},
			expected:   true,
		},
		{
			name:       "Or_MissingKey_OtherSideMatches",
			expression: "threshold.name = 'warning' OR ruleId = 'rule1'",
			labels:     map[string]string{"threshold.name": "warning"},
			expected:   true,
		},
		{
			name:       "Or_NoneMatch",
			expression: `env = "production" OR env = "staging"`,
			labels:     map[string]string{"env": "development"},
		},
		{
			name:       "Or_NoneMatch_ExprSyntax",
			expression: `service == "payment" || env == "staging"`,
			labels:     map[string]string{"env": "production", "service": "auth"},
		},
		{
			name:       "Or_OneMatches_ExprSyntax",
			expression: `service == "payment" || env == "production"`,
			labels:     map[string]string{"env": "production", "service": "auth"},
			expected:   true,
		},
		{
			name:       "Or_PlainLabelUsedAsPrefix_OtherSideMatches",
			expression: "region.zone = 'us-east-1a' OR team = 'payments'",
			labels:     map[string]string{"region": "us-east-1", "team": "payments"},
			expected:   true,
		},
		{
			name:       "Or_SecondMatches",
			expression: `env = "production" OR env = "staging"`,
			labels:     map[string]string{"env": "staging"},
			expected:   true,
		},
		{
			name:       "Parentheses_Match_ExprSyntax",
			expression: `(service == "auth" && env == "production") || critical == "true"`,
			labels:     map[string]string{"critical": "true", "env": "staging", "service": "payment"},
			expected:   true,
		},
		{
			name:       "Parentheses_Match_QBSyntax",
			expression: `(service = "auth" AND env = "production") OR critical = "true"`,
			labels:     map[string]string{"critical": "true", "env": "staging", "service": "payment"},
			expected:   true,
		},
		{
			name:       "PlainLabelUsedAsPrefix_IsNil",
			expression: "deployment.environment = nil",
			labels:     map[string]string{"deployment": "blue"},
			expected:   true,
		},
		{
			name:       "QuotedThresholdName_NoMatch_ExprSyntax",
			expression: `'threshold.name' == "high-cpu" && ruleId == "rule-123"`,
			labels:     map[string]string{"ruleId": "rule-123", "service": "auth", "threshold.name": "high-cpu"},
		},
		{
			name:       "QuotedThresholdName_NoMatch_MixedSyntax",
			expression: `'threshold.name' = "high-cpu" AND ruleId == "rule-123"`,
			labels:     map[string]string{"ruleId": "rule-123", "service": "auth", "threshold.name": "high-cpu"},
		},
		{
			name:       "UndefinedLabel_ReturnsFalse",
			expression: `env = "production"`,
			labels:     map[string]string{"service": "api"},
		},
		{
			name:       "ValueNotEqualNumber_StringLabel_TypeInvalid",
			expression: "value != 92",
			labels:     map[string]string{"value": "92"},
			expectedError: &tError{
				code:    CodeTypeInvalid,
				message: "invalid operation: != (mismatched types string and int)",
				from:    6,
				to:      8,
				line:    1,
				column:  7,
			},
		},
		{
			name:       "ValueNotEqualNumber_MissingLabel_Matches",
			expression: "value != 92",
			labels:     map[string]string{"env": "prod"},
			expected:   true,
		},
		{
			name:       "ValueEqualsNumber_StringLabel_TypeInvalid",
			expression: "value = 92",
			labels:     map[string]string{"value": "92"},
			expectedError: &tError{
				code:    CodeTypeInvalid,
				message: "invalid operation: = (mismatched types string and int)",
				from:    6,
				to:      7,
				line:    1,
				column:  7,
			},
		},
		{
			name:       "ValueGreaterThan_StringLabel_TypeInvalid",
			expression: "value > 80",
			labels:     map[string]string{"value": "92"},
			expectedError: &tError{
				code:    CodeTypeInvalid,
				message: "invalid operation: > (mismatched types string and int)",
				from:    6,
				to:      7,
				line:    1,
				column:  7,
			},
		},
		{
			name:       "ValueGreaterThan_MissingLabel_RunFailed",
			expression: "value > 80",
			labels:     map[string]string{"env": "prod"},
			expectedError: &tError{
				code:    CodeRunFailed,
				message: "invalid operation: <nil> > int",
				from:    6,
				to:      7,
				line:    1,
				column:  7,
			},
		},
		{
			name:          "MissingName_NilResult_NotBool",
			expression:    "missing",
			labels:        map[string]string{"env": "prod"},
			expectedError: &tError{code: CodeResultNotBool},
		},
		{
			name:          "MissingNestedName_NilResult_NotBool",
			expression:    "missing.x",
			labels:        map[string]string{"env": "prod"},
			expectedError: &tError{code: CodeResultNotBool},
		},
		{
			name:          "StringLiteral_NotBool",
			expression:    `"abc"`,
			labels:        map[string]string{"env": "prod"},
			expectedError: &tError{code: CodeResultNotBool},
		},
		{
			name:       "PlainLabelAsPrefix_NotEqual_Matches",
			expression: `host.name != "x"`,
			labels:     map[string]string{"host": "web-1"},
			expected:   true,
		},
		{
			name:       "PlainLabelAsPrefix_Equal_NoMatch",
			expression: `host.name == "x"`,
			labels:     map[string]string{"host": "web-1"},
		},
		{
			name:       "PlainLabelIndexed_IsNil",
			expression: `team["x"] == nil`,
			labels:     map[string]string{"team": "alpha"},
			expected:   true,
		},
		{
			name:       "DottedLabel_Nested_Matches",
			expression: `service.name = "checkout"`,
			labels:     map[string]string{"service.name": "checkout"},
			expected:   true,
		},
		{
			name:       "PlainLabel_DottedAccess_NoMatch",
			expression: `service.name = "checkout"`,
			labels:     map[string]string{"service": "checkout"},
		},
		{
			name:       "DeeperThanLabel_IsNil",
			expression: "service.name.extra = nil",
			labels:     map[string]string{"service.name": "checkout"},
			expected:   true,
		},
		{
			name:       "DottedLabel_Regexp_Matches",
			expression: `k8s.pod.name REGEXP "^api-"`,
			labels:     map[string]string{"k8s.pod.name": "api-7f9c"},
			expected:   true,
		},
		{
			name:       "DottedLabel_KeywordMember_Matches",
			expression: `service.IN = "x"`,
			labels:     map[string]string{"service.IN": "x"},
			expected:   true,
		},
		{
			name:       "AndIn_Matches",
			expression: `env = "prod" AND region IN ["eu", "us"]`,
			labels:     map[string]string{"env": "prod", "region": "us"},
			expected:   true,
		},
		{
			name:       "TypeError_AfterAnd_PositionOnSource",
			expression: `env = "prod" AND value > 5`,
			labels:     map[string]string{"env": "prod", "value": "9"},
			expectedError: &tError{
				code:    CodeTypeInvalid,
				message: "invalid operation: > (mismatched types string and int)",
				from:    23,
				to:      24,
				line:    1,
				column:  24,
			},
		},
		{
			name:       "TypeError_AfterNonASCII_PositionInCodePoints",
			expression: `city = "Zürich" AND value > 5`,
			labels:     map[string]string{"city": "Zürich", "value": "9"},
			expectedError: &tError{
				code:    CodeTypeInvalid,
				message: "invalid operation: > (mismatched types string and int)",
				from:    26,
				to:      27,
				line:    1,
				column:  27,
			},
		},
		{
			name:       "TrailingAnd_UnexpectedEOF",
			expression: `env = "prod" AND`,
			labels:     map[string]string{"env": "prod"},
			expectedError: &tError{
				code:    CodeSyntaxInvalid,
				message: "unexpected token EOF",
				from:    15,
				to:      16,
				line:    1,
				column:  16,
			},
		},
		{
			name:       "LenOfLabel_Present_Matches",
			expression: "len(env) > 2",
			labels:     map[string]string{"env": "prod"},
			expected:   true,
		},
		{
			name:       "LenOfLabel_Missing_RunFailed",
			expression: "len(env) > 2",
			labels:     map[string]string{"region": "eu"},
			expectedError: &tError{
				code:    CodeRunFailed,
				message: "invalid argument for len (type <nil>)",
				from:    0,
				to:      3,
				line:    1,
				column:  1,
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			program, err := Compile(testCase.expression, TypedDynamic())
			if err == nil {
				vars, _ := Nest(testCase.labels)

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

// The expected errors were generated once by compiling every case on the
// SigNoz/expr fork (v1.17.7-beta) with AllowUndefinedVariables and AsBool.
func TestUntyped_Compile(t *testing.T) {
	testCases := []struct {
		name          string
		expression    string
		expectedError *tError
	}{
		{
			name:       "Equals_Accepted",
			expression: `env = "prod"`,
		},
		{
			name:       "AndIn_Accepted",
			expression: `env = "prod" AND region IN ["eu"]`,
		},
		{
			name:       "DottedContains_Accepted",
			expression: `service.name CONTAINS "api"`,
		},
		{
			name:       "CompareUnknownName_Accepted",
			expression: "value > 80",
		},
		{
			name:       "BareName_Accepted",
			expression: "missing",
		},
		{
			name:       "BareNestedName_Accepted",
			expression: "missing.x",
		},
		{
			name:       "StringLiteral_NotBool",
			expression: `"abc"`,
			expectedError: &tError{
				code:    CodeResultNotBool,
				message: "expected bool, but got string",
			},
		},
		{
			name:       "Arithmetic_NotBool",
			expression: "1 + 1",
			expectedError: &tError{
				code:    CodeResultNotBool,
				message: "expected bool, but got int",
			},
		},
		{
			name:       "NilLiteral_NotBool",
			expression: "nil",
			expectedError: &tError{
				code:    CodeResultNotBool,
				message: "expected bool, but got unknown",
			},
		},
		{
			name:       "LenComparison_Accepted",
			expression: "len(env) > 2",
		},
		{
			name:       "TrailingAnd_UnexpectedEOF",
			expression: `env = "prod" AND`,
			expectedError: &tError{
				code:    CodeSyntaxInvalid,
				message: "unexpected token EOF",
				from:    15,
				to:      16,
				line:    1,
				column:  16,
			},
		},
		{
			name:       "DoubleEquals_SyntaxInvalid",
			expression: "count = = 1",
			expectedError: &tError{
				code:    CodeSyntaxInvalid,
				message: `unexpected token Operator("=")`,
				from:    8,
				to:      9,
				line:    1,
				column:  9,
			},
		},
		{
			name:       "LiteralTypeError_TypeInvalid",
			expression: `"a" > 1`,
			expectedError: &tError{
				code:    CodeTypeInvalid,
				message: "invalid operation: > (mismatched types string and int)",
				from:    4,
				to:      5,
				line:    1,
				column:  5,
			},
		},
		{
			name:       "LiteralTypeError_AfterNonASCII_PositionInCodePoints",
			expression: `city = "Zürich" AND "a" > 1`,
			expectedError: &tError{
				code:    CodeTypeInvalid,
				message: "invalid operation: > (mismatched types string and int)",
				from:    24,
				to:      25,
				line:    1,
				column:  25,
			},
		},
		{
			name:       "RegexpInvalid_TypeInvalid",
			expression: `name REGEXP "[a"`,
			expectedError: &tError{
				code:    CodeTypeInvalid,
				message: "error parsing regexp: missing closing ]: `[a`",
				from:    5,
				to:      11,
				line:    1,
				column:  6,
			},
		},
		{
			name:       "MemberNameRegexp_Accepted",
			expression: `attributes.regexp = "x"`,
		},
		{
			name:       "LetBinder_Accepted",
			expression: "let x = 1; x = 1",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := Compile(testCase.expression, Untyped())

			if testCase.expectedError != nil {
				tAssertError(t, testCase.expectedError, err)

				return
			}

			require.NoError(t, err)
		})
	}
}

// The expected results and errors were generated once by compiling every
// expression on upstream expr v1.17.7 with AllowUndefinedVariables, and
// with AsBool for conditions, then running it with each set of variables.
func TestUntyped_Run(t *testing.T) {
	stringKey := map[string]any{"attributes": map[string]any{"key": "checkbody", "service.name": "api"}, "body": "this has CheckBody 101regex", "resource": map[string]any{"service.name": "api"}, "severity_number": 17, "severity_text": "ERROR", "trace_id": nil}
	intKey := map[string]any{"attributes": map[string]any{"cluster": "alpha", "key": int64(5)}, "body": "101regex", "resource": map[string]any{}, "severity_number": 9, "severity_text": "INFO", "trace_id": nil}
	empty := map[string]any{"attributes": map[string]any{}, "body": nil, "resource": map[string]any{"service.name": "worker"}, "severity_number": 0, "severity_text": "", "trace_id": nil}

	testCases := []struct {
		name          string
		compile       func(string, Env) (*Program, error)
		expression    string
		vars          map[string]any
		expected      any
		expectedError *tError
	}{
		{
			name:       "Condition_KeyEquals_StringKeyWithBody",
			compile:    Compile,
			expression: `attributes["key"] == "checkbody"`,
			vars:       stringKey,
			expected:   true,
		},
		{
			name:       "Condition_KeyEquals_IntKeyNoTrace",
			compile:    Compile,
			expression: `attributes["key"] == "checkbody"`,
			vars:       intKey,
			expected:   false,
		},
		{
			name:       "Condition_KeyEquals_EmptyAttributesNilBody",
			compile:    Compile,
			expression: `attributes["key"] == "checkbody"`,
			vars:       empty,
			expected:   false,
		},
		{
			name:       "Condition_KeyNotEquals_StringKeyWithBody",
			compile:    Compile,
			expression: `attributes["key"] != "checkbody"`,
			vars:       stringKey,
			expected:   false,
		},
		{
			name:       "Condition_KeyNotEquals_IntKeyNoTrace",
			compile:    Compile,
			expression: `attributes["key"] != "checkbody"`,
			vars:       intKey,
			expected:   true,
		},
		{
			name:       "Condition_KeyNotEquals_EmptyAttributesNilBody",
			compile:    Compile,
			expression: `attributes["key"] != "checkbody"`,
			vars:       empty,
			expected:   true,
		},
		{
			name:       "Condition_KeyLessThanTen_StringKeyWithBody",
			compile:    Compile,
			expression: `attributes["key"] != nil && attributes["key"] < 10`,
			vars:       stringKey,
			expectedError: &tError{
				code:    CodeRunFailed,
				message: "invalid operation: string < int",
				from:    46,
				to:      47,
				line:    1,
				column:  47,
			},
		},
		{
			name:       "Condition_KeyLessThanTen_IntKeyNoTrace",
			compile:    Compile,
			expression: `attributes["key"] != nil && attributes["key"] < 10`,
			vars:       intKey,
			expected:   true,
		},
		{
			name:       "Condition_KeyLessThanTen_EmptyAttributesNilBody",
			compile:    Compile,
			expression: `attributes["key"] != nil && attributes["key"] < 10`,
			vars:       empty,
			expected:   false,
		},
		{
			name:       "Condition_KeyAtLeastTen_StringKeyWithBody",
			compile:    Compile,
			expression: `attributes["key"] != nil && attributes["key"] >= 10`,
			vars:       stringKey,
			expectedError: &tError{
				code:    CodeRunFailed,
				message: "invalid operation: string >= int",
				from:    46,
				to:      48,
				line:    1,
				column:  47,
			},
		},
		{
			name:       "Condition_KeyAtLeastTen_IntKeyNoTrace",
			compile:    Compile,
			expression: `attributes["key"] != nil && attributes["key"] >= 10`,
			vars:       intKey,
			expected:   false,
		},
		{
			name:       "Condition_KeyAtLeastTen_EmptyAttributesNilBody",
			compile:    Compile,
			expression: `attributes["key"] != nil && attributes["key"] >= 10`,
			vars:       empty,
			expected:   false,
		},
		{
			name:       "Condition_BodyContainsIgnoringCase_StringKeyWithBody",
			compile:    Compile,
			expression: `body != nil && lower(body) contains lower("checkbody")`,
			vars:       stringKey,
			expected:   true,
		},
		{
			name:       "Condition_BodyContainsIgnoringCase_IntKeyNoTrace",
			compile:    Compile,
			expression: `body != nil && lower(body) contains lower("checkbody")`,
			vars:       intKey,
			expected:   false,
		},
		{
			name:       "Condition_BodyContainsIgnoringCase_EmptyAttributesNilBody",
			compile:    Compile,
			expression: `body != nil && lower(body) contains lower("checkbody")`,
			vars:       empty,
			expected:   false,
		},
		{
			name:       "Condition_BodyNotContainsIgnoringCase_StringKeyWithBody",
			compile:    Compile,
			expression: `body != nil && lower(body) not contains lower("checkbody")`,
			vars:       stringKey,
			expected:   false,
		},
		{
			name:       "Condition_BodyNotContainsIgnoringCase_IntKeyNoTrace",
			compile:    Compile,
			expression: `body != nil && lower(body) not contains lower("checkbody")`,
			vars:       intKey,
			expected:   true,
		},
		{
			name:       "Condition_BodyNotContainsIgnoringCase_EmptyAttributesNilBody",
			compile:    Compile,
			expression: `body != nil && lower(body) not contains lower("checkbody")`,
			vars:       empty,
			expected:   false,
		},
		{
			name:       "Condition_BodyMatches_StringKeyWithBody",
			compile:    Compile,
			expression: `body != nil && body matches "[0-1]+regex$"`,
			vars:       stringKey,
			expected:   true,
		},
		{
			name:       "Condition_BodyMatches_IntKeyNoTrace",
			compile:    Compile,
			expression: `body != nil && body matches "[0-1]+regex$"`,
			vars:       intKey,
			expected:   true,
		},
		{
			name:       "Condition_BodyMatches_EmptyAttributesNilBody",
			compile:    Compile,
			expression: `body != nil && body matches "[0-1]+regex$"`,
			vars:       empty,
			expected:   false,
		},
		{
			name:       "Condition_BodyNotMatches_StringKeyWithBody",
			compile:    Compile,
			expression: `body != nil && body not matches "[0-1]+regex$"`,
			vars:       stringKey,
			expected:   false,
		},
		{
			name:       "Condition_BodyNotMatches_IntKeyNoTrace",
			compile:    Compile,
			expression: `body != nil && body not matches "[0-1]+regex$"`,
			vars:       intKey,
			expected:   false,
		},
		{
			name:       "Condition_BodyNotMatches_EmptyAttributesNilBody",
			compile:    Compile,
			expression: `body != nil && body not matches "[0-1]+regex$"`,
			vars:       empty,
			expected:   false,
		},
		{
			name:       "Condition_KeyInNumbers_StringKeyWithBody",
			compile:    Compile,
			expression: `attributes["key"] != nil && attributes["key"] in [1,2,3,4,5]`,
			vars:       stringKey,
			expected:   false,
		},
		{
			name:       "Condition_KeyInNumbers_IntKeyNoTrace",
			compile:    Compile,
			expression: `attributes["key"] != nil && attributes["key"] in [1,2,3,4,5]`,
			vars:       intKey,
			expected:   true,
		},
		{
			name:       "Condition_KeyInNumbers_EmptyAttributesNilBody",
			compile:    Compile,
			expression: `attributes["key"] != nil && attributes["key"] in [1,2,3,4,5]`,
			vars:       empty,
			expected:   false,
		},
		{
			name:       "Condition_KeyNotInStrings_StringKeyWithBody",
			compile:    Compile,
			expression: `attributes["key"] != nil && attributes["key"] not in ['1','2']`,
			vars:       stringKey,
			expected:   true,
		},
		{
			name:       "Condition_KeyNotInStrings_IntKeyNoTrace",
			compile:    Compile,
			expression: `attributes["key"] != nil && attributes["key"] not in ['1','2']`,
			vars:       intKey,
			expectedError: &tError{
				code:    CodeRunFailed,
				message: "reflect.Value.MapIndex: value of type int64 is not assignable to type string",
				from:    50,
				to:      52,
				line:    1,
				column:  51,
			},
		},
		{
			name:       "Condition_KeyNotInStrings_EmptyAttributesNilBody",
			compile:    Compile,
			expression: `attributes["key"] != nil && attributes["key"] not in ['1','2']`,
			vars:       empty,
			expected:   false,
		},
		{
			name:       "Condition_KeyPresent_StringKeyWithBody",
			compile:    Compile,
			expression: `"key" in attributes`,
			vars:       stringKey,
			expected:   true,
		},
		{
			name:       "Condition_KeyPresent_IntKeyNoTrace",
			compile:    Compile,
			expression: `"key" in attributes`,
			vars:       intKey,
			expected:   true,
		},
		{
			name:       "Condition_KeyPresent_EmptyAttributesNilBody",
			compile:    Compile,
			expression: `"key" in attributes`,
			vars:       empty,
			expected:   false,
		},
		{
			name:       "Condition_KeyAbsent_StringKeyWithBody",
			compile:    Compile,
			expression: `"key" not in attributes`,
			vars:       stringKey,
			expected:   false,
		},
		{
			name:       "Condition_KeyAbsent_IntKeyNoTrace",
			compile:    Compile,
			expression: `"key" not in attributes`,
			vars:       intKey,
			expected:   false,
		},
		{
			name:       "Condition_KeyAbsent_EmptyAttributesNilBody",
			compile:    Compile,
			expression: `"key" not in attributes`,
			vars:       empty,
			expected:   true,
		},
		{
			name:       "Condition_TraceIDNil_StringKeyWithBody",
			compile:    Compile,
			expression: "trace_id == nil",
			vars:       stringKey,
			expected:   true,
		},
		{
			name:       "Condition_TraceIDNil_IntKeyNoTrace",
			compile:    Compile,
			expression: "trace_id == nil",
			vars:       intKey,
			expected:   true,
		},
		{
			name:       "Condition_TraceIDNil_EmptyAttributesNilBody",
			compile:    Compile,
			expression: "trace_id == nil",
			vars:       empty,
			expected:   true,
		},
		{
			name:       "Condition_TraceIDNotNil_StringKeyWithBody",
			compile:    Compile,
			expression: "trace_id != nil",
			vars:       stringKey,
			expected:   false,
		},
		{
			name:       "Condition_TraceIDNotNil_IntKeyNoTrace",
			compile:    Compile,
			expression: "trace_id != nil",
			vars:       intKey,
			expected:   false,
		},
		{
			name:       "Condition_TraceIDNotNil_EmptyAttributesNilBody",
			compile:    Compile,
			expression: "trace_id != nil",
			vars:       empty,
			expected:   false,
		},
		{
			name:       "Condition_CombinedFilter_StringKeyWithBody",
			compile:    Compile,
			expression: `attributes["key"] != nil && attributes["key"] <= 10 and body != nil && body not matches "[0-1]+regex$" and "key" not in attributes`,
			vars:       stringKey,
			expectedError: &tError{
				code:    CodeRunFailed,
				message: "invalid operation: string <= int",
				from:    46,
				to:      48,
				line:    1,
				column:  47,
			},
		},
		{
			name:       "Condition_CombinedFilter_IntKeyNoTrace",
			compile:    Compile,
			expression: `attributes["key"] != nil && attributes["key"] <= 10 and body != nil && body not matches "[0-1]+regex$" and "key" not in attributes`,
			vars:       intKey,
			expected:   false,
		},
		{
			name:       "Condition_CombinedFilter_EmptyAttributesNilBody",
			compile:    Compile,
			expression: `attributes["key"] != nil && attributes["key"] <= 10 and body != nil && body not matches "[0-1]+regex$" and "key" not in attributes`,
			vars:       empty,
			expected:   false,
		},
		{
			name:       "Condition_SeverityTextOrNumber_StringKeyWithBody",
			compile:    Compile,
			expression: `severity_text == "ERROR" or severity_number >= 17`,
			vars:       stringKey,
			expected:   true,
		},
		{
			name:       "Condition_SeverityTextOrNumber_IntKeyNoTrace",
			compile:    Compile,
			expression: `severity_text == "ERROR" or severity_number >= 17`,
			vars:       intKey,
			expected:   false,
		},
		{
			name:       "Condition_SeverityTextOrNumber_EmptyAttributesNilBody",
			compile:    Compile,
			expression: `severity_text == "ERROR" or severity_number >= 17`,
			vars:       empty,
			expected:   false,
		},
		{
			name:       "Condition_ResourceServiceName_StringKeyWithBody",
			compile:    Compile,
			expression: `resource["service.name"] == "api"`,
			vars:       stringKey,
			expected:   true,
		},
		{
			name:       "Condition_ResourceServiceName_IntKeyNoTrace",
			compile:    Compile,
			expression: `resource["service.name"] == "api"`,
			vars:       intKey,
			expected:   false,
		},
		{
			name:       "Condition_ResourceServiceName_EmptyAttributesNilBody",
			compile:    Compile,
			expression: `resource["service.name"] == "api"`,
			vars:       empty,
			expected:   false,
		},
		{
			name:       "Condition_MissingKey_NilIsFalse_StringKeyWithBody",
			compile:    Compile,
			expression: `attributes["missing"]`,
			vars:       stringKey,
			expected:   false,
		},
		{
			name:       "Condition_MissingKey_NilIsFalse_IntKeyNoTrace",
			compile:    Compile,
			expression: `attributes["missing"]`,
			vars:       intKey,
			expected:   false,
		},
		{
			name:       "Condition_MissingKey_NilIsFalse_EmptyAttributesNilBody",
			compile:    Compile,
			expression: `attributes["missing"]`,
			vars:       empty,
			expected:   false,
		},
		{
			name:       "Condition_NestedMemberOfMissing_StringKeyWithBody",
			compile:    Compile,
			expression: `attributes.cluster.name == "c1"`,
			vars:       stringKey,
			expectedError: &tError{
				code:    CodeRunFailed,
				message: "cannot fetch name from <nil>",
				from:    19,
				to:      23,
				line:    1,
				column:  20,
			},
		},
		{
			name:       "Condition_NestedMemberOfMissing_IntKeyNoTrace",
			compile:    Compile,
			expression: `attributes.cluster.name == "c1"`,
			vars:       intKey,
			expectedError: &tError{
				code:    CodeRunFailed,
				message: "invalid operation: int(string)",
				from:    19,
				to:      23,
				line:    1,
				column:  20,
			},
		},
		{
			name:       "Condition_NestedMemberOfMissing_EmptyAttributesNilBody",
			compile:    Compile,
			expression: `attributes.cluster.name == "c1"`,
			vars:       empty,
			expectedError: &tError{
				code:    CodeRunFailed,
				message: "cannot fetch name from <nil>",
				from:    19,
				to:      23,
				line:    1,
				column:  20,
			},
		},
		{
			name:       "Condition_GuardedMember_StringKeyWithBody",
			compile:    Compile,
			expression: `attributes.cluster != nil && attributes.cluster == "alpha"`,
			vars:       stringKey,
			expected:   false,
		},
		{
			name:       "Condition_GuardedMember_IntKeyNoTrace",
			compile:    Compile,
			expression: `attributes.cluster != nil && attributes.cluster == "alpha"`,
			vars:       intKey,
			expected:   true,
		},
		{
			name:       "Condition_GuardedMember_EmptyAttributesNilBody",
			compile:    Compile,
			expression: `attributes.cluster != nil && attributes.cluster == "alpha"`,
			vars:       empty,
			expected:   false,
		},
		{
			name:       "Condition_KeyGreaterThanOne_StringKeyWithBody",
			compile:    Compile,
			expression: `attributes["key"] > 1`,
			vars:       stringKey,
			expectedError: &tError{
				code:    CodeRunFailed,
				message: "invalid operation: string > int",
				from:    18,
				to:      19,
				line:    1,
				column:  19,
			},
		},
		{
			name:       "Condition_KeyGreaterThanOne_IntKeyNoTrace",
			compile:    Compile,
			expression: `attributes["key"] > 1`,
			vars:       intKey,
			expected:   true,
		},
		{
			name:       "Condition_KeyGreaterThanOne_EmptyAttributesNilBody",
			compile:    Compile,
			expression: `attributes["key"] > 1`,
			vars:       empty,
			expectedError: &tError{
				code:    CodeRunFailed,
				message: "invalid operation: <nil> > int",
				from:    18,
				to:      19,
				line:    1,
				column:  19,
			},
		},
		{
			name:       "Condition_StringLiteral_StringKeyWithBody",
			compile:    Compile,
			expression: `"abc"`,
			vars:       stringKey,
			expectedError: &tError{
				code:    CodeResultNotBool,
				message: "expected bool, but got string",
			},
		},
		{
			name:       "Condition_StringLiteral_IntKeyNoTrace",
			compile:    Compile,
			expression: `"abc"`,
			vars:       intKey,
			expectedError: &tError{
				code:    CodeResultNotBool,
				message: "expected bool, but got string",
			},
		},
		{
			name:       "Condition_StringLiteral_EmptyAttributesNilBody",
			compile:    Compile,
			expression: `"abc"`,
			vars:       empty,
			expectedError: &tError{
				code:    CodeResultNotBool,
				message: "expected bool, but got string",
			},
		},
		{
			name:       "Value_Key_StringKeyWithBody",
			compile:    CompileValue,
			expression: `attributes["key"]`,
			vars:       stringKey,
			expected:   "checkbody",
		},
		{
			name:       "Value_Key_IntKeyNoTrace",
			compile:    CompileValue,
			expression: `attributes["key"]`,
			vars:       intKey,
			expected:   int64(5),
		},
		{
			name:       "Value_Key_EmptyAttributesNilBody",
			compile:    CompileValue,
			expression: `attributes["key"]`,
			vars:       empty,
		},
		{
			name:       "Value_SeverityPlusOne_StringKeyWithBody",
			compile:    CompileValue,
			expression: "severity_number + 1",
			vars:       stringKey,
			expected:   18,
		},
		{
			name:       "Value_SeverityPlusOne_IntKeyNoTrace",
			compile:    CompileValue,
			expression: "severity_number + 1",
			vars:       intKey,
			expected:   10,
		},
		{
			name:       "Value_SeverityPlusOne_EmptyAttributesNilBody",
			compile:    CompileValue,
			expression: "severity_number + 1",
			vars:       empty,
			expected:   1,
		},
		{
			name:       "Value_LowerSeverityText_StringKeyWithBody",
			compile:    CompileValue,
			expression: "lower(severity_text)",
			vars:       stringKey,
			expected:   "error",
		},
		{
			name:       "Value_LowerSeverityText_IntKeyNoTrace",
			compile:    CompileValue,
			expression: "lower(severity_text)",
			vars:       intKey,
			expected:   "info",
		},
		{
			name:       "Value_LowerSeverityText_EmptyAttributesNilBody",
			compile:    CompileValue,
			expression: "lower(severity_text)",
			vars:       empty,
			expected:   "",
		},
		{
			name:       "Value_MissingKey_StringKeyWithBody",
			compile:    CompileValue,
			expression: `attributes["missing"]`,
			vars:       stringKey,
		},
		{
			name:       "Value_MissingKey_IntKeyNoTrace",
			compile:    CompileValue,
			expression: `attributes["missing"]`,
			vars:       intKey,
		},
		{
			name:       "Value_MissingKey_EmptyAttributesNilBody",
			compile:    CompileValue,
			expression: `attributes["missing"]`,
			vars:       empty,
		},
		{
			name:       "Value_ResourceServiceSuffix_StringKeyWithBody",
			compile:    CompileValue,
			expression: `resource["service.name"] + "-svc"`,
			vars:       stringKey,
			expected:   "api-svc",
		},
		{
			name:       "Value_ResourceServiceSuffix_IntKeyNoTrace",
			compile:    CompileValue,
			expression: `resource["service.name"] + "-svc"`,
			vars:       intKey,
			expectedError: &tError{
				code:    CodeRunFailed,
				message: "invalid operation: <nil> + string",
				from:    25,
				to:      26,
				line:    1,
				column:  26,
			},
		},
		{
			name:       "Value_ResourceServiceSuffix_EmptyAttributesNilBody",
			compile:    CompileValue,
			expression: `resource["service.name"] + "-svc"`,
			vars:       empty,
			expected:   "worker-svc",
		},
		{
			name:       "Value_AttributeCount_StringKeyWithBody",
			compile:    CompileValue,
			expression: "len(attributes)",
			vars:       stringKey,
			expected:   2,
		},
		{
			name:       "Value_AttributeCount_IntKeyNoTrace",
			compile:    CompileValue,
			expression: "len(attributes)",
			vars:       intKey,
			expected:   2,
		},
		{
			name:       "Value_AttributeCount_EmptyAttributesNilBody",
			compile:    CompileValue,
			expression: "len(attributes)",
			vars:       empty,
			expected:   0,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			program, err := testCase.compile(testCase.expression, Untyped())
			if err == nil {
				var got any

				got, err = program.Run(testCase.vars)
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

func TestCompile_NodeLimit(t *testing.T) {
	testCases := []struct {
		name string
		env  Env
	}{
		{
			name: "Untyped_NodesExceeded",
			env:  Untyped(),
		},
		{
			name: "TypedDynamic_NodesExceeded",
			env:  TypedDynamic(),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := Compile("a"+strings.Repeat(" + a", 10_000)+" > 0", testCase.env)
			require.Error(t, err)
			assert.Equal(t, CodeNodesExceeded, errors.CodeOf(err))
		})
	}
}

func TestProgram_Bool(t *testing.T) {
	testCases := []struct {
		name            string
		compile         func(string, Env) (*Program, error)
		src             string
		env             Env
		vars            any
		expected        bool
		expectedCode    errors.Code
		expectedMessage string
	}{
		{
			name:     "TypedDynamic_BoolResult_Matches",
			compile:  Compile,
			src:      `tier = "gold"`,
			env:      TypedDynamic(),
			vars:     map[string]any{"tier": "gold"},
			expected: true,
		},
		{
			name:    "Untyped_NilResult_False",
			compile: Compile,
			src:     `plan`,
			env:     Untyped(),
			vars:    map[string]any{},
		},
		{
			name:            "Value_StringResult_NotBool",
			compile:         CompileValue,
			src:             `name`,
			env:             TypedDynamic(),
			vars:            map[string]any{"name": "alpha"},
			expectedCode:    CodeResultNotBool,
			expectedMessage: "result is string (alpha), expected bool",
		},
		{
			name:            "TypedDynamic_StringMapVars_RunFailed",
			compile:         Compile,
			src:             `env = "prod"`,
			env:             TypedDynamic(),
			vars:            map[string]string{"env": "prod"},
			expectedCode:    CodeRunFailed,
			expectedMessage: "vars are map[string]string, expected map[string]any",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			program, err := testCase.compile(testCase.src, testCase.env)
			require.NoError(t, err)

			got, err := program.Bool(testCase.vars)
			if testCase.expectedCode != (errors.Code{}) {
				require.Error(t, err)
				assert.Equal(t, testCase.expectedCode, errors.CodeOf(err))
				assert.Equal(t, testCase.expectedMessage, errors.MessageOf(err))

				return
			}

			require.NoError(t, err)
			assert.Equal(t, testCase.expected, got)
		})
	}
}

// Label sets of different shapes, evaluated at once, each get the program
// typed for their own shape.
func TestProgram_ConcurrentShapes(t *testing.T) {
	program, err := Compile(`value.x = "y" OR env = "prod"`, TypedDynamic())
	require.NoError(t, err)

	testCases := []struct {
		name     string
		vars     map[string]any
		expected bool
	}{
		{
			name:     "StringValue_NoMatch",
			vars:     map[string]any{"value": "92"},
			expected: false,
		},
		{
			name:     "MapValue_Matches",
			vars:     map[string]any{"value": map[string]any{"x": "y"}},
			expected: true,
		},
		{
			name:     "EnvOnly_Matches",
			vars:     map[string]any{"env": "prod"},
			expected: true,
		},
		{
			name:     "NilValue_NoMatch",
			vars:     map[string]any{"value": nil, "env": "staging"},
			expected: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var wg sync.WaitGroup
			for range 50 {
				wg.Go(func() {
					got, err := program.Bool(testCase.vars)
					assert.NoError(t, err)
					assert.Equal(t, testCase.expected, got)
				})
			}

			wg.Wait()
		})
	}
}

func TestProgram_Source(t *testing.T) {
	testCases := []struct {
		name string
		src  string
		env  Env
	}{
		{
			name: "Untyped_AsWritten",
			src:  `env = "prod" AND region IN ["eu"]`,
			env:  Untyped(),
		},
		{
			name: "TypedDynamic_AsWritten",
			src:  `tier REGEXP "^g" OR NOT (plan = "free")`,
			env:  TypedDynamic(),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			program, err := Compile(testCase.src, testCase.env)
			require.NoError(t, err)
			assert.Equal(t, testCase.src, program.Source())
		})
	}
}
