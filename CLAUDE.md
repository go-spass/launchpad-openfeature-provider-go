# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project status

This repository is an empty scaffold: `go.mod`, `README.md`, `LICENSE`, and `.gitignore` only.
There is no Go source, no `go.sum`, no tests, and no `Makefile` yet. Every commit so far is
project setup (a spec-kit scaffold was added in `254f9a5` and removed again in `bc292b8`).

Treat the architecture notes below as the intended design, not as a description of existing code.
Update this file as real packages land.

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

No `Makefile` exists yet, so use the Go toolchain directly:

```sh
go build ./...
go vet ./...
go test ./...
go test -race -cover ./...
go test ./path/to/pkg -run TestName        # single test
go test ./path/to/pkg -run 'TestX/subtest' # single subtest
gofmt -l .                                 # list unformatted files
go mod tidy
```

If a `Makefile` is added, prefer its targets over raw `go` commands.

## Go version

`go.mod` declares `go 1.26.5`; the installed toolchain is newer (1.27.x). Don't bump the `go`
directive casually — it sets the minimum version consumers of this provider must have, and a
library should stay conservative.
