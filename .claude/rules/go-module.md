---
paths:
  - "**/*.go"
  - "go.mod"
  - ".github/workflows/*"
---

# Module and packages

- baseplate.go is one module, `github.com/SigNoz/baseplate.go`. Each package sits at the module root (`errors/`, `cobraenv/`), with no `pkg/` or `internal/` prefix, because other SigNoz repos import them.
- The `go` directive is the latest Go minor version with no patch (`go 1.27`), so any patch release builds the module. Raise it, and `GO_VERSION` in `.github/workflows/ci.yml`, when a new minor version is released.
- Releases are semver tags `v0.x.y`. Consumers require a tag; they never use `replace` or a committed `go.work`.
- A package depends only on the standard library, other baseplate packages and the libraries it can't do without. A consumer importing one package must not need anything another package uses.
- Agree a package's names and API before writing it.
