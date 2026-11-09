---
paths:
  - "**/*.go"
---

# Comments and documentation

baseplate.go is a published library, and pkg.go.dev builds its documentation from these comments and the examples. Write them for a reader who has only the package page.

## Package documentation

- Every package has a `doc.go` that holds the package comment and nothing else. No other file has a package comment or opens with a file-level comment.
- The package comment starts with `Package <name>`, says what the package is for, then covers the concepts a caller needs (for `errors`: types, codes, wrapping, lookups, logging, HTTP) with a short usage sample.
- Use `# Heading` lines for sections, `[Name]` and `[pkg.Name]` doc links (`[TypeOf]`, `[log/slog.Attr]`), and indented blocks for code. No Markdown emphasis, tables or fenced code blocks, because pkg.go.dev doesn't render them.
- Describe in the package comment what pkg.go.dev doesn't list on its own, such as the methods of an unexported type that constructors return.

## Identifier documentation

- Every exported constant, variable, type, function and method has a doc comment, and so does every method a caller can reach on an unexported type (editors show it on hover). It is a full sentence that starts with the identifier's name: `// TypeOf returns the type of the first typed error in err's chain, or TypeUnknown when there is none.`
- Say what it does, what it returns and how it handles edge cases: `nil`, an untyped error, an empty value, a value that is copied rather than changed. Don't describe the implementation.
- A `var ( ... )` or `const ( ... )` block of related values gets one comment above the block. A value that needs explaining gets a one-line comment directly above it, never a list of values in the type's comment.
- A struct field's comment goes at the end of the field's line, never above it, and doesn't repeat the field's name: `Delay time.Duration // In nanoseconds.` A comment about several adjacent fields goes on the first of them and names them all.
- Mark replaced API with a paragraph starting `Deprecated:` that names the replacement.
- Unexported identifiers get a comment only for non-obvious behaviour, constraints, formats and edge cases, and never restate the code.

## Examples

- Show how a package is used with runnable examples in `example_test.go`, in the external `<pkg>_test` package so they read the way a caller writes them (`errors.New(...)`).
- Name them as pkg.go.dev expects: `Example` for the package, `ExampleNew` for a function, `ExampleNew_details` for a variant. Each ends with an `// Output:` comment, so `go test` checks it. Keep the output deterministic: no stack traces, times or map iteration order.
- Use examples to show chaining and the methods of unexported types.

## Everywhere

- Keep comments short, without narration, and end each one with a period (`godot` checks it). No end-of-line comments except on struct fields.
- Rationale that needs prose, such as why a version is pinned or why a design was chosen, goes in the PR, not the code.
- Directives aren't comments: keep `//go:embed`, `//go:generate` and the like. `//nolint` is never allowed ([go-checks.md](go-checks.md)).
- Generated code keeps the comments its generator emits; don't add or trim them.
