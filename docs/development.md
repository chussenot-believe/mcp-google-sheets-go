# Development

## Summary

This page covers the source layout, the `mise` task interface, the test suite, and the CI pipeline. Use it when you are modifying the server.

## Source layout

| File | Responsibility |
| --- | --- |
| [`main.go`](../main.go) | CLI parsing, transport selection, server bootstrap. |
| [`auth.go`](../auth.go) | Four-tier credential resolution chain; OAuth interactive flow; token persistence. |
| [`tools.go`](../tools.go) | All 20 tool registrations and handlers. Includes argument-parsing helpers. |
| [`resources.go`](../resources.go) | `spreadsheet://{spreadsheet_id}/info` resource template. |
| [`helpers.go`](../helpers.go) | Pure helpers: Drive query escaping, A1 notation parsing, column letter conversion, chart range splitting. |
| [`helpers_test.go`](../helpers_test.go) | Unit tests for pure helpers and security-sensitive defaults. |
| [`mise.toml`](../mise.toml) | Pinned Go toolchain and task definitions. |
| [`.gitlab-ci.yml`](../.gitlab-ci.yml) | CI pipeline (mise install → `mise run ci` → optional Docker build). |
| [`Dockerfile`](../Dockerfile) | Multi-stage build → distroless image. |

There is no `internal/` split: the program is small enough to be a single `main` package.

## Toolchain

The Go version is pinned in `mise.toml` (`go = "1.23"`). `mise install` provisions it locally. CI uses the same pin.

## mise tasks

Run `mise tasks ls` to enumerate tasks. The canonical set:

| Task | Equivalent shell | Purpose |
| --- | --- | --- |
| `mise run build` | `go build -v ./...` | Build the binary. |
| `mise run test` | `go test -race -count=1 ./...` | Run tests under the race detector. |
| `mise run vet` | `go vet ./...` | Static analysis. |
| `mise run fmt-check` | `gofmt -l .` (exits non-zero if any file is not formatted) | Format check — CI gate. |
| `mise run fmt` | `gofmt -w .` | Format files in place. |
| `mise run tidy` | `go mod tidy` | Tidy module file after dep changes. |
| `mise run ci` | All of: `fmt-check`, `vet`, `build`, `test` | What CI runs. Run this before pushing. |

## Adding a new tool

1. Decide the tool's name (`lower_snake_case`).
2. Add a `registerXxx(s, d)` function in `tools.go`. Follow the pattern of existing tools:
   - Skip registration if `!d.isEnabled("name")`.
   - Build the tool with `mcp.NewTool("name", ...)`.
   - Implement the handler closure.
3. Call your `registerXxx(s, d)` from `registerAllTools`.
4. Document the tool in [`docs/tools.md`](tools.md) using the standard entry shape.
5. Add unit tests for any pure helpers extracted along the way.
6. Run `mise run ci`.

### Argument-parsing helpers

`tools.go` exposes:

- `argString(args, key)` — optional string, returns `""` if missing.
- `argStringRequired(args, key)` — required string, returns an error.
- `argBool(args, key, default)` — optional bool with default.
- `argInt(args, key, default)` — optional integer with default.
- `argIntPtr(args, key)` — optional integer that distinguishes "missing" from "zero".

Use these instead of bare type assertions — they handle the `float64` numbers that come out of JSON.

### Return helpers

- `jsonResult(v)` — marshal `v` as indented JSON and wrap in a text tool result.
- `errResult(format, args...)` — return an MCP error result.

## Tests

Currently `helpers_test.go` covers:

- `escapeDriveQuery` — including a classic injection payload.
- `columnIndexToLetter` and round-trip with `letterToColumnIndex`.
- `parseA1Notation` — full ranges, column-only ranges, invalid input.
- `domainAllowed` — case insensitivity and missing `@`.
- `parseEnabledTools` — whitespace, empty entries.
- `chooseValueInputOption` — default `RAW`, opt-in `USER_ENTERED`, unknown values fall back.

Add integration tests against a real Google service by tagging them with `// +build integration` so they remain opt-in.

## CI pipeline

Defined in [`.gitlab-ci.yml`](../.gitlab-ci.yml):

| Stage | Job | Action |
| --- | --- | --- |
| `check` | `ci` | Install mise, `mise install`, `mise run ci`. |
| `package` | `docker` | Build the Docker image. Runs on MRs and default-branch commits. |

The pipeline caches:

- `.mise/` and `.mise-cache/` — the provisioned Go toolchain.
- `.gopath/pkg/mod/` — the module download cache.
- `.gocache/` — the Go build cache.

Cache key is based on `mise.toml` and `go.sum` so toolchain or dependency changes invalidate the cache automatically.

## Releasing

There is no automated release workflow in this repo yet. Suggested procedure:

1. Update any version strings in code (`serverVersion` in `main.go`).
2. `mise run ci` locally.
3. Tag: `git tag v0.x.y && git push --tags`.
4. Build and push the image with that tag.

## Style

- `gofmt` is the format authority. `mise run fmt-check` enforces it.
- No comment-per-line on obvious code. Add a comment only when the *why* is non-obvious — e.g. the `RAW`-by-default decision in `chooseValueInputOption`.
- Imports are grouped by `goimports` conventions: stdlib, third-party, internal — separated by blank lines.
