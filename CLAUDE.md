# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project status

This repository holds its tooling plus one Go package, `launchpad`, containing the provider
skeleton: `doc.go` (package comment), `provider.go` (the provider) and `provider_test.go` (its
tests). `Provider`, constructed by `NewProvider()`, satisfies the OpenFeature
`FeatureProvider` interface in full and is asserted against it at compile time, so an SDK
upgrade that changes the method set fails `go build` rather than only `make test`.

The provider holds no flags. Every evaluation, for all five value types, returns the default
value the caller supplied with the `FLAG_NOT_FOUND` error code and the `ERROR` reason, built by
one shared unexported `flagNotFound` helper. Flag storage, targeting and rollout are not
implemented; the optional `StateHandler`, `EventHandler` and `Tracker` SDK interfaces are not
implemented either.

The module has one dependency, `github.com/open-feature/go-sdk`, pinned at `v1.20.0`, and a
`go.sum`. That pin is a floor for consumers, since Go's minimal version selection upgrades but
never downgrades; lowering it later is non-breaking.

Treat the architecture notes below as the intended design where they describe work that has not
landed yet. Update this file as real packages land.

## What LaunchPad is

LaunchPad is an [OpenFeature](https://openfeature.dev) **provider** for Go, starting as a simple
in-memory feature flag service. The split matters for every design decision:

- OpenFeature's Go SDK owns the client-facing API, evaluation context, and hooks. Consumers call
  the SDK, never LaunchPad directly.
- LaunchPad implements the SDK's provider interface so users can swap it for LaunchDarkly, flagd,
  etc. without touching their call sites.

Consequence: the provider's public surface is dictated by the OpenFeature spec, not by what is
convenient here. Before designing a type, check the `github.com/open-feature/go-sdk` interface it
must satisfy — `FeatureProvider` (metadata, hooks, and typed evaluation methods for bool, string,
int, float, and object), with each evaluation returning a resolution detail carrying value, reason,
variant, and error code rather than a bare `(value, error)` pair. Verify exact signatures against
the SDK version pinned in `go.mod` once the dependency is added; the interface has changed across
SDK majors.

Module path: `github.com/go-spass/launchpad-openfeature-provider-go`

## Development workflow

Two rules govern every change in this repository. They are not style preferences; treat them as
part of the definition of done.

### TDD — always start with a failing test

- Every behavioral change starts with a test that fails. Write the test, run it, and report the
  actual failure output *before* writing any implementation code.
- In Go, a build failure from a symbol that does not exist yet is not a useful red. Add the
  minimal declaration — the type or function returning zero values — so the test compiles and
  fails on its assertion, then implement against that failure.
- Then write the minimum code to go green, then refactor with the test staying green.
- A task is not done until `go test -race ./...` passes for the packages it touched.
- Narrow exceptions, so they are not improvised: docs-only edits, `go.mod` and tooling changes,
  and pure refactors already covered by passing tests.

### Branching — never develop on `main`

- `main` holds no feature work: not the OpenSpec proposal, specs, design or tasks, and not the
  implementation.
- Before creating or editing an OpenSpec change, and before the first code edit, create the branch
  `<type>/<change-id>` off an up-to-date `main`. `<type>` is the conventional-commit type
  (`feat`, `fix`, `refactor`, `docs`, `test`, `chore`); `<change-id>` is the change's directory
  name under `openspec/changes/` — e.g. `feat/in-memory-provider`.
- A change's whole lifecycle — proposal through archive — lives on that single branch and reaches
  `main` only through a merged PR.
- Check `git rev-parse --abbrev-ref HEAD` before the first write of a session. If it reports
  `main`, stop and branch first.

## Commands

A `Makefile` exists; prefer its targets over raw `go` commands. Bare `make` runs `check`.

```sh
make check   # the full gate: fmt-check, vet, lint, test — run this before calling a task done
make build   # go build ./...
make vet     # go vet ./...
make test    # go test -v -race -cover ./...
make lint    # golangci-lint run ./...
make fmt     # gofmt -w: the only target that edits files
make tidy    # go mod tidy
```

`make lint` needs `golangci-lint` on `PATH` (`brew install golangci-lint`); it is not a module
dependency, and the target fails with that hint rather than passing silently when the binary is
missing. Its linter set is pinned in `.golangci.yml` (config schema `version: "2"`).

`check` asserts formatting instead of applying it: `gofmt -l .` exits 0 even when it lists
unformatted files, so the `fmt-check` target fails on non-empty output. `tidy` stays out of the
gate because it rewrites `go.mod`.

Use the toolchain directly for what has no target — single tests above all:

```sh
go test ./path/to/pkg -run TestName        # single test
go test ./path/to/pkg -run 'TestX/subtest' # single subtest
```

## Go version

`go.mod` declares `go 1.27.4`, which matches the installed toolchain (1.27.4). Don't bump the
`go` directive casually — it sets the minimum version consumers of this provider must have, and a
library should stay conservative.
