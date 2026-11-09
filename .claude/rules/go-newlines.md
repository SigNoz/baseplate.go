---
paths:
  - "**/*.go"
---

# Newlines before logical statements

Leave a blank line before each logical statement inside a function body: `if`, `for`, `switch`, `return`, `continue`, and a new local declaration that starts a fresh step. Adjacent lines that belong to the same step stay together, so each step reads as its own paragraph.

```go
func example() ([]Item, error) {
	cfg, err := load()
	if err != nil {
		return nil, err
	}

	items := make([]Item, 0, len(cfg.Raw))
	for _, r := range cfg.Raw {
		items = append(items, convert(r))
	}

	return items, nil
}
```

A call and its error check are the same step, so they stay adjacent and the blank line goes before the next step.
