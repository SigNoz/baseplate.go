package cobraenv_test

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/SigNoz/baseplate.go/cobraenv"
)

// The command line sets --log-level, so MYAPP_LOG_LEVEL is ignored, while
// MYAPP_HTTP_ADDRESS fills --http-address.
func ExampleParse() {
	_ = os.Setenv("MYAPP_HTTP_ADDRESS", "0.0.0.0:9090")
	_ = os.Setenv("MYAPP_LOG_LEVEL", "debug")

	var address, logLevel string

	root := &cobra.Command{
		Use: "myapp",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			return cobraenv.Parse(cmd, "myapp")
		},
		RunE: func(*cobra.Command, []string) error {
			_, _ = fmt.Println(address, logLevel)

			return nil
		},
	}
	root.Flags().StringVar(&address, "http-address", "localhost:8080", "Address to listen on.")
	root.Flags().StringVar(&logLevel, "log-level", "info", "Log level.")
	root.SetArgs([]string{"--log-level", "warn"})

	if err := root.Execute(); err != nil {
		_, _ = fmt.Println(err)
	}
	// Output:
	// 0.0.0.0:9090 warn
}
