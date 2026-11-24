# Design

## Context

See `proposal.md` — Why. The constraints that shape the approach:

- `Provider` is a stateless struct with pointer receivers, and `NewProvider()` takes no arguments
  and cannot fail. The previous change chose pointer receivers specifically so that giving the
  provider state would not be a breaking API change.
- All five evaluation methods funnel through one unexported `flagNotFound` helper. That is the
  seam this change widens.
- Tests live in the external package `launchpad_test` and exercise only the exported surface,
  with no fixtures on disk.
- Specs: `specs/flag-store/spec.md` (what a flag is, invariants, immutability),
  `specs/flag-evaluation/spec.md` (how a value is decided), and the narrowed universal claims in
  `specs/openfeature-provider/spec.md`.

## Goals / Non-Goals

**Goals:**

- Validation that cannot be bypassed: it must be impossible to hand the provider a flag set that
  violates an invariant, rather than merely discouraged.
- A single resolution seam shared by all five typed methods, so state handling, variant
  selection, reason and error code cannot drift apart between value types.
- A supply-time shape that the later YAML change can reuse unchanged, so loading a file adds a
  decoder and no new provider API.

**Non-Goals:**

- Any mutex or other synchronization. See Decisions — nothing is mutable in this change, and
  adding a lock now would be unexercised code.
- Deep-copying nested object variant values. See Risks.
- Reflection. The accepted numeric types are enumerated explicitly.

## Decisions

### A validated `FlagSet` value, not a fallible constructor

Invariants are checked when flags are supplied, and something must report that failure.
`NewProvider() *Provider` cannot report it.

*Chosen:* a two-step shape. `NewFlagSet(map[string]Flag) (FlagSet, error)` validates and returns
an opaque, already-valid value; `NewProvider(WithFlags(set))` cannot fail.

```
  map[string]Flag  --NewFlagSet-->  FlagSet  --WithFlags-->  NewProvider  -->  *Provider
   (caller's data)    validates     (valid by      option      infallible
                      here          construction)
```

*Alternative — `NewProvider(opts ...Option) (*Provider, error)`:* rejected because it changes
`NewProvider`'s signature, breaking every existing call site and the documented usage in
`doc.go`, one change after that API shipped.

*Alternative — infallible construction, error surfaced later through `StateHandler.Init`:*
rejected because a typo in a flag definition would then present as a provider that is merely
unhealthy at runtime, and because it forces this change to implement `StateHandler`, which the
proposal puts out of scope.

The decisive advantage is that validity becomes a property of the type. A `FlagSet` in hand has
already been checked, so `WithFlags` has nothing to re-verify and no error path. It also makes
the follow-on change additive in the smallest possible way: `LoadFile(path) (FlagSet, error)`
returns the same type, and the provider API does not move at all.

The zero `FlagSet` is a valid empty set. A caller who ignores the error from `NewFlagSet` and
passes the zero value gets a provider that resolves nothing — which is what
`specs/flag-store/spec.md` requires, rather than a provider holding half-validated flags.

### Variadic options rather than an explicit flag-set argument

*Chosen:* `NewProvider(opts ...Option)`. `NewProvider()` with no arguments keeps compiling and
keeps its current behavior, so the previous change's tests and documented usage remain valid
verbatim.

*Alternative — `NewProvider(set FlagSet)`:* rejected for the same signature-break reason, and
because later knobs (a logger, a default state) would each force another break.

### One shared `resolve`, mirroring `flagNotFound`

*Chosen:* an unexported method returning the untyped outcome plus the detail, with each typed
method doing only its own assertion:

```
  BooleanEvaluation --+
  StringEvaluation ---+
  IntEvaluation ------+--> resolve(flag) (any, ProviderResolutionDetail)
  FloatEvaluation ----+       |  1. lookup        -> miss: FLAG_NOT_FOUND / ERROR
  ObjectEvaluation ---+       |  2. state check   -> disabled: DISABLED, no variant
                              |  3. variants[DefaultVariant]
                              v     -> hit: value, STATIC, variant name
                       per-method type assertion
                         ok  -> value
                         bad -> caller default, TYPE_MISMATCH / ERROR
```

*Alternative — a generic helper over the five types, as the SDK's `memprovider` does with
`genericResolve[T]`:* rejected for now. Generics would remove the repeated assertion but cannot
express the numeric widening rule, which is asymmetric between `int64` and `float64`, without a
type switch inside the generic function anyway. Revisit if a sixth value type appears.

The reason this seam matters beyond deduplication: steps 1–3 are exactly where targeting will
insert itself later, and `resolve` returning `(any, detail)` is already the right signature for
"a variant was selected, by whatever means". Static resolution is the degenerate case.

### Numbers are normalized at supply time, by an explicit type switch

Go callers write `50`, `int64(50)` or `50.0` for the same intent, and the eventual YAML and JSON
decoders disagree with each other about which Go type a bare `50` becomes.

*Chosen:* `NewFlagSet` normalizes every numeric variant value once — integral values to `int64`,
values with a fractional part to `float64` — via an explicit type switch over `int`, the sized
signed and unsigned integers, `float32` and `float64`. Evaluation then needs one assertion:
`IntEvaluation` accepts `int64`; `FloatEvaluation` accepts `float64` or widens `int64`.

*Alternative — store values as given and reconcile at evaluation time:* rejected because the
reconciliation would then run on every evaluation instead of once, and because the homogeneity
check would have to understand the same numeric equivalences separately.

*Alternative — `reflect` over all numeric kinds:* rejected as a heavier dependency on reflection
than an enumerated switch, for a fixed and small set of types. A value of any type outside the
switch is left as it is and categorised as an object, which an object evaluation returns
unchanged, so the switch needs no denylist to stay in step with Go.

One numeric value is refused rather than normalized: a whole number beyond the range an integer
evaluation can return. `int64` cannot hold it and `float64` would silently lose precision, so
there is no correct value to resolve to and it is a supply-time error naming the flag.

*Alternative — also refusing `complex64`, `complex128` and `uintptr`, which no format could
serialize:* rejected as an arbitrary denylist. A channel, a function and a pointer are equally
unserializable and equally likely to be a mistake, and refusing three types while accepting
infinitely many others would be an oddity written into a durable spec. Validating the full set of
types a variant may hold belongs with the format that motivates it — see the settled context
below — not here, where nothing yet constrains what Go code may supply.

Consequence recorded in `specs/flag-store/spec.md`: for the homogeneity invariant, every number
is one type, so `{standard: 50, precise: 2.5}` is a well-formed float flag rather than a
mixed-type error.

### No synchronization

A `FlagSet` is immutable once constructed, the provider never replaces it, and `maps.Clone` at
supply time means the caller's map is not retained. Concurrent evaluation is therefore
concurrent reads of a map nothing writes, which needs no lock.

*Alternative — an `sync.RWMutex` now, as `memprovider` has:* rejected because `memprovider`
needs it for `UpdateFlags`, which this change does not have. The lock arrives with reload, in
the change that introduces it, where a test can actually exercise contention. Adding it now
would be untested code defending against a mutation that cannot happen.

### Object evaluation never type-mismatches

`ObjectEvaluation` takes and returns `any`, so every resolved value satisfies it. It is the one
method whose typed step is a pass-through. Recorded as a scenario rather than left implicit,
because it reads like an omission otherwise.

### Settled context for the follow-on YAML change

Not implemented here, recorded so the next change starts from a decision rather than a debate:
LaunchPad will adopt a **declared subset of the flagd flag definition format** — `flags`, and
per flag `state`, `defaultVariant`, `variants` — accepting and ignoring `$schema`, and
**rejecting** `targeting` and `metadata` with an explicit "not supported yet" error rather than
ignoring them, so a dropped rollout rule cannot pass silently. The decoder will map onto the
`Flag` and `FlagSet` types this change introduces, and will not carry serialization tags on
them.

That change is also the right home for validating which types a variant may hold at all. The
format, not Go, is what makes a channel or a function impossible to express, so the rule has a
motivation there that it lacks here.

## Risks / Trade-offs

- **Nested object variant values stay shared with the caller** → `maps.Clone` on the flag set and
  on each flag's variants is shallow, so a variant holding a `map[string]any` can still be
  mutated through the caller's reference. Documented on `NewFlagSet` as the caller handing over
  ownership of object values, which is the same contract the SDK's `memprovider` states. A deep
  copy would need a traversal of arbitrary `any` values and is not worth it before any real
  object flag exists.
- **Validation errors must name every offender, not the first** → Go map iteration order is
  random, so reporting only the first failure would make the error message non-deterministic and
  the test flaky. `NewFlagSet` accumulates violations and joins them, and the test asserts on a
  two-violation set to pin that down.
- **`TYPE_MISMATCH` is discoverable only at evaluation time** → inherent to inferring type from
  variant values, and unavoidable once the format is flagd's. Mitigated by making the message
  name the flag key, and by load-time homogeneity validation, which is what stops the type from
  varying per evaluation.
- **Narrowing three `openfeature-provider` requirements could silently drop coverage** → the
  delta keeps every existing scenario, narrowed rather than deleted, and adds one per requirement
  covering the newly reachable unresolvable outcomes. `openspec validate --strict` passes on the
  change.
- **A number too large for an integer evaluation has no correct resolution** → refused when the
  flag set is supplied, naming the flag, rather than losing precision silently on every
  evaluation. A value of a type the normalizing switch does not list is not an error: it becomes
  an object variant, so the switch carries no denylist to keep in step with Go.

## Testing

All tests are in the external package `launchpad_test`, table-driven, with no disk fixtures, no
sleeps and no network. Nothing here needs a clock.

| File | Covers | First failing test |
|---|---|---|
| `launchpad/flagset_test.go` (new) | `specs/flag-store/spec.md` — structure, homogeneity, dangling default variant, empty variants, accumulated violations, numeric category, clone-on-supply | `TestNewFlagSet_RejectsMixedTypeVariants` asserts `NewFlagSet` returns an error naming the flag; fails on a stub returning `(FlagSet{}, nil)` |
| `launchpad/provider_test.go` (existing) | `specs/flag-evaluation/spec.md` and the narrowed `openfeature-provider` requirements — per-type resolution, `DISABLED`, `TYPE_MISMATCH`, numeric widening, unknown key, variant reporting | `TestEvaluation_ResolvesEnabledFlag` asserts `true` with reason `STATIC` and variant `on`; fails against today's unconditional `flagNotFound` |

Per the project's TDD rule, each behavior gets its failing test first, with the minimal
declaration added so the test compiles and fails on its assertion rather than on a missing
symbol. The eight existing tests in `provider_test.go` must keep passing unchanged where they
assert no-flag behavior, since `NewProvider()` with no options is unchanged; the three that
assert "any flag key" behavior are narrowed alongside the spec.

Concurrency is asserted with a goroutine fan-out over a shared provider under `go test -race`,
which is already the project's gate, rather than with a timing-dependent test.
