---
paths:
  - "**/*.go"
---

# Errors

Use `github.com/SigNoz/baseplate.go/errors` for every error. golangci-lint rejects the stdlib `errors` package (depguard) and `fmt.Errorf` (forbidigo); only `errors/` itself imports stdlib `errors`.

- Create errors with a type and a code: `errors.New(errors.TypeInvalidInput, CodePrefixMissing, "prefix is required")`, `errors.Newf(errors.TypeNotFound, CodeFlagNotFound, "no flag %s", name)`.
- Each package declares the codes of its errors in a `var` block with `errors.MustNewCode`, named for the failure: `CodePrefixMissing = errors.MustNewCode("prefix_missing")`. baseplate.go has no generic codes for callers; `errors.CodeUnknown` only reports an untyped error.
- Wrap with a type, a code and a short context, never `%w`: `errors.Wrapf(err, errors.TypeInternal, CodeEnvReadFailed, "read %s", path)`.
- Pick the type for what the caller can do about it: `TypeInvalidInput` for bad arguments, flags and input, `TypeNotFound` for something that doesn't exist, `TypeInternal` for a dependency that failed. When no base type fits, declare one with `errors.MustNewType(name, status, retryable)`, as SigNoz does for `license-unavailable`.
- Errors another layer already typed pass through unchanged, so callers can still branch on them (`errors.IsNotFound`).
- Messages say "expected", never "want": `code %q is invalid, expected a letter followed by letters, digits or underscores`.
- No sentinel error values (`var ErrX = errors.New(...)`): a value created at init has a useless stack trace. Return a fresh typed error and branch on the type or code: `errors.IsNotFound(err)`, `errors.Ast(err, errors.TypeForbidden)`, `errors.Asc(err, CodePrefixInvalid)`.
- `errors.Is`, `errors.As`, `errors.AsType` and `errors.Join` pass through to the stdlib, as in `errors.Is(err, fs.ErrNotExist)`.
- Never put a value that may be a secret, such as a flag or env value, in a message or a wrapped cause. Name the flag or variable instead.
- Check every error, including `Close`, `os.Remove` and writes. Skip only an expected outcome, and say why in a comment.
