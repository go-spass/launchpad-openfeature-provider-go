# Proposal

## Why

The repository has a verification gate but no provider: `launchpad/doc.go` holds a package
comment and nothing else, so `make test` reports `[no test files]` and nothing in the module
has ever been compiled against the OpenFeature SDK. Until a type satisfies
`openfeature.FeatureProvider`, every later decision about flags — how a flag declares its value
type, how variants resolve, how targeting reports a reason — is designed against a remembered
interface rather than a compiled one. `CLAUDE.md` already warns that the interface has moved
across SDK majors.

This change pins the SDK, satisfies the interface, and establishes the end-to-end test harness
(register the provider with a real `openfeature` client, call it, assert on the resolution
detail) that every subsequent change will hang its assertions off. It deliberately ships no flag
storage: an empty provider has well-defined behavior under the OpenFeature specification, and
that behavior is what this change implements and tests.

## What Changes

- Add the first module dependency, `github.com/open-feature/go-sdk` pinned at `v1.20.0`, which
  creates the repository's first `go.sum`.
- Add `launchpad.Provider`, constructed by `launchpad.NewProvider()` and used as `*Provider`,
  satisfying `openfeature.FeatureProvider` in full:
  - `Metadata()` reports a stable, non-empty provider name.
  - `Hooks()` reports an empty, non-nil hook slice — the provider contributes no hooks.
  - `BooleanEvaluation`, `StringEvaluation`, `IntEvaluation`, `FloatEvaluation` and
    `ObjectEvaluation` each return the caller's `defaultValue` unchanged, with
    `ResolutionError` carrying the `FLAG_NOT_FOUND` error code and `Reason` set to `ERROR`.
    No flag key resolves, because the provider holds no flags yet.
- Add a compile-time assertion that `launchpad.Provider` satisfies the interface, so an SDK
  upgrade that changes the method set fails at `go build` rather than at a call site.
- Replace the placeholder role of `launchpad/doc.go`: the package comment stays, and the
  package now carries behavior and tests.
- Update `CLAUDE.md`: the "Project status" section currently describes a scaffold with no
  provider code and no `go.sum`, both of which this change makes untrue.

Out of scope, each deliberately left to its own later change: any flag type, variant or store;
the optional `StateHandler`, `EventHandler` and `Tracker` interfaces; and any targeting or
rollout evaluation. This change introduces no flag domain types at all, so there is nothing yet
for OpenFeature types to leak into.

## Capabilities

### New Capabilities

- `openfeature-provider`: how LaunchPad presents itself to the OpenFeature Go SDK — provider
  identity, hook contribution, the five typed evaluation entry points, and the mapping from an
  internal resolution outcome onto the specification's value, variant, reason and error-code
  fields. This capability owns the adapter contract and will keep owning it as flag storage and
  targeting arrive underneath; it does not own how a flag's value is decided, which belongs to a
  separate flag-evaluation capability introduced by a later change.

### Modified Capabilities

None. `openspec list --specs` reports no specs, so no existing requirement changes.

## Impact

- **New files**: `launchpad/provider.go`, `launchpad/provider_test.go`, `go.sum`.
- **Modified files**: `go.mod` (first `require`), `CLAUDE.md` (project-status section),
  `launchpad/doc.go` only if the package comment needs rewording.
- **Dependencies**: `github.com/open-feature/go-sdk v1.20.0` becomes the module's only
  dependency. It is a runtime dependency of consumers, not a tooling one, so the pin is part of
  the provider's public contract. The `go` directive stays at `1.27.4`.
- **Public API**: introduces `launchpad.Provider` and `launchpad.NewProvider`. This is the
  module's first exported surface; everything added later resolves flags through it.
- **Consumers**: a consumer can register LaunchPad with `openfeature.SetProviderAndWait` and
  call any typed evaluation method without panicking. Every call returns the supplied default,
  so the provider is correct but not yet useful.
- **Verification**: `make check` runs against a module with dependencies for the first time;
  `make tidy` becomes meaningful. `make test` stops reporting `[no test files]`.
- **Branch**: all work for this change is developed on `feat/provider-skeleton`, branched from
  an up-to-date `main`. No part of this change — proposal, specs, design, tasks, or
  implementation — is committed to `main`; it reaches `main` only through a merged pull request.
