---
paths:
  - "**/*.go"
---

# Enums

- Each enum-like type gets its own file, named after the type in snake case: `RunState` in `run_state.go`, `TriggerSource` in `trigger_source.go`. The file holds the values, the type, its constructor and its methods, and nothing else; the entity that uses the enum stays in its own file (`Run` in `run.go`).
- A value taken from an open list that another standard owns, such as OpenTelemetry's `gen_ai.provider.name`, is a plain `string` documented with that standard, not an enum: the list grows without us, and the code that uses the value rejects what it cannot serve.
- An error type in `errors` is the exception: a package declares its own with `errors.MustNewType` instead of picking from a fixed list, and `Type` has no `UnmarshalText`, because a type's name doesn't carry its HTTP status or retryability.
- Enum-like types are `struct{ s string }`, so code outside the package can only use the declared values. The values live in a `var ( ... )` block before the type and carry the full type name as prefix: `RunStateQueued`, not `Queued`.
- `New<Type>(s string) (<Type>, error)` parses a value and rejects anything undeclared with `errors.TypeInvalidInput` and the package's own code, naming the expected values. Compare values with `==`.
- Every enum has `String`, `MarshalText` and `UnmarshalText`, and `UnmarshalText` goes through the constructor. An enum stored in a database also has `Value` and `Scan`, and `Scan` goes through the constructor, so an unknown value is rejected when it is read.

```go
var (
	RunStateQueued  = RunState{"queued"}
	RunStateRunning = RunState{"running"}
)

type RunState struct{ s string }

func NewRunState(s string) (RunState, error) {
	switch s {
	case RunStateQueued.s, RunStateRunning.s:
		return RunState{s}, nil
	}

	return RunState{}, errors.Newf(errors.TypeInvalidInput, CodeRunStateInvalid, "run state %q is invalid, expected one of %s, %s", s, RunStateQueued, RunStateRunning)
}

func (state RunState) String() string {
	return state.s
}
```
