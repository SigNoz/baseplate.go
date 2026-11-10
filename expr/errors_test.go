package expr

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/SigNoz/baseplate.go/errors"
)

// tError is an expected error. A zero code, an empty message or nil
// suggestions aren't checked, and a zero line means the error has no
// position.
type tError struct {
	code        errors.Code
	message     string
	suggestions []string
	from        int
	to          int
	line        int
	column      int
}

func tAssertError(t *testing.T, expected *tError, err error) {
	t.Helper()

	require.Error(t, err)

	if expected.code != (errors.Code{}) {
		assert.Equal(t, expected.code, errors.CodeOf(err))
	}

	if expected.message != "" {
		assert.Equal(t, expected.message, errors.MessageOf(err))
	}

	if expected.suggestions != nil {
		assert.Equal(t, expected.suggestions, errors.AsException(err).Suggestions)
	}

	position, ok := errors.AsType[*PositionError](err)
	if expected.line == 0 {
		assert.False(t, ok, "unexpected position")

		return
	}

	require.True(t, ok, "missing position")
	assert.Equal(t, expected.from, position.From)
	assert.Equal(t, expected.to, position.To)
	assert.Equal(t, expected.line, position.Line)
	assert.Equal(t, expected.column, position.Column)
}

func TestPositionError_Error(t *testing.T) {
	testCases := []struct {
		name          string
		src           string
		expectedError string
	}{
		{
			name:          "OneLine_CaretUnderError",
			src:           `region = = "eu"`,
			expectedError: "(1:10)\n | region = = \"eu\"\n | .........^",
		},
		{
			name:          "SecondLine_LineCounted",
			src:           "tier = \"gold\"\nAND",
			expectedError: "(2:3)\n | AND\n | ..^",
		},
		{
			name:          "AfterNonASCII_ColumnInCodePoints",
			src:           `city = "Zürich" AND "a" > 1`,
			expectedError: "(1:25)\n | city = \"Zürich\" AND \"a\" > 1\n | ........................^",
		},
		{
			name:          "AfterEmoji_ColumnInCodePoints",
			src:           `tag = "🚀" AND = 1`,
			expectedError: "(1:15)\n | tag = \"🚀\" AND = 1\n | ..............^",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := Compile(testCase.src, Untyped())
			require.Error(t, err)

			position, ok := errors.AsType[*PositionError](err)
			require.True(t, ok)
			assert.Equal(t, testCase.expectedError, position.Error())
		})
	}
}
