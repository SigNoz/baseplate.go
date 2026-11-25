package drain3miner

import (
	"math"
	"strings"

	"github.com/SigNoz/baseplate.go/errors"
)

// The codes of the errors for a [Config].
var (
	// CodeModeInvalid is for a mode that isn't declared.
	CodeModeInvalid = errors.MustNewCode("mode_invalid")
	// CodeConfigInvalid is for a setting out of range.
	CodeConfigInvalid = errors.MustNewCode("config_invalid")
)

// Config holds the settings of a [Miner]. Start from [DefaultConfig] and
// change what differs.
type Config struct {
	Mode  Mode        // The algorithm, [ModeStream].
	Drain DrainConfig // Used by [ModeStream].
}

// DrainConfig holds the settings of Drain. The defaults from
// [DefaultConfig] are Drain3's.
type DrainConfig struct {
	Depth             int      // Depth of the search tree, at least 3; lines are routed by their first Depth-2 tokens. Default 4.
	SimThreshold      float64  // Share of equal tokens, from 0 to 1, a line needs to join a pattern. Default 0.4.
	MaxChildren       int      // Children per tree node, at least 1, before tokens share the parameter's node. Default 100.
	MaxClusters       int      // Patterns kept, least recently used dropped first; 0 keeps all. Default 0.
	ExtraDelimiters   []string // Non-empty strings that split tokens, besides whitespace. Default none.
	ParamString       string   // Token that stands for a value in templates, without whitespace. Default "<*>".
	KeepNumericTokens bool     // Route tokens with a digit like other tokens instead of as values. Default false.
}

// DefaultConfig returns the settings of a [ModeStream] miner with Drain3's
// defaults.
func DefaultConfig() Config {
	return Config{
		Mode: ModeStream,
		Drain: DrainConfig{
			Depth:        4,
			SimThreshold: 0.4,
			MaxChildren:  100,
			ParamString:  "<*>",
		},
	}
}

func (config Config) validate() error {
	if _, err := NewMode(config.Mode.s); err != nil {
		return err
	}

	return config.Drain.validate()
}

func (config DrainConfig) validate() error {
	switch {
	case config.Depth < 3:
		return errors.Newf(errors.TypeInvalidInput, CodeConfigInvalid, "drain depth %d is invalid, expected at least 3", config.Depth)
	case math.IsNaN(config.SimThreshold) || config.SimThreshold < 0 || config.SimThreshold > 1:
		return errors.Newf(errors.TypeInvalidInput, CodeConfigInvalid, "drain similarity threshold %v is invalid, expected a number from 0 to 1", config.SimThreshold)
	case config.MaxChildren < 1:
		return errors.Newf(errors.TypeInvalidInput, CodeConfigInvalid, "drain max children %d is invalid, expected at least 1", config.MaxChildren)
	case config.MaxClusters < 0:
		return errors.Newf(errors.TypeInvalidInput, CodeConfigInvalid, "drain max clusters %d is invalid, expected 0 or more", config.MaxClusters)
	case config.ParamString == "" || strings.ContainsFunc(config.ParamString, isSpace):
		return errors.Newf(errors.TypeInvalidInput, CodeConfigInvalid, "drain param string %q is invalid, expected a token without whitespace", config.ParamString)
	}

	for i, delimiter := range config.ExtraDelimiters {
		if delimiter == "" {
			return errors.Newf(errors.TypeInvalidInput, CodeConfigInvalid, "drain extra delimiter %d is empty, expected a non-empty string", i)
		}
	}

	return nil
}
