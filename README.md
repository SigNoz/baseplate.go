# baseplate.go
A baseplate for Go!

```sh
go get github.com/SigNoz/baseplate.go@latest
```

## errors

Typed errors with a code, a stack trace captured where the error is created, details for API responses, and HTTP status mapping.

To use it for every error in a repo, reject the standard `errors` package and `fmt.Errorf` in `.golangci.yml`:

```yaml
linters:
  enable:
    - depguard
    - forbidigo
  settings:
    depguard:
      rules:
        noerrors:
          files:
            - $all
          deny:
            - pkg: errors
              desc: Do not use errors package. Use github.com/SigNoz/baseplate.go/errors instead.
    forbidigo:
      forbid:
        - pattern: fmt.Errorf
          msg: Do not use fmt.Errorf. Use github.com/SigNoz/baseplate.go/errors instead.
```
