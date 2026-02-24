---
name: go-practices
description:   Best practices for writing, building, testing, and shipping Go code. Use when working on Go projects, setting up CI/CD pipelines, reviewing Go code, or when the user mentions Go, Golang, golangci-lint, cobra, go modules, or Go testing.
allowed-tools: go, golangci-lint, cobra, go modules, go testing
metadata:
  author: renan-alm
  version: "1.0"
---

# Go Best Practices

## CI / CD Pipeline Order

Run steps in this order to fail fast and minimize wasted compute:

1. **Checkout** code
2. **Set up Go** — use `go-version-file: go.mod` so the version is defined in one place
3. **Download & verify dependencies** — `go mod tidy && go mod verify`
4. **Lint** — run `golangci-lint run ./...` _before_ build/test; it is cheaper and catches issues early
5. **Build** — `go build ./...`
6. **Test** — `go test -v -race ./...`

> **Tip:** When the prebuilt golangci-lint binary lags behind a new Go release, install from source
> (`go install github.com/golangci-lint/golangci-lint@latest`) so it is compiled with the
> same Go toolchain used by the project.

## Project Layout

- Use the Standard Go Project Layout conventions (`cmd/`, `internal/`, `pkg/`).
- Keep `main.go` minimal — it should only call an `Execute()` or `Run()` function from a package.
- Group related types, interfaces, and helpers into cohesive packages; avoid mega-packages.
- Use `internal/` for code that must not be imported by external consumers.

## Module & Dependency Management

- Always commit both `go.mod` and `go.sum`.
- Run `go mod tidy` before committing to remove unused dependencies.
- Pin direct dependencies to specific versions; let indirect deps float.
- Use `go mod verify` in CI to ensure module checksums match.

## Code Style & Formatting

- **Always** run `gofmt -s` (or `goimports`) — formatting is not optional in Go.
- Follow [Effective Go](https://go.dev/doc/effective_go) and the [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments) wiki.
- Exported names must have doc comments (`// FuncName does …`).
- Use `MixedCaps` / `mixedCaps`, never underscores in Go names.
- Keep functions short and focused; prefer early returns over deep nesting.
- Group imports in three blocks: stdlib, external, internal — `goimports` does this automatically.

## Error Handling

- **Always** check returned errors — never discard them with `_`.
- Wrap errors with context using `fmt.Errorf("doing X: %w", err)` so callers can unwrap.
- Define sentinel errors (`var ErrNotFound = errors.New("not found")`) or custom error types for programmatic inspection.
- Avoid `panic` in library code; reserve it for truly unrecoverable situations.
- Use `errors.Is` and `errors.As` instead of type assertions on errors.

## Testing

- Name test files `*_test.go` in the same package (white-box) or `*_test` package (black-box).
- Use table-driven tests for repetitive cases:

```go
tests := []struct {
    name  string
    input string
    want  string
}{ /* cases */ }
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) { /* assert */ })
}
```

- Use `t.Helper()` in test helper functions so failures report the caller's line.
- Run tests with `-race` in CI to detect data races.
- Use `t.Parallel()` where safe to speed up the test suite.
- Prefer the standard `testing` package; add `testify` only when assertion helpers provide clear value.
- Aim for meaningful coverage, not 100% — focus on business logic and edge cases.

## Concurrency

- Share memory by communicating (channels), don't communicate by sharing memory.
- Always use `sync.WaitGroup`, `sync.Mutex`, or channels to coordinate goroutines.
- Avoid goroutine leaks — ensure every goroutine has a clear exit path (e.g., context cancellation).
- Pass `context.Context` as the first parameter to functions that do I/O or may block.
- Use `select` with `ctx.Done()` to handle cancellation.

## Interfaces

- Define interfaces at the **consumer** side, not the producer side.
- Keep interfaces small — the bigger the interface, the weaker the abstraction.
- Accept interfaces, return structs.
- Use the standard `io.Reader`, `io.Writer`, `fmt.Stringer`, etc. when they fit.

## Logging

- Use structured logging (`log/slog` from stdlib, or `zerolog`/`zap`).
- Log at appropriate levels: `INFO` for progress/summaries, `DEBUG` for detailed operations, `WARN` for recoverable issues, `ERROR` for failures.
- Include contextual fields (request ID, resource name) rather than interpolating into message strings.

## Configuration

- Use a single source of truth for config (e.g., a YAML file parsed with `gopkg.in/yaml.v3`).
- Validate configuration eagerly at startup and fail fast with clear messages.
- Support backward compatibility for renamed config keys using fallback chains.
- Keep secrets out of config files — use environment variables or secret managers.

## Build & Release

- Use `-ldflags "-s -w -X main.version=$(VERSION)"` to embed version info and strip debug symbols.
- Cross-compile with `GOOS` and `GOARCH` environment variables.
- Use a `Makefile` with common targets: `build`, `test`, `lint`, `fmt`, `vet`, `tidy`, `clean`, `install`.
- Tag releases with semver (`vX.Y.Z`).

## Security

- Run `go vet ./...` in CI — it catches common bugs the compiler misses.
- Use `govulncheck` to scan for known vulnerabilities in dependencies.
- Sanitize all external input; never trust user-supplied data.
- Prefer `crypto/rand` over `math/rand` for security-sensitive randomness.

## Performance

- **Measure before optimizing** — use `go test -bench` and `pprof`.
- Pre-allocate slices and maps when the size is known (`make([]T, 0, n)`).
- Avoid unnecessary allocations in hot paths; reuse buffers where appropriate.
- Use `sync.Pool` for frequently allocated and discarded objects.
- Prefer `strings.Builder` over `+` concatenation in loops.

## Documentation

- Every exported symbol should have a doc comment starting with its name.
- Write package-level doc comments in a `doc.go` file for complex packages.
- Use `go doc` and `godoc` to verify documentation renders correctly.
- Include runnable examples (`func ExampleFuncName()`) to serve as both docs and tests.

allowed-tools: Bash(go:*) Bash(gofmt:*) Bash(golangci-lint:*) Bash(govulncheck:*) Read
