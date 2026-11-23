# Tasks

This change adds no Go behavior, so no task writes a Go test. The analogue of a failing test for a
`Makefile` target is running it and seeing it fail for the intended reason, so each target is
preceded by a task that runs it and records the real output. Run every command from the repository
root.

## 1. Branch

- [x] 1.1 Create and switch to `chore/make-tooling-gate` off an up-to-date `main`; verify with
  `git rev-parse --abbrev-ref HEAD` reporting `chore/make-tooling-gate`. *(Done before the
  planning artifacts were written, per the project's branch discipline.)*

## 2. Make the toolchain non-empty

- [x] 2.1 Record the baseline red: run `go vet ./...` and `go test ./...` and verify both exit 1
  (`echo $?`) with "no packages to vet" / "no packages to test". This is the failure that the next
  task fixes — capture the output before changing anything.
- [x] 2.2 Add `launchpad/doc.go` containing only a package comment and `package launchpad`; verify
  `go vet ./...` now exits 0 and `go test ./...` exits 0 printing `[no test files]` for
  `.../launchpad`, and that `gofmt -l .` prints nothing.

## 3. Lint configuration

- [x] 3.1 Verify `golangci-lint` is usable and learn how it behaves on a declaration-free package:
  run `golangci-lint --version` and `golangci-lint run ./...`, and record the exit status and
  output. If it exits non-zero on `doc.go` alone, apply the design's fallback (keep `lint` out of
  `check`, note it in `CLAUDE.md`) and say so in the task notes rather than working around it
  silently.
- [x] 3.2 Add `.golangci.yml` with `version: "2"` and an explicit linter set; verify
  `golangci-lint run ./...` exits 0 and that removing the config changes nothing it reports
  (i.e. the config is valid, not merely ignored) — `golangci-lint config verify` if available.

## 4. Makefile targets

- [x] 4.1 Record the red: run `make check` and verify it fails with no rule to make target /
  no `Makefile` present.
- [x] 4.2 Add the `Makefile` with `.PHONY` declarations, `.DEFAULT_GOAL := check`, and the targets
  `build`, `vet`, `test`, `lint`, `fmt`, `tidy`, `check`; verify `make build`, `make vet`, and
  `make test` each exit 0, with `make test` running `go test -race -cover ./...`.
- [x] 4.3 Verify `make lint` exits 0 with the binary installed, and that it fails with the
  `brew install golangci-lint` hint when the binary is absent — run it with a restricted
  environment (e.g. `env PATH=/usr/bin:/bin make lint`) and confirm a non-zero exit and the hint
  in the output.
- [x] 4.4 Verify `make fmt` exits 0 and leaves the tree unchanged (`git diff --quiet` succeeds
  after it), and that `make tidy` exits 0, leaves `go.mod` unchanged, and creates no `go.sum`
  (`git status --porcelain` empty, `test ! -f go.sum`).
- [x] 4.5 Verify the formatting assertion actually fails the gate: temporarily misformat
  `launchpad/doc.go` (add stray indentation), run `make check`, confirm it exits non-zero and names
  the file, then restore the file with `git checkout -- launchpad/doc.go` and confirm `make check`
  exits 0 again.
- [x] 4.6 Verify bare `make` runs the gate (`.DEFAULT_GOAL`): run `make` and confirm it performs
  the same steps as `make check` and exits 0.

## 5. Documentation

- [x] 5.1 Update `CLAUDE.md`'s Commands section to present the `make` targets as the preferred
  entry point, keeping the raw `go` equivalents for single-test runs; verify every command written
  there runs as written and exits as documented.
- [x] 5.2 Correct `CLAUDE.md`'s Go version section, which says `go.mod` declares `go 1.26.5`;
  verify against `go.mod`, which declares `go 1.27.4`, and keep the advice against bumping the
  directive casually.
- [x] 5.3 Update `CLAUDE.md`'s Project status section, which described the repository as an empty
  scaffold with no `Makefile`; verify it names what the tree now holds (`Makefile`,
  `.golangci.yml`, the `launchpad` package and why `doc.go` exists, still no `go.sum` and no
  tests) and no longer contradicts the Commands section.

## 6. Verification gate

- [x] 6.1 Run the full gate from a clean tree and verify each step passes: `gofmt -l .` prints
  nothing, `go vet ./...` exits 0, `go test -race -cover ./...` exits 0, `golangci-lint run ./...`
  exits 0, and `make check` exits 0. Confirm `git status --porcelain` lists only the files this
  change intends to add or modify (`Makefile`, `.golangci.yml`, `launchpad/doc.go`, `CLAUDE.md`,
  and the `openspec/changes/make-tooling-gate/` artifacts).

## Workflow follow-up

- Open a pull request from `chore/make-tooling-gate` into `main`; no part of this change reaches
  `main` any other way.
- Archive the change (`openspec archive`) on this branch once the gate passes and the PR is
  approved, then verify the archived result.
