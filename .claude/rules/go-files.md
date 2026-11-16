# File layout

Order a file so it reads top down, from what a caller uses to what the code uses:

1. Constants and variables.
2. The type, then any type that only it uses.
3. Its constructors, exported first.
4. Its methods, exported first, then unexported.
5. Package-level helpers, last. This includes an unexported function that the constructors share, such as `compile` behind `Compile` and `CompileValue`.

A file with no type holds its functions with the exported ones first, then the helpers they call.

Every method of a type goes in the type's file. Don't put a method in another file because of what it deals with, such as `positionError` in `errors.go`.

```go
type source struct { ... }

type respelledToken struct { ... }

func newSource(text string) (*source, error) { ... }

func (s *source) offset(up int, end bool) int { ... }

func (s *source) positionError(err error, code errors.Code) error { ... }

func upstreamSpelling(tokens []lexer.Token, i int) string { ... }
```
