package drain3miner

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/SigNoz/baseplate.go/errors"
)

func TestNew(t *testing.T) {
	testCases := []struct {
		name         string
		edit         func(*Config)
		expectedCode errors.Code
	}{
		{name: "Defaults_Valid"},
		{
			name:         "ZeroMode_Invalid",
			edit:         func(config *Config) { config.Mode = Mode{} },
			expectedCode: CodeModeInvalid,
		},
		{
			name:         "DepthTwo_Invalid",
			edit:         func(config *Config) { config.Drain.Depth = 2 },
			expectedCode: CodeConfigInvalid,
		},
		{
			name:         "ThresholdAboveOne_Invalid",
			edit:         func(config *Config) { config.Drain.SimThreshold = 1.5 },
			expectedCode: CodeConfigInvalid,
		},
		{
			name:         "ThresholdNaN_Invalid",
			edit:         func(config *Config) { config.Drain.SimThreshold = math.NaN() },
			expectedCode: CodeConfigInvalid,
		},
		{
			name:         "ZeroMaxChildren_Invalid",
			edit:         func(config *Config) { config.Drain.MaxChildren = 0 },
			expectedCode: CodeConfigInvalid,
		},
		{
			name:         "NegativeMaxClusters_Invalid",
			edit:         func(config *Config) { config.Drain.MaxClusters = -1 },
			expectedCode: CodeConfigInvalid,
		},
		{
			name:         "EmptyParamString_Invalid",
			edit:         func(config *Config) { config.Drain.ParamString = "" },
			expectedCode: CodeConfigInvalid,
		},
		{
			name:         "ParamStringWithSpace_Invalid",
			edit:         func(config *Config) { config.Drain.ParamString = "< * >" },
			expectedCode: CodeConfigInvalid,
		},
		{
			name:         "EmptyDelimiter_Invalid",
			edit:         func(config *Config) { config.Drain.ExtraDelimiters = []string{",", ""} },
			expectedCode: CodeConfigInvalid,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			config := DefaultConfig()
			if testCase.edit != nil {
				testCase.edit(&config)
			}

			miner, err := New(config)

			if testCase.expectedCode == (errors.Code{}) {
				require.NoError(t, err)
				assert.NotNil(t, miner)

				return
			}

			require.Error(t, err)
			assert.Nil(t, miner)
			assert.True(t, errors.Ast(err, errors.TypeInvalidInput))
			assert.True(t, errors.Asc(err, testCase.expectedCode))
		})
	}
}
