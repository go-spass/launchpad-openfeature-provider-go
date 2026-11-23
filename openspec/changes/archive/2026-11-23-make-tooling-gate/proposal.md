# Proposal

## Why

The repository is an empty scaffold with no build, test, or lint entry point, so there is no
single command that tells a contributor whether the tree is healthy. The project's workflow
requires every change to end with a verification gate (`gofmt`, `go vet`, `go test -race -cover`),
and right now each of those has to be remembered and typed by hand. Putting the gate in place
before the first provider code lands means the first feature is written against a check that
already works, instead of one assembled while behavior is in flight.

## What Changes

- Add a `Makefile` with the targets `build`, `test`, `lint`, `fmt`, `vet`, `tidy`, and `check`,
  where `check` runs the full gate the workflow already defines.
- Adopt `golangci-lint` as the project's linter, invoked from `make lint`. The binary is expected
  on `PATH`; when it is missing the target fails with an install hint rather than silently
  passing. `go.mod` stays free of tooling dependencies, and no `go.sum` is created by this change.
- Add a minimal `.golangci.yml` (schema version 2) so lint behavior is pinned to the repository
  rather than to each contributor's defaults.
- Add a single package, `launchpad`, consisting of `doc.go` with a package declaration and
  package comment and no code. This is what makes the gate runnable: on a module with zero
  packages both `go vet ./...` and `go test ./...` exit non-zero ("no packages to vet" / "no
  packages to test"), so `make check` cannot pass until at least one package exists. The package
  carries no behavior.
- Update `CLAUDE.md`: record the `make` targets as the preferred commands now that a `Makefile`
  exists, and correct the Go version section, which states `go.mod` declares `go 1.26.5` while it
  actually declares `go 1.27.4`.

No behavior of the provider changes, and no provider code is added. This is a tooling change, so
it declares no capabilities and sets `skip_specs: true` in its `.openspec.yaml`; the acceptance
criteria live in `design.md` and `tasks.md` as observable outcomes of running the targets.

## Capabilities

### New Capabilities

None. This change adds no system behavior — it introduces developer tooling only, so there is no
requirement for a spec to describe. `skip_specs: true` is set in the change's `.openspec.yaml`.

### Modified Capabilities

None. No existing capability's requirements change; the project has no specs yet
(`openspec list --specs` reports none).

## Impact

- **New files**: `Makefile`, `.golangci.yml`, `launchpad/doc.go`.
- **Modified files**: `CLAUDE.md` (commands section, Go version section).
- **Dependencies**: none added. `golangci-lint` becomes an external development prerequisite,
  installed outside the module (`brew install golangci-lint`); it is not required to build or
  test the provider, only to run `make lint` and `make check`.
- **Unaffected**: `go.mod` (no new requires, no `go.sum`), `README.md`, `LICENSE`, `.gitignore`.
- **Follow-on work**: the first feature change (the provider skeleton that satisfies the
  OpenFeature `FeatureProvider` interface) replaces `launchpad/doc.go`'s placeholder role with
  real code and its tests; `make test` is vacuous until then, reporting `[no test files]`.
- **Branch**: all work for this change is developed on `chore/make-tooling-gate`, branched from an
  up-to-date `main`. No part of this change — proposal, design, tasks, or implementation — is
  committed to `main`; it reaches `main` only through a merged pull request.
