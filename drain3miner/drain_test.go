package drain3miner

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/SigNoz/baseplate.go/errors"
)

// The expected patterns and matches are what Drain3 0.9.11 returns for the
// same lines and settings, with match's "fallback" strategy.
func TestMinerProcess(t *testing.T) {
	const noMatch = "(no match)"

	testCases := []struct {
		name             string
		drain            func(*DrainConfig)
		lines            []string
		expectedPatterns []Pattern
		expectedMatches  []string
	}{
		{
			name: "Defaults_ValuesBecomeParams",
			lines: []string{
				"user alice logged in from 10.0.0.1",
				"user bob logged in from 10.0.0.2",
				"user carol logged out",
				"disk full on /var",
				"user dave logged out",
				"disk full on /tmp",
			},
			expectedPatterns: []Pattern{
				tPattern("user <*> logged in from <*>", 2),
				tPattern("user <*> logged out", 2),
				tPattern("disk full on <*>", 2),
			},
			expectedMatches: []string{"user <*> logged in from <*>", "user <*> logged in from <*>", "user <*> logged out", "disk full on <*>", "user <*> logged out", "disk full on <*>"},
		},
		{
			name:  "NumericFirstToken_SharesParamNode",
			lines: []string{"12 apples sold", "13 apples sold"},
			expectedPatterns: []Pattern{
				tPattern("<*> apples sold", 2),
			},
			expectedMatches: []string{"<*> apples sold", "<*> apples sold"},
		},
		{
			name:  "KeepNumericTokens_RoutedApart",
			drain: func(config *DrainConfig) { config.KeepNumericTokens = true },
			lines: []string{"12 apples sold", "13 apples sold"},
			expectedPatterns: []Pattern{
				tPattern("12 apples sold", 1),
				tPattern("13 apples sold", 1),
			},
			expectedMatches: []string{"12 apples sold", "13 apples sold"},
		},
		{
			name:  "HighThreshold_KeepsPathsAndStatuses",
			drain: func(config *DrainConfig) { config.SimThreshold = 0.9 },
			lines: []string{"GET /api/users 200", "GET /api/orders 200", "GET /api/users 500"},
			expectedPatterns: []Pattern{
				tPattern("GET /api/users 200", 1),
				tPattern("GET /api/orders 200", 1),
				tPattern("GET /api/users 500", 1),
			},
			expectedMatches: []string{"GET /api/users 200", "GET /api/orders 200", "GET /api/users 500"},
		},
		{
			name:  "ExtraDelimiters_SplitTokens",
			drain: func(config *DrainConfig) { config.ExtraDelimiters = []string{"=", ","} },
			lines: []string{"level=info msg=started,port=80", "level=warn msg=stopped,port=81"},
			expectedPatterns: []Pattern{
				tPattern("level <*> msg <*> port <*>", 2),
			},
			expectedMatches: []string{"level <*> msg <*> port <*>", "level <*> msg <*> port <*>"},
		},
		{
			name:  "MaxClusters_LeastRecentlyUsedDropped",
			drain: func(config *DrainConfig) { config.MaxClusters = 2 },
			lines: []string{"alpha one", "beta two three", "gamma four five six", "alpha one"},
			expectedPatterns: []Pattern{
				tPattern("gamma four five six", 1),
				tPattern("alpha one", 1),
			},
			expectedMatches: []string{"alpha one", noMatch, "gamma four five six", "alpha one"},
		},
		{
			name:  "MaxChildren_LaterTokensShareParamNode",
			drain: func(config *DrainConfig) { config.MaxChildren = 2 },
			lines: []string{"a x y", "b x y", "c x y", "d x y"},
			expectedPatterns: []Pattern{
				tPattern("a x y", 1),
				tPattern("<*> x y", 3),
			},
			expectedMatches: []string{"a x y", "<*> x y", "<*> x y", "<*> x y"},
		},
		{
			name:  "EmptyAndBlankLines_OnePattern",
			lines: []string{"", " \t ", "x"},
			expectedPatterns: []Pattern{
				tPattern("", 2),
				tPattern("x", 1),
			},
			expectedMatches: []string{"", "", "x"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			miner := tNewMiner(t, testCase.drain)
			for _, line := range testCase.lines {
				miner.Process(line)
			}

			assert.Equal(t, testCase.expectedPatterns, miner.Patterns())

			matches := make([]string, 0, len(testCase.lines))
			for _, line := range testCase.lines {
				pattern, ok := miner.Match(line)
				if !ok {
					assert.Equal(t, Pattern{}, pattern)
					matches = append(matches, noMatch)

					continue
				}

				matches = append(matches, pattern.Template)
			}

			assert.Equal(t, testCase.expectedMatches, matches)
		})
	}
}

func TestMinerMatch_UnknownLine_NotFound(t *testing.T) {
	miner := tNewMiner(t, nil)
	miner.Process("cache warmed in 20ms")

	pattern, ok := miner.Match("cache cleared by admin")

	assert.False(t, ok)
	assert.Equal(t, Pattern{}, pattern)
	assert.Equal(t, []Pattern{
		tPattern("cache warmed in 20ms", 1),
	}, miner.Patterns())
}

// A miner loading another's state midway ends where the other would have.
func TestMinerLoadState_ContinuesStream(t *testing.T) {
	testCases := []struct {
		name  string
		drain func(*DrainConfig)
	}{
		{name: "Defaults"},
		{
			name:  "MaxClusters",
			drain: func(config *DrainConfig) { config.MaxClusters = 3 },
		},
		{
			name:  "MaxChildren",
			drain: func(config *DrainConfig) { config.MaxChildren = 2 },
		},
	}

	lines := []string{
		"conn 1 opened from 10.1.0.4",
		"a x y",
		"conn 2 opened from 10.1.0.9",
		"b x y",
		"queue depth 40",
		"c x y",
		"conn 3 closed",
		"queue depth 41",
		"d x y",
		"conn 4 closed",
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			whole := tNewMiner(t, testCase.drain)

			var wholeResults []tResult
			for _, line := range lines {
				pattern, change := whole.Process(line)
				wholeResults = append(wholeResults, tResult{pattern, change})
			}

			first := tNewMiner(t, testCase.drain)

			var results []tResult
			for _, line := range lines[:5] {
				pattern, change := first.Process(line)
				results = append(results, tResult{pattern, change})
			}

			state, err := first.State()
			require.NoError(t, err)

			second := tNewMiner(t, testCase.drain)
			require.NoError(t, second.LoadState(state))

			for _, line := range lines[5:] {
				pattern, change := second.Process(line)
				results = append(results, tResult{pattern, change})
			}

			assert.Equal(t, wholeResults, results)
			assert.Equal(t, whole.Patterns(), second.Patterns())

			for _, line := range lines {
				expected, expectedOK := whole.Match(line)
				pattern, ok := second.Match(line)
				assert.Equal(t, expectedOK, ok)
				assert.Equal(t, expected, pattern)
			}
		})
	}
}

func TestMinerLoadState_LowerMaxClusters_DropsLeastRecent(t *testing.T) {
	miner := tNewMiner(t, nil)
	for _, line := range []string{"alpha one", "beta two three", "gamma four five six"} {
		miner.Process(line)
	}

	state, err := miner.State()
	require.NoError(t, err)

	smaller := tNewMiner(t, func(config *DrainConfig) { config.MaxClusters = 2 })
	require.NoError(t, smaller.LoadState(state))

	assert.Equal(t, []Pattern{
		tPattern("beta two three", 1),
		tPattern("gamma four five six", 1),
	}, smaller.Patterns())
}

func TestMinerLoadState_Invalid(t *testing.T) {
	testCases := []struct {
		name  string
		drain func(*DrainConfig)
		edit  func(state *drainState)
		raw   string
	}{
		{
			name: "NotJSON",
			raw:  "{",
		},
		{
			name: "OtherVersion",
			edit: func(state *drainState) { state.Version = 2 },
		},
		{
			name:  "OtherDepth",
			drain: func(config *DrainConfig) { config.Depth = 5 },
		},
		{
			name:  "OtherParamString",
			drain: func(config *DrainConfig) { config.ParamString = "<?>" },
		},
		{
			name: "DuplicateCluster",
			edit: func(state *drainState) { state.Clusters = append(state.Clusters, state.Clusters[0]) },
		},
		{
			name: "ClusterAboveLastID",
			edit: func(state *drainState) { state.LastID = 0 },
		},
		{
			name: "TokenWithSpace",
			edit: func(state *drainState) { state.Clusters[0].Tokens = []string{"two words", "x"} },
		},
		{
			name: "TokenCountMismatch",
			edit: func(state *drainState) { state.Clusters[0].Tokens = []string{"one"} },
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			source := tNewMiner(t, nil)
			source.Process("worker 7 started")
			source.Process("worker 8 started")

			state, err := source.State()
			require.NoError(t, err)

			if testCase.edit != nil {
				var decoded drainState
				require.NoError(t, json.Unmarshal(state, &decoded))
				testCase.edit(&decoded)

				state, err = json.Marshal(decoded)
				require.NoError(t, err)
			}

			if testCase.raw != "" {
				state = []byte(testCase.raw)
			}

			miner := tNewMiner(t, testCase.drain)
			miner.Process("disk full on /var")
			before := miner.Patterns()

			err = miner.LoadState(state)

			require.Error(t, err)
			assert.True(t, errors.Ast(err, errors.TypeInvalidInput))
			assert.True(t, errors.Asc(err, CodeStateInvalid))
			assert.Equal(t, before, miner.Patterns())
		})
	}
}

// The IDs are 64-bit FNV-1a hashes computed independently of Go.
func TestHashTemplate_FNV1a(t *testing.T) {
	testCases := []struct {
		template string
		expected uint64
	}{
		{template: "", expected: 14695981039346656037},
		{template: "x", expected: 12638214688346347271},
		{template: "user <*> logged in from <*>", expected: 7157027767056440977},
		{template: "disk full on <*>", expected: 4923888337447630434},
		{template: "GET /api/users 200", expected: 8168002615984564967},
		{template: "ünïcödé <*> 日本語", expected: 1061945747591625026},
	}

	for _, testCase := range testCases {
		t.Run(testCase.template, func(t *testing.T) {
			assert.Equal(t, testCase.expected, hashTemplate(testCase.template))
		})
	}
}

func TestMinerProcess_Changes(t *testing.T) {
	testCases := []struct {
		name            string
		drain           func(*DrainConfig)
		lines           []string
		expectedChanges []*Change
	}{
		{
			name:  "NewThenWidenedThenSame",
			lines: []string{"user alice logged in", "user bob logged in", "user carol logged in"},
			expectedChanges: []*Change{
				tChange("", "user alice logged in"),
				tChange("user alice logged in", "user <*> logged in"),
				nil,
			},
		},
		{
			name:  "WidenedTwice_EachStepOneChange",
			lines: []string{"job a done b", "job c done b", "job d done e"},
			expectedChanges: []*Change{
				tChange("", "job a done b"),
				tChange("job a done b", "job <*> done b"),
				tChange("job <*> done b", "job <*> done <*>"),
			},
		},
		{
			name:  "DroppedPattern_CreatedAgain",
			drain: func(config *DrainConfig) { config.MaxClusters = 1 },
			lines: []string{"alpha one", "beta two three", "alpha one"},
			expectedChanges: []*Change{
				tChange("", "alpha one"),
				tChange("", "beta two three"),
				tChange("", "alpha one"),
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			miner := tNewMiner(t, testCase.drain)

			changes := make([]*Change, 0, len(testCase.lines))
			for _, line := range testCase.lines {
				pattern, change := miner.Process(line)
				if change != nil {
					assert.Equal(t, change.To, pattern.ID)
				}

				changes = append(changes, change)
			}

			assert.Equal(t, testCase.expectedChanges, changes)
		})
	}
}

// ProcessBatch gives every line the pattern, count and change Process gives
// it, whatever the batch size, and leaves the same state.
func TestMinerProcessBatch_SameAsProcess(t *testing.T) {
	lines := tBatchLines()

	testCases := []struct {
		name      string
		drain     func(*DrainConfig)
		batchSize int
	}{
		{
			name:      "Defaults_BatchOfOne",
			batchSize: 1,
		},
		{
			name:      "Defaults_BatchOfSeven",
			batchSize: 7,
		},
		{
			name:      "Defaults_OneBatch",
			batchSize: len(lines),
		},
		{
			name:      "MaxClusters_DropsInBatch",
			drain:     func(config *DrainConfig) { config.MaxClusters = 3 },
			batchSize: 5,
		},
		{
			name:      "KeepNumericTokens_BatchOfThree",
			drain:     func(config *DrainConfig) { config.KeepNumericTokens = true },
			batchSize: 3,
		},
		{
			name:      "HighThreshold_BatchOfFifty",
			drain:     func(config *DrainConfig) { config.SimThreshold = 0.9 },
			batchSize: 50,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			single := tNewMiner(t, testCase.drain)
			batched := tNewMiner(t, testCase.drain)

			expected := make([]tResult, 0, len(lines))
			for _, line := range lines {
				pattern, change := single.Process(line)
				expected = append(expected, tResult{pattern: pattern, change: change})
			}

			actual := make([]tResult, 0, len(lines))
			for start := 0; start < len(lines); start += testCase.batchSize {
				patterns, changes := batched.ProcessBatch(lines[start:min(start+testCase.batchSize, len(lines))])
				require.Len(t, changes, len(patterns))

				for i := range patterns {
					actual = append(actual, tResult{pattern: patterns[i], change: changes[i]})
				}
			}

			assert.Equal(t, expected, actual)
			assert.Equal(t, single.Patterns(), batched.Patterns())

			expectedState, err := single.State()
			require.NoError(t, err)

			actualState, err := batched.State()
			require.NoError(t, err)

			assert.JSONEq(t, string(expectedState), string(actualState))
		})
	}
}

// A line repeated after its pattern widened and was dropped creates the
// pattern from its own tokens again, not from the widened template.
func TestMinerProcessBatch_WidenedThenDropped_RecreatedFromLine(t *testing.T) {
	miner := tNewMiner(t, func(config *DrainConfig) { config.MaxClusters = 1 })

	patterns, changes := miner.ProcessBatch([]string{"queue a drained", "queue b drained", "worker idle now ok", "queue a drained"})

	assert.Equal(t, []Pattern{
		tPattern("queue a drained", 1),
		tPattern("queue <*> drained", 2),
		tPattern("worker idle now ok", 1),
		tPattern("queue a drained", 1),
	}, patterns)
	assert.Equal(t, []*Change{
		tChange("", "queue a drained"),
		tChange("queue a drained", "queue <*> drained"),
		tChange("", "worker idle now ok"),
		tChange("", "queue a drained"),
	}, changes)
}

func TestMinerProcessBatch_NoLines_Nil(t *testing.T) {
	miner := tNewMiner(t, nil)

	patterns, changes := miner.ProcessBatch(nil)

	assert.Nil(t, patterns)
	assert.Nil(t, changes)
	assert.Empty(t, miner.Patterns())
}

// tBatchLines returns lines that repeat, create patterns, widen them in the
// middle of a run of repeats, and include empty and whitespace-only lines.
func tBatchLines() []string {
	lines := []string{"", "   "}
	for i := range 60 {
		lines = append(lines,
			fmt.Sprintf("user u%d logged in from 10.0.0.%d", i%4, i%3),
			"GET /health 200",
			"GET /health 200",
			fmt.Sprintf("job %c finished step %d", 'a'+rune(i%5), i%2),
			fmt.Sprintf("cache %s miss key k%d", []string{"hit", "miss", "warm"}[i%3], i),
			fmt.Sprintf("shard %d of %d ready", i%7, 8),
		)
		if i%10 == 9 {
			lines = append(lines, fmt.Sprintf("rare event number %d at node n%d", i, i/10), "")
		}
	}

	return lines
}

type tResult struct {
	pattern Pattern
	change  *Change
}

func tPattern(template string, count int64) Pattern {
	return Pattern{
		ID:       hashTemplate(template),
		Template: template,
		Count:    count,
	}
}

// tChange returns the change from template from to template to; an empty
// from stands for a new pattern.
func tChange(from, to string) *Change {
	change := &Change{To: hashTemplate(to)}
	if from != "" {
		change.From = hashTemplate(from)
	}

	return change
}

// Lines processed from many goroutines, while others read, all land in
// patterns; go test -race checks the locking.
func TestMiner_ConcurrentUse(t *testing.T) {
	miner := tNewMiner(t, nil)

	var wg sync.WaitGroup
	for worker := range 8 {
		wg.Go(func() {
			for i := range 200 {
				miner.Process(fmt.Sprintf("worker %d step %d done", worker, i))
				_, _ = miner.ProcessBatch([]string{
					fmt.Sprintf("worker %d step %d done", worker, i),
					"worker 0 step 0 done",
				})
				_, _ = miner.Match("worker 1 step 2 done")
			}
		})
		wg.Go(func() {
			for range 50 {
				_ = miner.Patterns()
				_, err := miner.State()
				assert.NoError(t, err)
			}
		})
	}

	wg.Wait()

	assert.Equal(t, []Pattern{tPattern("worker <*> step <*> done", 4800)}, miner.Patterns())
}

// Many goroutines creating patterns of several token counts under a small
// MaxClusters drop each other's patterns; go test -race checks the
// locking, and the miner keeps at most MaxClusters.
func TestMiner_ConcurrentUse_MaxClustersAcrossTokenCounts(t *testing.T) {
	miner := tNewMiner(t, func(config *DrainConfig) { config.MaxClusters = 4 })

	var wg sync.WaitGroup
	for worker := range 8 {
		wg.Go(func() {
			for i := range 300 {
				words := strings.Repeat(" w", i%6)
				miner.Process(fmt.Sprintf("event%d kind%d%s", worker, i%5, words))
				_, _ = miner.ProcessBatch([]string{
					fmt.Sprintf("batch%d item%d%s", worker, i%3, words),
					"steady line here",
				})
			}
		})
		wg.Go(func() {
			for range 50 {
				assert.LessOrEqual(t, len(miner.Patterns()), 4)
				_, err := miner.State()
				assert.NoError(t, err)
			}
		})
	}

	wg.Wait()

	assert.LessOrEqual(t, len(miner.Patterns()), 4)
}

// A cluster dropped for MaxClusters is deleted from its group even when
// no other line of its token count comes.
func TestMiner_MaxClusters_DroppedClustersDeleted(t *testing.T) {
	miner := tNewMiner(t, func(config *DrainConfig) { config.MaxClusters = 1 })

	for i := range 20 {
		_, _ = miner.Process(strings.TrimSpace(strings.Repeat("word ", i+1)))
	}

	d, ok := miner.(*drain)
	require.True(t, ok)

	held := 0
	for _, g := range *d.forest.Load().groups.Load() {
		held += len(g.clusters)
	}

	assert.Equal(t, 1, held)
}

// With MaxClusters 1, a new token count's pattern replaces the kept one;
// a concurrent read sees exactly one pattern, never the dropped one
// without its replacement.
func TestMiner_ConcurrentUse_ReadsSeeReplacement(t *testing.T) {
	miner := tNewMiner(t, func(config *DrainConfig) { config.MaxClusters = 1 })
	miner.Process("first")

	var wg sync.WaitGroup
	for worker := range 4 {
		wg.Go(func() {
			for i := range 200 {
				_, _ = miner.Process(strings.TrimSpace(strings.Repeat(fmt.Sprintf("w%d ", worker), 1+(i*4+worker)%40)))
			}
		})
	}

	wg.Go(func() {
		for range 500 {
			assert.Len(t, miner.Patterns(), 1)
		}
	})

	wg.Wait()
}

func tNewMiner(t *testing.T, edit func(*DrainConfig)) Miner {
	t.Helper()

	config := DefaultConfig()
	if edit != nil {
		edit(&config.Drain)
	}

	miner, err := New(config)
	require.NoError(t, err)

	return miner
}
