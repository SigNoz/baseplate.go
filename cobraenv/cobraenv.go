package cobraenv

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/SigNoz/baseplate.go/errors"
)

// The codes of the errors [Parse] returns.
var (
	CodePrefixMissing = errors.MustNewCode("prefix_missing")
	CodePrefixInvalid = errors.MustNewCode("prefix_invalid")
	CodeEnvInvalid    = errors.MustNewCode("env_invalid")
)

// Parse fills every flag of cmd that wasn't passed on the command line from
// its environment variable, named after prefix and the flag: --log-level
// reads ZEUS_LOG_LEVEL for the prefix "zeus". Flags passed on the command
// line and variables that are unset or empty are left alone.
//
// It returns an error when prefix is empty or isn't a letter followed by
// letters, digits or underscores, and otherwise one error per variable that
// its flag can't parse, joined. It returns nil when every variable parses.
func Parse(cmd *cobra.Command, prefix string) error {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return errors.New(errors.TypeInvalidInput, CodePrefixMissing, "prefix is required")
	}

	if !validPrefix(prefix) {
		return errors.Newf(errors.TypeInvalidInput, CodePrefixInvalid, "prefix %q is invalid, expected a letter followed by letters, digits or underscores", prefix)
	}

	flags := cmd.Flags()

	var errs []error

	flags.VisitAll(func(flag *pflag.Flag) {
		if flag.Changed {
			return
		}

		name := envName(prefix, flag.Name)

		value, ok := os.LookupEnv(name)
		if !ok || value == "" {
			return
		}

		// The error from Set quotes the value, which may be a secret, so it
		// is dropped rather than wrapped.
		if err := flags.Set(flag.Name, value); err != nil {
			errs = append(errs, errors.Newf(errors.TypeInvalidInput, CodeEnvInvalid, "%s is invalid for flag --%s, expected %s", name, flag.Name, flag.Value.Type()))
		}
	})

	return errors.Join(errs...)
}

func envName(prefix, flag string) string {
	return strings.ToUpper(strings.ReplaceAll(prefix+"_"+flag, "-", "_"))
}

func validPrefix(prefix string) bool {
	for i, r := range prefix {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z':
		case i > 0 && (r >= '0' && r <= '9' || r == '_'):
		default:
			return false
		}
	}

	return true
}
