package drain3miner_test

import (
	"fmt"

	"github.com/SigNoz/baseplate.go/drain3miner"
)

func Example() {
	miner, err := drain3miner.New(drain3miner.DefaultConfig())
	if err != nil {
		_, _ = fmt.Println(err)

		return
	}

	for _, line := range []string{
		"disk full on /var",
		"disk full on /tmp",
		"disk full on /home",
	} {
		pattern, change := miner.Process(line)
		if change != nil {
			_, _ = fmt.Println(pattern.Template, "changed", change.From, "->", change.To)

			continue
		}

		_, _ = fmt.Println(pattern.Template, "unchanged", pattern.ID)
	}

	pattern, ok := miner.Match("disk full on /opt")
	_, _ = fmt.Println(ok, pattern.ID, pattern.Count)
	// Output:
	// disk full on /var changed 0 -> 874277695240487156
	// disk full on <*> changed 874277695240487156 -> 4923888337447630434
	// disk full on <*> unchanged 4923888337447630434
	// true 4923888337447630434 3
}

// A batch, such as the bodies of one request's logs, returns each line's
// pattern and change at the line's index.
func ExampleMiner_ProcessBatch() {
	miner, err := drain3miner.New(drain3miner.DefaultConfig())
	if err != nil {
		_, _ = fmt.Println(err)

		return
	}

	patterns, changes := miner.ProcessBatch([]string{
		"disk full on /var",
		"disk full on /var",
		"disk full on /tmp",
		"disk full on /var",
	})

	for i, pattern := range patterns {
		_, _ = fmt.Println(pattern.Template, pattern.Count, changes[i] != nil)
	}
	// Output:
	// disk full on /var 1 true
	// disk full on /var 2 false
	// disk full on <*> 3 true
	// disk full on <*> 4 false
}

// A miner restarted from its saved state goes on with the same patterns.
func ExampleMiner_loadState() {
	config := drain3miner.DefaultConfig()

	miner, err := drain3miner.New(config)
	if err != nil {
		_, _ = fmt.Println(err)

		return
	}

	miner.Process("job 41 finished in 3s")

	state, err := miner.State()
	if err != nil {
		_, _ = fmt.Println(err)

		return
	}

	restarted, err := drain3miner.New(config)
	if err != nil {
		_, _ = fmt.Println(err)

		return
	}

	if err := restarted.LoadState(state); err != nil {
		_, _ = fmt.Println(err)

		return
	}

	restarted.Process("job 42 finished in 5s")

	for _, pattern := range restarted.Patterns() {
		_, _ = fmt.Println(pattern.ID, pattern.Count, pattern.Template)
	}
	// Output:
	// 11991422607704262509 2 job <*> finished in <*>
}
