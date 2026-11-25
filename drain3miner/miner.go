package drain3miner

// Miner groups lines of text into patterns. [New] returns one for a [Config].
// It is safe for concurrent use.
type Miner interface {
	// Process puts line into a pattern, creating a pattern or widening its
	// template as needed, and returns that pattern. It returns a non-nil
	// [Change] when the pattern was created or its template widened, and
	// with it its ID.
	Process(line string) (Pattern, *Change)
	// Match returns the pattern whose template matches line, with every
	// parameter standing for one token, and false when none does. It
	// changes no pattern.
	Match(line string) (Pattern, bool)
	// Patterns returns the patterns kept, in the order they were created.
	Patterns() []Pattern
	// State returns the miner's patterns and what it learned about them,
	// to restore later with LoadState. It holds no settings. It returns a
	// TypeInternal error with [CodeStateEncodeFailed] when encoding fails.
	State() ([]byte, error)
	// LoadState replaces the miner's patterns with those in state, keeping
	// the miner's settings. It returns a TypeInvalidInput error with
	// [CodeStateInvalid] for state that doesn't parse or was saved with
	// settings that shape patterns differently, and leaves the miner
	// unchanged then.
	LoadState(state []byte) error
}

// Pattern is a group of lines and the template they share.
type Pattern struct {
	ID       uint64 // 64-bit FNV-1a hash of Template.
	Template string // Tokens joined by single spaces, values as the param string.
	Count    int64  // Lines processed into the pattern.
}

// Change is a pattern ID that [Miner.Process] created or replaced: From is
// the ID before the line, 0 for a new pattern, and To the ID after it.
type Change struct {
	From uint64
	To   uint64
}

// New returns a [Miner] for config. It returns a TypeInvalidInput error with
// [CodeModeInvalid] for a mode that isn't declared and [CodeConfigInvalid]
// for a setting out of range.
func New(config Config) (Miner, error) {
	if err := config.validate(); err != nil {
		return nil, err
	}

	return newDrain(config.Drain), nil
}
