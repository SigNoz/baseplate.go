package cobraenv

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/SigNoz/baseplate.go/errors"
)

type tFlags struct {
	address  string
	logLevel string
	port     int
	peers    []string
	debug    bool
}

// tExecute runs "<prefix> run <args>" with Parse in the root's
// PersistentPreRunE, a persistent --address flag on the root and the other
// flags on the run subcommand.
func tExecute(t *testing.T, prefix string, args ...string) (tFlags, error) {
	t.Helper()

	var flags tFlags

	root := &cobra.Command{
		Use:           "root",
		SilenceErrors: true,
		SilenceUsage:  true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			return Parse(cmd, prefix)
		},
	}
	root.PersistentFlags().StringVar(&flags.address, "address", "localhost:8080", "")

	run := &cobra.Command{
		Use:  "run",
		RunE: func(*cobra.Command, []string) error { return nil },
	}
	run.Flags().StringVar(&flags.logLevel, "log-level", "info", "")
	run.Flags().IntVar(&flags.port, "port", 4317, "")
	run.Flags().StringSliceVar(&flags.peers, "peers", []string{"self"}, "")
	run.Flags().BoolVar(&flags.debug, "debug", false, "")
	root.AddCommand(run)

	root.SetArgs(append([]string{"run"}, args...))

	err := root.Execute()

	return flags, err
}

func TestParse(t *testing.T) {
	testCases := []struct {
		name          string
		prefix        string
		env           map[string]string
		args          []string
		expectedFlags tFlags
	}{
		{
			name:          "NoEnv_Defaults",
			prefix:        "alpha",
			expectedFlags: tFlags{address: "localhost:8080", logLevel: "info", port: 4317, peers: []string{"self"}},
		},
		{
			name:          "Env_FillsFlags",
			prefix:        "bravo",
			env:           map[string]string{"BRAVO_LOG_LEVEL": "debug", "BRAVO_PORT": "9000", "BRAVO_DEBUG": "true"},
			expectedFlags: tFlags{address: "localhost:8080", logLevel: "debug", port: 9000, peers: []string{"self"}, debug: true},
		},
		{
			name:          "EnvAndArg_ArgWins",
			prefix:        "charlie",
			env:           map[string]string{"CHARLIE_LOG_LEVEL": "warn", "CHARLIE_PORT": "9001"},
			args:          []string{"--log-level", "error"},
			expectedFlags: tFlags{address: "localhost:8080", logLevel: "error", port: 9001, peers: []string{"self"}},
		},
		{
			name:          "EmptyEnv_Ignored",
			prefix:        "delta",
			env:           map[string]string{"DELTA_LOG_LEVEL": ""},
			expectedFlags: tFlags{address: "localhost:8080", logLevel: "info", port: 4317, peers: []string{"self"}},
		},
		{
			name:          "InheritedPersistentFlag_Filled",
			prefix:        "echo",
			env:           map[string]string{"ECHO_ADDRESS": "0.0.0.0:9090"},
			expectedFlags: tFlags{address: "0.0.0.0:9090", logLevel: "info", port: 4317, peers: []string{"self"}},
		},
		{
			name:          "SliceEnv_SplitOnCommas",
			prefix:        "foxtrot",
			env:           map[string]string{"FOXTROT_PEERS": "a:1,b:2"},
			expectedFlags: tFlags{address: "localhost:8080", logLevel: "info", port: 4317, peers: []string{"a:1", "b:2"}},
		},
		{
			name:          "PaddedMixedCasePrefix_Normalized",
			prefix:        " My_App2 ",
			env:           map[string]string{"MY_APP2_LOG_LEVEL": "debug"},
			expectedFlags: tFlags{address: "localhost:8080", logLevel: "debug", port: 4317, peers: []string{"self"}},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			for key, value := range testCase.env {
				t.Setenv(key, value)
			}

			flags, err := tExecute(t, testCase.prefix, testCase.args...)
			require.NoError(t, err)

			assert.Equal(t, testCase.expectedFlags, flags)
		})
	}
}

func TestParse_Error(t *testing.T) {
	testCases := []struct {
		name             string
		prefix           string
		env              map[string]string
		expectedCodes    []errors.Code
		expectedMessages []string
	}{
		{
			name:             "EmptyPrefix_Missing",
			prefix:           "  ",
			expectedCodes:    []errors.Code{CodePrefixMissing},
			expectedMessages: []string{"prefix is required"},
		},
		{
			name:             "DashedPrefix_Invalid",
			prefix:           "my-app",
			expectedCodes:    []errors.Code{CodePrefixInvalid},
			expectedMessages: []string{`prefix "my-app" is invalid, expected a letter followed by letters, digits or underscores`},
		},
		{
			name:             "DigitFirstPrefix_Invalid",
			prefix:           "9lives",
			expectedCodes:    []errors.Code{CodePrefixInvalid},
			expectedMessages: []string{`prefix "9lives" is invalid, expected a letter followed by letters, digits or underscores`},
		},
		{
			name:          "UnparsableEnvs_AllReportedWithoutValues",
			prefix:        "golf",
			env:           map[string]string{"GOLF_PORT": "s3cr3t-port", "GOLF_DEBUG": "s3cr3t-bool"},
			expectedCodes: []errors.Code{CodeEnvInvalid, CodeEnvInvalid},
			expectedMessages: []string{
				"GOLF_DEBUG is invalid for flag --debug, expected bool",
				"GOLF_PORT is invalid for flag --port, expected int",
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			for key, value := range testCase.env {
				t.Setenv(key, value)
			}

			_, err := tExecute(t, testCase.prefix)
			require.Error(t, err)

			assert.NotContains(t, err.Error(), "s3cr3t")

			var joined interface{ Unwrap() []error }

			errs := []error{err}
			if errors.As(err, &joined) {
				errs = joined.Unwrap()
			}

			require.Len(t, errs, len(testCase.expectedCodes))

			for i, err := range errs {
				assert.Equal(t, errors.TypeInvalidInput, errors.TypeOf(err))
				assert.Equal(t, testCase.expectedCodes[i], errors.CodeOf(err))
				assert.Equal(t, testCase.expectedMessages[i], errors.MessageOf(err))
			}
		})
	}
}
