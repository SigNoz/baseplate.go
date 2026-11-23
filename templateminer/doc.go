// Package templateminer groups lines of text, such as log messages, span
// names or SQL statements, into patterns: templates such as
// "user <*> logged in from <*>", where <*> stands for the tokens that vary
// between the lines of a pattern.
//
// # Modes
//
// [Config.Mode] picks the algorithm. [ModeStream] is the only mode: it
// mines lines one at a time with Drain (He et al., ICWS 2017), following
// Drain3 0.9.11, so each line has its pattern as soon as it is processed.
//
// # Usage
//
// [New] returns a [Miner]. [Miner.Process] puts a line into a pattern and
// returns it, [Miner.Match] finds a line's pattern without changing any,
// and [Miner.Patterns] lists the patterns so far:
//
//	miner, err := templateminer.New(templateminer.DefaultConfig())
//	if err != nil {
//		return err
//	}
//
//	for _, line := range lines {
//		pattern, change := miner.Process(line)
//		// Store pattern.ID with the line; record change when not nil.
//	}
//
//	for _, pattern := range miner.Patterns() {
//		// Use pattern.Template and pattern.Count.
//	}
//
// # Patterns
//
// A line is split into tokens on whitespace and on
// [DrainConfig.ExtraDelimiters]. Lines with different token counts never
// share a pattern. A line joins the pattern with the most equal tokens at
// the same positions, once at least [DrainConfig.SimThreshold] of the
// tokens are equal; the tokens that differ become
// [DrainConfig.ParamString]. A template never becomes more specific, and
// tokens with a digit are treated as values unless
// [DrainConfig.KeepNumericTokens] is set.
//
// A pattern's ID is the 64-bit FNV-1a hash of its template, so miners that
// reach the same template agree on its ID, across processes and restarts.
// When a template widens, its ID changes: [Miner.Process] returns a
// [Change] from the old ID to the new one, and a Change from 0 when it
// creates a pattern. Two patterns can share a template, and then
// an ID. With [DrainConfig.MaxClusters] set, the least recently used
// pattern is dropped when a new one would exceed it; a later line like it
// creates a new pattern.
//
// Lines are not masked: a caller that wants values such as IDs or
// timestamps replaced does so before processing the line.
//
// # State
//
// [Miner.State] saves what a miner learned, its patterns and how it routes
// lines to them, and [Miner.LoadState] restores it into another miner, for
// example after a restart. The state holds no settings: the loading miner
// keeps its own, and rejects state saved with another
// [DrainConfig.Depth] or [DrainConfig.ParamString]. A loading miner with
// a lower [DrainConfig.MaxClusters] drops the least recently used patterns
// beyond it. A miner without MaxClusters doesn't track use, so state it
// saves is in creation order and the earliest created patterns go first.
//
// # Concurrency
//
// A [Miner] is safe for concurrent use. [Miner.Process] calls for lines
// that fit a pattern as it is run together, as do [Miner.Match],
// [Miner.Patterns] and [Miner.State]. Creating a pattern, widening a
// template and [Miner.LoadState] take the miner's lock one call at a time.
//
// # Errors
//
// [New] returns a [github.com/SigNoz/baseplate.go/errors.TypeInvalidInput]
// error with [CodeModeInvalid] for an unknown mode and [CodeConfigInvalid]
// for a Drain setting out of range, [NewMode] one with [CodeModeInvalid],
// and [Miner.LoadState] one with [CodeStateInvalid].
package templateminer
