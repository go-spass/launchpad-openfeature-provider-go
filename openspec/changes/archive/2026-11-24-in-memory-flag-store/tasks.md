# Tasks

## 1. Branch

- [x] 1.1 Create and switch to the branch `feat/in-memory-flag-store` off an up-to-date `main`, and verify `git rev-parse --abbrev-ref HEAD` reports it

## 2. Flag definition and flag-set validation

- [x] 2.1 Write `launchpad/flagset_test.go` in package `launchpad_test` with `TestNewFlagSet_RejectsMixedTypeVariants`, asserting `NewFlagSet` returns an error naming the flag key for variants `on: true` and `off: "disabled"`; run it and report the failure — it must fail on the nil-error assertion, not on a missing symbol
- [x] 2.2 Add the minimal declarations so 2.1 compiles and fails on its assertion: `Flag`, `State` with its enabled and disabled values, `FlagSet`, and `NewFlagSet` returning `(FlagSet{}, nil)`
- [x] 2.3 Implement the homogeneity check so 2.1 passes, and verify `go test ./launchpad -run TestNewFlagSet` is green
- [x] 2.4 Add failing tests for the remaining invariants — dangling `defaultVariant` naming both flag key and missing variant, empty variants, and a two-violation set whose error names both offenders — and report each failure
- [x] 2.5 Implement the remaining invariants, accumulating violations and joining them so the message is order-independent, and verify all `TestNewFlagSet` subtests pass
- [x] 2.6 Add a failing test that a well-formed set is accepted — two flags of different value types in one set, and a single-variant flag — then make it pass
- [x] 2.7 Document `Flag`, `State`, `FlagSet` and `NewFlagSet`, stating on `NewFlagSet` that the caller hands over ownership of object variant values because the copy is shallow, and verify `go vet ./...` and `gofmt -l .` are clean

## 3. Numeric normalization

- [x] 3.1 Write a failing test that `NewFlagSet` accepts a flag whose variants hold `50` and `2.5` together, and that a whole number beyond the range an integer evaluation can return fails validation naming the flag; report the failures
- [x] 3.2 Implement normalization in `NewFlagSet` — an explicit type switch over `int`, the sized signed and unsigned integers, `float32` and `float64`, mapping integral values to `int64` and fractional values to `float64` — and verify 3.1 passes
- [x] 3.3 Add a failing test that normalization is observable through evaluation later, by asserting `NewFlagSet` treats every number as one type for the homogeneity check, then confirm it passes with no further change

## 4. Immutability of a supplied flag set

- [x] 4.1 Write a failing test that mutating the caller's map after `NewFlagSet` does not change what the set holds, and report the failure
- [x] 4.2 Clone the flag set and each flag's variants on supply so 4.1 passes, and verify `go test -race ./launchpad` is green

## 5. Provider construction

- [x] 5.1 Write a failing test that `NewProvider(launchpad.WithFlags(set))` returns a provider and that `NewProvider()` with no options still returns one, asserting the eight existing tests in `provider_test.go` are untouched; report the failure on the missing `WithFlags`
- [x] 5.2 Add the `Option` type and `WithFlags`, change `NewProvider` to `NewProvider(opts ...Option) *Provider` holding a `FlagSet`, and verify 5.1 and every pre-existing test pass unchanged
- [x] 5.3 Add a failing test that a provider built from the zero `FlagSet`, as a caller who ignored the `NewFlagSet` error would pass, resolves every key as not found; make it pass

## 6. Resolution

- [x] 6.1 Write `TestEvaluation_ResolvesEnabledFlag` asserting `new-checkout` returns `true` with reason `STATIC` and variant `on`; run it and report the failure — today's unconditional `flagNotFound` returns the caller's default with `FLAG_NOT_FOUND`
- [x] 6.2 Add the unexported `resolve` seam returning `(any, openfeature.ProviderResolutionDetail)` — lookup, then state, then the default variant's value — and route `BooleanEvaluation` through it so 6.1 passes
- [x] 6.3 Write a failing table test resolving one enabled flag per value type through the matching typed method, then route the remaining four methods through `resolve` and verify it passes
- [x] 6.4 Write a failing test that a disabled flag returns the caller's default with reason `DISABLED`, no error and an empty variant, and that it is distinguishable from an unknown key reporting `FLAG_NOT_FOUND`; implement the state check and verify
- [x] 6.5 Write a failing test that a string request against a boolean flag returns the caller's default with `TYPE_MISMATCH` and `ERROR`, with the flag key in the message, covering every value type; implement the per-method assertion and verify
- [x] 6.6 Write a failing test that an object evaluation of a boolean flag returns `true` with reason `STATIC`, then confirm the pass-through assertion satisfies it
- [x] 6.7 Write a failing test for the numeric rule — `50` resolving for both integer and float requests, `2.5` resolving as float and reporting `TYPE_MISMATCH` as integer — implement the widening in `FloatEvaluation`, and verify
- [x] 6.8 Write a failing test that flag keys are matched exactly, with `New-Checkout` reporting `FLAG_NOT_FOUND` against a held `new-checkout`, and confirm it passes
- [x] 6.9 Add a goroutine fan-out test evaluating a shared provider concurrently and verify `go test -race ./launchpad` reports no data race

## 7. Narrowed adapter requirements and documentation

- [x] 7.1 Narrow the three existing `provider_test.go` tests that assert "any flag key" behavior so they assert it for a key the provider does not hold, and add the new scenarios — default pass-through, no `FLAG_NOT_FOUND` from a held flag, and empty-versus-named variant — then verify `go test ./launchpad` is green
- [x] 7.2 Add an end-to-end test through a registered OpenFeature SDK client asserting a resolved flag reports its value, variant and the `STATIC` reason, reusing the domain-scoped named provider and `openfeature.Shutdown` cleanup the existing end-to-end test uses
- [x] 7.3 Rewrite the flag paragraph in `launchpad/doc.go`, which states that the provider holds no flags and no key resolves, and verify the documented usage compiles as written
- [x] 7.4 Update the `## Project status` section of `CLAUDE.md` to describe the flag set, validation and resolution, and verify no sentence still claims the provider holds no flags

## 8. Verification gate

- [x] 8.1 Run the verification gate — `gofmt -l .`, `go vet ./...`, `go test -race -cover ./...` — and verify it passes with coverage reported for the `launchpad` package

## Workflow follow-up

- Sync the delta specs to the main specs and archive the change on this branch once the gate passes.
- `main` receives this change only through a merged pull request.
