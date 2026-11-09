---
paths:
  - "**/*.go"
  - "go.mod"
---

# Checks and dependencies

- Run `make checks` before opening a PR. It runs primus `go-fmt`, `go-deps`, `go-lint` (golangci-lint with `bodyclose`, `depguard`, `errorlint`, `forbidigo`, `godot`, `modernize`, `nilerr` and `noctx` on top of the standard linters) and `go-test`, and all of them must pass. CI runs the same targets. `depguard` and `forbidigo` reject the standard `errors` package and `fmt.Errorf` everywhere but `errors/`; create and wrap errors with `github.com/SigNoz/baseplate.go/errors` ([go-errors.md](go-errors.md)). Fix what `modernize` reports (`slices.Contains`, `maps.Copy`, `wg.Go`, `new(expr)`) instead of excluding it.
- Never add `//nolint` anywhere in baseplate.go, and never exclude a linter or path in `.golangci.yml` to get past a finding. Change the code or the design until the linter passes. If that seems impossible, stop and raise it with the user.
- Run the checks through primus, not a bare `golangci-lint` or `go test`: the targets pin the versions and flags CI uses. They need `PRIMUS_HOME` set; check it before cloning primus again.
- Keep every module in `go.mod` on its latest version: after `go get`, run `go get -u -t ./...` and `go mod tidy`. When a latest version breaks the build, say so and decide the migration. Don't quietly hold the version back.
