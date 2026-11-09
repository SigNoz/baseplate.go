// Package cobraenv fills the flags of a [github.com/spf13/cobra.Command]
// from environment variables.
//
// [Parse] reads one variable per flag, named after the prefix and the flag:
// the prefix and the flag name joined by an underscore, upper-cased, with
// dashes turned into underscores. For the prefix "myapp", --log-level reads
// MYAPP_LOG_LEVEL and --http-address reads MYAPP_HTTP_ADDRESS.
//
// # Precedence
//
// A flag passed on the command line wins over its variable, and a variable
// wins over the flag's default. A variable set to the empty string counts as
// unset. A variable is parsed the way the flag parses a command-line value,
// so MYAPP_PEERS=a,b fills a string slice flag with two values.
//
// # Usage
//
// Call [Parse] from the root command's PersistentPreRunE. Cobra has parsed
// the command line and merged the inherited persistent flags into the
// command's flags by then, so every flag of the command being run is filled:
//
//	root := &cobra.Command{
//		Use: "myapp",
//		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
//			return cobraenv.Parse(cmd, "myapp")
//		},
//	}
//
// # Errors
//
// [Parse] returns a [github.com/SigNoz/baseplate.go/errors.TypeInvalidInput]
// error with [CodePrefixMissing] or [CodePrefixInvalid] for a bad prefix,
// and one with [CodeEnvInvalid] for each variable its flag can't parse,
// joined together. Messages name the variable, the flag and the expected
// type, never the value, because a variable may hold a secret.
package cobraenv
