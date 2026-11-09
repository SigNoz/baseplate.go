---
paths:
  - "**/*_test.go"
---

# Tests

- Use testify (`assert`, `require`) and table-driven cases. Tests live next to the source file.
- Use `require` for anything the rest of the test can't proceed without: setup, `require.NoError(t, err)`, nil and length checks before indexing or dereferencing. Use `assert` for the expectations themselves, so one failed check still reports the rest.
- Declare cases as `testCases := []struct{ name string; ... }` and iterate with `for _, testCase := range testCases { t.Run(testCase.name, ...) }`. Case names are PascalCase segments joined by `_`, one per aspect (scenario, condition, expectation): `ActiveRun_DuplicateRequest_Deduplicated`, `UnknownProvider_ExpectedOneOf`.
- Fields say `expected`, never `want`: `expectedRun`, `expectedErrorType`.
- Every test helper struct and function starts with `t`, so it can't be confused with real code: `tNewRun`, `tRequests`.
- Don't hoist shared test constants. Vary fixture strings across cases instead.
- Generate mocks with mockery: list the interface in `.mockery.yml` and run mockery. Never hand-write mocks. Generated mocks live in the source package's `<pkg>test` sibling (`harnesstest.NewMockHarness(t)`).
- Don't write tests for thin wrappers around another library, such as a function that only calls the standard library, and don't generate mocks only for them. Verify them by building and linting them. Test the real logic.
- Run tests with `-race`.
