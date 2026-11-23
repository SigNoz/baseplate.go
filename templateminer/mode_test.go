package templateminer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/SigNoz/baseplate.go/errors"
)

func TestNewMode(t *testing.T) {
	testCases := []struct {
		name         string
		input        string
		expectedMode Mode
		expectedErr  bool
	}{
		{
			name:         "Stream_Valid",
			input:        "stream",
			expectedMode: ModeStream,
		},
		{
			name:        "Batch_NotDeclared",
			input:       "batch",
			expectedErr: true,
		},
		{
			name:        "Empty_Invalid",
			input:       "",
			expectedErr: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			mode, err := NewMode(testCase.input)

			if testCase.expectedErr {
				require.Error(t, err)
				assert.True(t, errors.Asc(err, CodeModeInvalid))

				return
			}

			require.NoError(t, err)
			assert.Equal(t, testCase.expectedMode, mode)
		})
	}
}

func TestModeText_RoundTrip(t *testing.T) {
	text, err := ModeStream.MarshalText()
	require.NoError(t, err)

	var mode Mode
	require.NoError(t, mode.UnmarshalText(text))
	assert.Equal(t, ModeStream, mode)
	assert.Error(t, mode.UnmarshalText([]byte("drain")))
}
