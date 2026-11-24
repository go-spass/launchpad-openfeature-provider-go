# openfeature-provider Specification

## Purpose

Defines how LaunchPad presents itself to the OpenFeature Go SDK: the provider's identity, the
hooks it contributes, the typed evaluation entry points it exposes, and how an internal
resolution outcome is reported as the specification's value, variant, reason and error code.
Consumers reach LaunchPad only through this contract, so swapping LaunchPad for another provider
requires no change at their call sites.

## Requirements

### Requirement: Provider interface conformance

The provider SHALL satisfy the OpenFeature Go SDK's feature-provider interface in full, so a
consumer can supply it anywhere that interface is required without an adapter or wrapper.

#### Scenario: Provider is usable where the interface is required

- **GIVEN** a consumer holds a variable whose type is the SDK's feature-provider interface
- **WHEN** the consumer assigns a LaunchPad provider to that variable
- **THEN** the assignment is accepted and the module compiles

#### Scenario: A method removed from the implementation breaks the build

- **GIVEN** the module asserts its provider against the interface
- **WHEN** the implementation stops satisfying the interface, whether through an SDK upgrade or
  a local edit
- **THEN** the failure surfaces as a build error in this module, not as a runtime failure in a
  consumer

### Requirement: Provider identity

The provider SHALL report metadata carrying a stable, non-empty name that identifies LaunchPad
as the source of a flag evaluation.

#### Scenario: Name is non-empty

- **WHEN** a consumer reads the provider's metadata
- **THEN** the reported name is a non-empty string

#### Scenario: Name does not vary between reads or instances

- **GIVEN** two separately constructed LaunchPad providers
- **WHEN** a consumer reads the metadata of each, twice
- **THEN** all four reads report the same name

### Requirement: Hook contribution

The provider SHALL report an empty, usable collection of evaluation hooks, because it
contributes no hook behavior of its own.

#### Scenario: Hook collection is empty and safe to range over

- **WHEN** a consumer reads the provider's hooks
- **THEN** the collection contains no entries
- **AND** the collection can be iterated without a nil dereference

### Requirement: Typed evaluation coverage

The provider SHALL accept evaluation requests for every OpenFeature value type — boolean,
string, integer, float and object — and SHALL return a resolution detail for each.

#### Scenario: Every value type is answered

- **GIVEN** a LaunchPad provider and any flag key
- **WHEN** a consumer requests evaluation once for each of the five value types
- **THEN** each request returns a resolution detail
- **AND** no request panics or blocks

#### Scenario: A nil evaluation context is accepted

- **WHEN** a consumer requests an evaluation supplying no evaluation context
- **THEN** the request returns a resolution detail rather than failing

### Requirement: Unresolvable flag returns the caller's default value

When the provider cannot resolve a flag key to a value of the requested type, it SHALL return
the default value supplied by the caller, unmodified. This covers a key the provider does not
hold, a flag whose state is disabled, and a flag whose resolved variant does not hold the
requested type. A key that does resolve returns its own value instead; which keys resolve is
owned by the `flag-evaluation` capability.

#### Scenario: Boolean default passes through unchanged

- **WHEN** a consumer requests a boolean evaluation of a flag key the provider does not hold,
  with default `true`
- **THEN** the returned value is `true`
- **AND** the same request with default `false` returns `false`

#### Scenario: Defaults of every other type pass through unchanged

- **WHEN** a consumer requests a string, integer, float or object evaluation of a flag key the
  provider does not hold
- **THEN** the returned value equals the default the consumer supplied for that request

#### Scenario: The default passes through unchanged for every unresolvable outcome

- **GIVEN** a provider holding a disabled flag, and a boolean flag whose variants hold booleans
- **WHEN** a consumer evaluates the disabled flag, and requests a string evaluation of the
  boolean flag
- **THEN** each returns the default value that consumer supplied for that request, unmodified

### Requirement: Unresolvable flag reports FLAG_NOT_FOUND

When the provider holds no flag under the requested key, the resolution detail SHALL carry the
`FLAG_NOT_FOUND` error code and the reason `ERROR`, so a consumer can distinguish a missing
flag from a flag that deliberately evaluated to the default value. An unresolvable flag the
provider does hold reports its own error code instead, never `FLAG_NOT_FOUND`.

#### Scenario: Error code and reason are reported together

- **WHEN** a consumer requests an evaluation of a flag key the provider does not hold, for any
  value type
- **THEN** the resolution detail's error code is `FLAG_NOT_FOUND`
- **AND** the resolution detail's reason is `ERROR`

#### Scenario: The error message names the flag key

- **WHEN** a consumer requests an evaluation of the flag key `missing-flag`
- **THEN** the resolution detail carries an error whose message includes `missing-flag`

#### Scenario: A held flag never reports FLAG_NOT_FOUND

- **GIVEN** a provider holding a disabled flag, and a boolean flag whose variants hold booleans
- **WHEN** a consumer evaluates the disabled flag, and requests a string evaluation of the
  boolean flag
- **THEN** neither resolution detail carries the `FLAG_NOT_FOUND` error code

### Requirement: No variant is reported when no flag resolves

The provider SHALL report an empty variant whenever no variant was selected, and SHALL report
the selected variant's name whenever one was. No variant is selected for a key the provider does
not hold, for a disabled flag, or for a flag whose resolved variant does not hold the requested
type.

#### Scenario: Variant is empty on a missing flag

- **WHEN** a consumer requests an evaluation of a flag key the provider does not hold, for any
  value type
- **THEN** the resolution detail's variant is the empty string

#### Scenario: Variant is empty whenever no variant was selected

- **GIVEN** a provider holding a disabled flag, and a boolean flag whose variants hold booleans
- **WHEN** a consumer evaluates the disabled flag, and requests a string evaluation of the
  boolean flag
- **THEN** each resolution detail's variant is the empty string

#### Scenario: The selected variant is named when one was selected

- **GIVEN** a provider holding an enabled flag whose default variant is named `on`
- **WHEN** a consumer requests an evaluation of that flag, of the type its variants hold
- **THEN** the resolution detail's variant is `on`

### Requirement: End-to-end evaluation through an OpenFeature client

A consumer SHALL be able to register the provider with the OpenFeature SDK and evaluate a flag
through an SDK client, receiving the supplied default value together with an error identifying
the flag as not found.

#### Scenario: Registered provider answers a client evaluation

- **GIVEN** a LaunchPad provider registered with the OpenFeature SDK
- **WHEN** a consumer evaluates the boolean flag `missing-flag` through an SDK client with
  default `true`
- **THEN** the client returns `true`
- **AND** the client reports an error identifying the flag as not found

#### Scenario: Evaluation details reach the consumer through the client

- **GIVEN** a LaunchPad provider registered with the OpenFeature SDK
- **WHEN** a consumer requests evaluation details rather than a bare value
- **THEN** the details report the provider's name as the source
- **AND** the details report the reason `ERROR` and the error code `FLAG_NOT_FOUND`
