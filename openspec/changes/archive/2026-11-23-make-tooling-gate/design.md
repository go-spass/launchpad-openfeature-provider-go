# Design

## Context

See `proposal.md` — Why. The constraints that shape the approach:

- The module is empty: `go.mod` (module `github.com/go-spass/launchpad-openfeature-provider-go`,
  `go 1.27.4`), `README.md`, `LICENSE`, `.gitignore`, `CLAUDE.md`, and the `openspec/` tree. No Go
  packages, no `go.sum`.
- Measured on this tree, with zero packages: `go build ./...` exits 0 with a warning,
  `gofmt -l .` exits 0 and prints nothing, but **`go vet ./...` and `go test ./...` both exit 1**
  ("no packages to vet", "no packages to test"). Any gate that chains them fails on an empty
  module regardless of how the `Makefile` is written.
- `golangci-lint` 2.15.0 is installed on this machine via Homebrew. The v2 config schema differs
  from v1: it requires a top-level `version: "2"` and splits formatters out of `linters`.
- The project's workflow already fixes the gate's content: `gofmt -l .`, `go vet ./...`,
  `go test -race -cover ./...`. This change wraps those, it does not redefine them.

## Goals / Non-Goals

**Goals:**

- One command, `make check`, that runs the whole gate and exits non-zero on any failure.
- Each gate step also runnable alone, so a failure can be reproduced without the others.
- Lint behavior pinned to the repository (config file) rather than to each contributor's defaults.
- The gate exits 0 on the tree this change produces, so the next change starts from green.

**Non-Goals:**

- CI configuration. The `Makefile` is what CI will eventually call, but no workflow file is added.
- Pinning the `golangci-lint` binary version (see Decisions). Reproducibility of the linter's
  version is explicitly deferred.
- Coverage thresholds, coverage reports, or `-coverprofile` artifacts.
- Any provider behavior. `launchpad/doc.go` exists to make the toolchain non-empty, not to begin
  the implementation.

## Decisions

### 1. Add one real package rather than teach the `Makefile` to tolerate an empty module

`launchpad/doc.go` contains a package comment and `package launchpad`, nothing else. With one
package present, `go vet ./...` and `go test ./...` exit 0 (`test` reporting `[no test files]`),
so every target behaves the way it will behave for the rest of the project's life.

*Alternative considered:* guard the `vet` and `test` targets with `go list ./...` and skip when the
package list is empty. Rejected: the guard would survive long after it was needed, and a
conditional that turns "nothing ran" into success is exactly the failure mode a gate exists to
prevent. The chosen cost is one file that the next change overwrites the purpose of.

*Why this is not a TDD violation:* `doc.go` declares no behavior — no function, type, or variable —
so there is no assertion a test could make about it, and the workflow's TDD rule carries a tooling
exception. The first behavior in `package launchpad` arrives in the next change, test first.

### 2. `golangci-lint` from `PATH`, with an install hint, not pinned in `go.mod`

`make lint` checks for the binary with `command -v` and, if missing, prints
`brew install golangci-lint` and exits non-zero. `go.mod` gains no `tool` directive and the change
creates no `go.sum`.

*Alternatives considered:* (a) `go get -tool github.com/golangci/golangci-lint/v2/cmd/golangci-lint`
and `go tool golangci-lint run` — pins the version and needs no external install, but writes the
linter and its whole dependency tree into `go.mod`/`go.sum` before the provider has a single real
dependency, which for a library consumers import is noise in the one file they read. (b) `go install`
a pinned version into `./bin` — pinned and isolated, at the cost of a bootstrap step and a
`.gitignore` entry. Both remain available later; (a) in particular is the natural move when CI
arrives and version drift starts to matter. The trade-off accepted here is explicit: contributor
machines may run different linter versions, and the repo does not yet notice.

### 3. `fmt` writes, `check` only asserts

`make fmt` runs `gofmt -w` and is the one target that edits files. The gate's formatting step must
not mutate the tree, so `check` asserts cleanliness instead: `gofmt -l .` prints offending files but
**exits 0 either way**, so the assertion is on its output being empty — the target fails and names
the files when it is not. Likewise `tidy` (`go mod tidy`) is a maintenance target and is kept out of
`check`; a `go.mod`/`go.sum` drift assertion is a reasonable later addition but is not part of this
change.

### 4. Gate order: cheapest and most localized first

`check` runs formatting, then `vet`, then `lint`, then `test -race -cover`. A formatting or vet
failure is faster to produce and to fix than a race-enabled test run, so the gate fails on the
cheapest signal available. `govet` is also enabled inside `golangci-lint`, which overlaps the `vet`
target; the duplication is deliberate — `make vet` must work for someone who has not installed
`golangci-lint`, and the cost is a second pass over a tiny module.

### 5. `.DEFAULT_GOAL := check`

Bare `make` runs the gate, which is the command contributors need most often. Targets are declared
`.PHONY`, since none of them produces a file of its own name.

## Testing

This change adds no Go behavior, so it adds no Go test package or test file. Its verification is
the observable exit status and output of the targets themselves, run from the repository root on
the tree this change produces:

| Command | Expected observable outcome |
| --- | --- |
| `make build` | exits 0 |
| `make vet` | exits 0 (no "no packages to vet") |
| `make test` | exits 0, output includes `[no test files]` for `.../launchpad` |
| `make lint` | exits 0 with the binary installed; exits non-zero naming the install hint when `PATH` has no `golangci-lint` (verify by running the target with `PATH` restricted) |
| `make fmt` | exits 0, leaves the tree unchanged (`git diff --quiet` after it) |
| `make tidy` | exits 0, leaves `go.mod` unchanged and creates no `go.sum` |
| `make check` | exits 0 |
| `make check` with a deliberately misformatted file | exits non-zero and names that file; the file is then reverted |

`tasks.md` carries these as ordered steps, each with the failure to observe before the target
exists. Every check is local: no network access, no sleeps, no clock dependence. `go mod tidy`
needs no network here because the module has no requires — if that ever changes, `tidy` stays out
of `check` for exactly that reason.

The first real Go test arrives with the next change (the provider skeleton) in
`launchpad/provider_test.go`, package `launchpad_test`, asserting on `Metadata()` and on an
unknown-flag evaluation. Naming it here so the next change has somewhere to put its red.

## Risks / Trade-offs

- **`golangci-lint` may refuse a package with no declarations** ("no go files to analyze" or an
  empty run) → run `make lint` as its own task before wiring it into `check`, and record the real
  output. If it exits non-zero on `doc.go` alone, the fallback is to keep `lint` out of `check`
  until the first real code lands, and say so in `CLAUDE.md`.
- **Linter version drift across machines** (accepted in Decision 2) → the `.golangci.yml` carries
  `version: "2"`, which keeps the config schema stable; pinning the binary is a one-line follow-up
  (`go get -tool`) when CI makes drift visible.
- **`make test` asserts nothing until the next change** → accepted and stated in the proposal; the
  `[no test files]` line in `make test` output is the honest signal, and it disappears with the
  first real test.
- **A `Makefile` can drift from the workflow's documented commands** → `CLAUDE.md` is updated in
  this change to point at the `make` targets as the preferred entry point, so there is one source
  of truth rather than two lists.
- **Tab-versus-space breakage in recipes**, the classic `Makefile` failure → `make check` is run
  from a clean checkout as the change's last task, which catches it immediately.
