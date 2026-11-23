# Design

## Context

See `proposal.md` — Why. The constraints that actually shape the approach:

- The module has no dependencies and no `go.sum`. This change adds both for the first time.
- The interface to satisfy, verified against `github.com/open-feature/go-sdk v1.20.0`:

  ```go
  type FeatureProvider interface {
      Metadata() Metadata
      BooleanEvaluation(ctx context.Context, flag string, defaultValue bool, flatCtx FlattenedContext) BoolResolutionDetail
      StringEvaluation(ctx context.Context, flag string, defaultValue string, flatCtx FlattenedContext) StringResolutionDetail
      FloatEvaluation(ctx context.Context, flag string, defaultValue float64, flatCtx FlattenedContext) FloatResolutionDetail
      IntEvaluation(ctx context.Context, flag string, defaultValue int64, flatCtx FlattenedContext) IntResolutionDetail
      ObjectEvaluation(ctx context.Context, flag string, defaultValue any, flatCtx FlattenedContext) InterfaceResolutionDetail
      Hooks() []Hook
  }
  ```

- Each typed detail is an alias for `GenericResolutionDetail[T]`, which embeds
  `ProviderResolutionDetail{ResolutionError, Reason, Variant, FlagMetadata}`. So every evaluation
  method shares the same non-value half of its return, which is the lever this design pulls on.
- `StateHandler`, `EventHandler` and `Tracker` are opt-in interfaces the SDK probes for at
  registration. Not implementing them is a valid provider, and they stay out of this change.
- `openfeature.ErrorReason` and `openfeature.NewFlagNotFoundResolutionError(msg)` already exist;
  the error code and reason strings are not ours to invent.
- Registering a provider mutates SDK-global state. That is the one thing in this change that can
  make tests order-dependent, and it is handled explicitly below.

## Goals / Non-Goals

**Goals:**

- One shared construction point for the "no flag resolved" outcome, so the five evaluation
  methods cannot drift apart in reason, error code or variant.
- A public API shaped so that adding flag storage in the next change is additive, not breaking.
- Tests that exercise the provider only through its exported surface and through a real SDK
  client, with no network access, no sleeps and no fixtures on disk.

**Non-Goals:**

- Any abstraction for a future flag store. There is nothing to abstract over yet, and inventing
  the seam before the flag type exists would prejudge the representation question.
- Deciding whether the flag domain lives in `launchpad` or in a second package. This change adds
  no domain types, so the decision has no content to apply to and is deferred.
- Choosing the oldest SDK version LaunchPad could compile against. See Decisions.

## Decisions

### Pin `github.com/open-feature/go-sdk` at `v1.20.0`

`v1.20.0` is the current release and is the version the interface above was read from.

*Alternative considered:* require the oldest SDK release whose `FeatureProvider` method set still
matches, which for a library is the more courteous choice — Go's minimal version selection will
upgrade a consumer but never downgrade one, so the `require` line is a floor imposed on everyone
who imports LaunchPad. Rejected for now because finding that floor means compiling against each
of ~20 releases, and the module has no external consumers yet. Recorded as a trade-off rather
than settled: lowering the floor later is a non-breaking change.

### `NewProvider() *Provider`, with pointer receivers

*Alternative considered:* a value type with value receivers, which is simpler while `Provider`
has no fields. Rejected because the next change gives `Provider` a flag store, and a change after
that will likely give it a mutex — at which point value receivers become wrong (a copied mutex)
and switching from `Provider` to `*Provider` is a breaking API change. Paying for the pointer now
costs nothing and removes a scheduled break.

`NewProvider()` takes no arguments in this change. Flag sources arrive later as variadic options
or as an explicit store argument; either is additive.

### A single unexported helper builds every unresolved outcome

```go
func flagNotFound(flag string) openfeature.ProviderResolutionDetail
```

It returns `Reason: openfeature.ErrorReason`, `ResolutionError:
openfeature.NewFlagNotFoundResolutionError(...)` with a message naming the flag key, and leaves
`Variant` at its zero value. Each of the five evaluation methods is then one statement:

```go
func (p *Provider) BooleanEvaluation(_ context.Context, flag string, defaultValue bool, _ openfeature.FlattenedContext) openfeature.BoolResolutionDetail {
    return openfeature.BoolResolutionDetail{Value: defaultValue, ProviderResolutionDetail: flagNotFound(flag)}
}
```

*Alternative considered:* a generic helper returning `GenericResolutionDetail[T]` so each method
is a single call. It works, but gains nothing over the above and makes the five methods read less
like the interface they implement. The shared half is already the only part at risk of drifting.

*Alternative considered:* letting each method construct its own detail inline. Rejected — five
independent copies of the same reason/error-code pair is exactly the drift this helper prevents,
and when the next change replaces "always not found" with a store lookup, there is one call site
per method to rewrite rather than one block per method.

### Conformance is asserted in the implementation file, not in a test

`var _ openfeature.FeatureProvider = (*Provider)(nil)` goes in `launchpad/provider.go`. An SDK
upgrade that changes the method set then fails `go build` and `make check`, not just `make test`.

*Alternative considered:* asserting it in the test file only. Rejected because a consumer running
`go build` on a broken combination should see the failure without running our tests.

### `Hooks()` returns a non-nil empty slice

`return []openfeature.Hook{}`, not `return nil`. Both satisfy the interface and both are safe to
range over, but the spec requirement is written as "empty and safe to iterate", and a non-nil
empty slice makes that assertable directly rather than relying on Go's nil-slice semantics.

### Tests live in an external test package and register a domain-scoped provider

- **File:** `launchpad/provider_test.go`, **package:** `launchpad_test`. The external test
  package means the tests can only reach the exported API, which is the surface the spec
  describes. A test that needed `launchpad`'s internals would be testing the wrong thing at this
  stage.
- The end-to-end requirement registers through `openfeature.SetNamedProviderAndWait("launchpad-test", p)`
  and evaluates through `openfeature.NewClient("launchpad-test")`, rather than
  `SetProviderAndWait` plus the default client.

  *Why:* provider registration writes to SDK-global state. Using the default provider slot makes
  the test clobber anything another test registered, and makes behavior depend on test order.
  A named domain keeps the registration confined to a key this test owns. The end-to-end subtest
  does not call `t.Parallel()`, and registers `openfeature.Shutdown` via `t.Cleanup` so the
  global map does not leak into other packages' tests.

  `SetNamedProviderAndWait` returns once the provider is ready, so no sleep or poll is needed.

### Test shape: a closure table for the shared half, per-type subtests for values

The five evaluation methods have five different signatures, so one table cannot drive them
directly without erasing the types under test. Instead:

- A table of `map[string]func(*launchpad.Provider) openfeature.ProviderResolutionDetail` — one
  entry per value type — drives the assertions that are identical across types: error code
  `FlagNotFoundCode`, reason `ErrorReason`, empty `Variant`, and an error message containing the
  flag key. This covers the FLAG_NOT_FOUND, reason, variant and message requirements in one
  place, for all five types, and a sixth type added by a future SDK would show up as a missing
  table entry.
- Default-value passthrough gets one typed subtest per value type, because the whole point is
  that the concrete type survives. Boolean is asserted with both `true` and `false` defaults, so
  a hardcoded return cannot pass.

Requirement-to-test mapping, so each failing test can be written before its code:

| Spec requirement | Test in `launchpad/provider_test.go` |
|---|---|
| Provider interface conformance | compile-time assertion in `provider.go`; `TestProviderSatisfiesFeatureProvider` assigns to the interface |
| Provider identity | `TestMetadata` — non-empty name; same name across two instances and repeated reads |
| Hook contribution | `TestHooks` — length 0, non-nil |
| Typed evaluation coverage | `TestEvaluation_AllTypesAnswered` — table of closures, nil `FlattenedContext` subtest |
| Unresolvable flag returns the caller's default | `TestEvaluation_ReturnsCallerDefault` — one subtest per type, boolean with both values |
| Unresolvable flag reports FLAG_NOT_FOUND | `TestEvaluation_ReportsFlagNotFound` — closure table |
| No variant is reported | `TestEvaluation_ReportsNoVariant` — closure table |
| End-to-end through an OpenFeature client | `TestClientEvaluation_MissingFlag` — named provider, `BooleanValue` and `BooleanValueDetails` |

## Risks / Trade-offs

- **The `require` line sets a version floor for every consumer** → accepted deliberately; the
  module has no external consumers, and lowering the floor later is non-breaking. Recorded in
  Decisions rather than left implicit.
- **SDK-global provider registration makes tests order-dependent** → domain-scoped named
  provider, no `t.Parallel()` on the end-to-end subtest, `openfeature.Shutdown` in `t.Cleanup`.
- **A provider where every flag is missing looks broken** → the package comment in `doc.go` and
  the `Provider` doc comment state that the provider holds no flags yet and that every evaluation
  returns the caller's default with `FLAG_NOT_FOUND`. `CLAUDE.md`'s project-status section is
  updated in the same change so the repository does not describe itself as having no provider.
- **First dependency means the first `go.sum`** → `go.sum` is committed, and `make tidy` stops
  being a no-op. `make check` deliberately excludes `tidy`, so the gate does not rewrite
  `go.mod`; a stale `go.mod` would therefore surface as a build failure rather than being
  silently fixed. Acceptable, and unchanged from the existing gate's design.
- **Writing the interface assertion before the methods exist produces a build error, not a test
  failure** → the repository's TDD rule anticipates this: the tasks add the five method stubs
  returning zero values first, so the tests compile and fail on their assertions rather than on a
  missing symbol.

## Migration Plan

Not applicable. The module has no released version and no consumers; this change only adds
exported API.

## Open Questions

- What is the oldest `go-sdk` release whose `FeatureProvider` method set LaunchPad satisfies?
  Answering it lowers the floor in `go.mod`, which is a non-breaking follow-up and changes
  nothing about this change's specs, approach or tasks.
