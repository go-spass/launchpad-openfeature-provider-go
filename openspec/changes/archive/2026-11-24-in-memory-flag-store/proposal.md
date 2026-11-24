# Proposal

## Why

LaunchPad satisfies the OpenFeature provider contract but holds no flags, so every evaluation
returns the caller's default with `FLAG_NOT_FOUND`. The provider is correct and useless: nothing
it can be handed changes what a consumer observes.

The obvious next step is loading a flag file, but that bundles two contracts into one change — a
Go domain type and a YAML document schema — and the document would shape the type rather than the
other way round. Serialization concerns (string enums, absent-versus-zero, whatever the decoder
made of a number) leak into a public API permanently and cannot be withdrawn. So this change
introduces the flag definition and real resolution with flags constructed in Go, and leaves the
file format to the change that follows, which then maps onto a type that already exists and is
already tested.

## What Changes

- A flag definition type: a key, an enabled/disabled state, a set of named variants, and the
  variant chosen when nothing else selects one. Variant values carry the flag's type; no flag
  declares its type separately.
- Invariants enforced where the flags are supplied, not at evaluation time: a flag's variants
  must all hold the same value type, and the default variant must name a variant that exists.
  A flag set that violates either is rejected with an error naming the offending flag.
- A provider that serves a flag set, supplied at construction. `NewProvider()` keeps its current
  signature and behavior when given no flags, so this is additive and not **BREAKING**.
- Real resolution across all five value types: a known, enabled flag returns its default
  variant's value with the `STATIC` reason and the variant name; a disabled flag returns the
  caller's default with the `DISABLED` reason; an unknown flag keeps today's `FLAG_NOT_FOUND`
  behavior.
- A type-mismatch outcome: asking for a type the resolved variant does not hold returns the
  caller's default with `TYPE_MISMATCH`, rather than a zero value or a panic.
- One rule for numbers, because the two OpenFeature numeric methods and the eventual decoders
  disagree: integral variant values resolve for both integer and float requests, and a
  non-integral value is a type mismatch for an integer request.

Out of scope, each left to its own later change: any file format, decoding or on-disk source;
multi-source sync; targeting and percentage rollout; flag-set replacement or reload after
construction; and the optional `StateHandler`, `EventHandler` and `Tracker` interfaces.

Development happens on the branch `feat/in-memory-flag-store`. No work for this change —
proposal, specs, design, tasks or implementation — is committed to `main`; `main` receives it
only through a merged pull request.

## Capabilities

### New Capabilities

- `flag-store`: what a flag definition is and what a well-formed set of them looks like — the
  key, state, variants and default variant, the invariants checked when a set is supplied, and
  the guarantee that a supplied set cannot be mutated out from under a running provider. This
  capability owns the shape of a flag and will keep owning it when the shape arrives from a file
  or a sync source instead of from Go code.
- `flag-evaluation`: how a flag's value is decided — which variant is selected, how state
  affects the outcome, whether the selected value satisfies the requested type, and which reason
  and error code the outcome carries. This is the capability the previous change reserved for
  this purpose. Targeting and rollout will add requirements here without disturbing the adapter
  contract above it.

### Modified Capabilities

- `openfeature-provider`: three requirements assert their behavior for *every* flag key, which
  was accurate only while the provider held no flags. `Unresolvable flag returns the caller's
  default value`, `Unresolvable flag reports FLAG_NOT_FOUND` and `No variant is reported when no
  flag resolves` narrow from "any flag key" to "a flag key that does not resolve". The adapter
  contract itself does not change: identity, hooks, interface conformance and typed evaluation
  coverage are untouched.

## Impact

- **Code**: `launchpad/provider.go` gains the flag set and replaces the unconditional
  `flagNotFound` return in each of the five evaluation methods with a shared resolution step.
  New flag domain declarations land in the `launchpad` package rather than a second package;
  there is still one consumer-facing concept, so a package boundary would add an import without
  removing a dependency.
- **Public API**: additive. `NewProvider()` with no arguments keeps returning a provider that
  resolves nothing, so the previous change's tests and documented usage stay valid.
- **Dependencies**: none added. The module keeps its single direct dependency on the
  OpenFeature SDK, and the tests need no fixtures on disk.
- **Documentation**: `launchpad/doc.go` and `CLAUDE.md` both state that the provider holds no
  flags and that no key resolves; both become wrong when this change lands.
- **Follow-on work**: the YAML flag file change depends on the type this change introduces, and
  on the decision recorded in `design.md` to adopt a declared subset of the flagd flag
  definition format.
