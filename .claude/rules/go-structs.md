# Structs

Declare and write structs one field per line.

## Declarations

Declare each field on its own line, even fields of the same type. A comment about several adjacent fields goes on the first of them and names them all ([go-comments.md](go-comments.md)).

```go
type rewrite struct {
	from   int    // From and to are offsets in the source.
	to     int
	upFrom int    // UpFrom and upTo are offsets in the upstream text.
	upTo   int
	text   string // Text and up are the token's spellings in the source and upstream.
	up     string
}
```

Not `from, to int`.

## Literals

Write a keyed struct literal with two or more fields one field per line, each ending in a comma. A literal with one field stays on one line.

```go
bound := (&file.Error{
	From: from,
	To:   to,
}).Bind(file.NewSource(s.text))

nilSafe := patcher.NilSafe{}
err := &tError{code: CodeRunFailed}
```

Don't group fields on a line, even pairs that belong together such as `from: f, to: t`.
