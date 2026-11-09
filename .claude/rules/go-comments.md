---
paths:
  - "**/*.go"
---

# Comments

Comment only non-obvious behavior, constraints, formats and edge cases, and never restate what the code says. Rationale that needs prose, such as why a version is pinned, why a job exists or how a subsystem fits together, goes in the README, `docs/` or the PR.

- No file-level comments: a file never opens with a banner or overview, and there is no package comment.
- No doc comments that only restate a declaration's name or type. A comment above a type, function, method, interface method, constant or variable is fine when it says something non-obvious about it. An enum value that needs explaining gets a one-line comment directly above it in the `var ( ... )` block, never a list of values in the type's comment.
- Keep comments to a line or two, without narration, and end each one with a period (`godot` checks it).
- A struct field's comment goes at the end of the field's line, never above it, and doesn't repeat the field's name: `Seq int64 // Orders the run's events, from 1 with no gaps.` A comment about several adjacent fields goes on the first of them and names them all: `Method string // Method, Kind, MessageID and ToolCallID are copied out of Payload when present, and empty otherwise.`
- No other end-of-line comments.
- Directives aren't comments: keep `//go:embed`, `//go:generate` and the like. `//nolint` is never allowed ([go-checks.md](go-checks.md)).
- Generated code keeps the comments its generator emits; don't add or trim them.
- Match the comment density of the surrounding file.
