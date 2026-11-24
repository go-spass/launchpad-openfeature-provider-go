# Spec Delta

## Purpose

Defines how a flag's value is decided: which variant a given evaluation selects, how a flag's
state affects the outcome, whether the selected value satisfies the type the caller asked for,
and which reason and error code the outcome carries. It owns the decision, while the
`openfeature-provider` capability owns only how that decision is reported to the SDK; targeting
and rollout will add requirements here without disturbing that reporting contract.

## ADDED Requirements

### Requirement: An enabled flag resolves to its default variant's value

When a requested flag key names an enabled flag, the evaluation SHALL return the value held by
that flag's default variant, reporting the `STATIC` reason and the selected variant's name.

#### Scenario: A boolean flag resolves to its default variant

- **GIVEN** a provider holding the enabled flag `new-checkout` with variants `on` holding
  `true` and `off` holding `false`, defaulting to `on`
- **WHEN** a consumer requests a boolean evaluation of `new-checkout` with default `false`
- **THEN** the returned value is `true`
- **AND** the reason is `STATIC` and the variant is `on`
- **AND** no error is reported

#### Scenario: Every value type resolves from its variants

- **GIVEN** a provider holding one enabled flag per value type — boolean, string, integer,
  float and object — each defaulting to a variant holding a known value
- **WHEN** a consumer requests the matching typed evaluation of each flag
- **THEN** each returns the value its default variant holds, with the `STATIC` reason and that
  variant's name

#### Scenario: The caller's default is not returned when a flag resolves

- **GIVEN** a provider holding the enabled string flag `greeting` whose default variant holds
  `"Good day"`
- **WHEN** a consumer requests a string evaluation of `greeting` with default `"fallback"`
- **THEN** the returned value is `"Good day"`, not `"fallback"`

### Requirement: A disabled flag returns the caller's default value

When a requested flag key names a flag whose state is disabled, the evaluation SHALL return the
default value the caller supplied, reporting the `DISABLED` reason, no error, and no variant,
because no variant was selected.

#### Scenario: A disabled flag reports DISABLED and the caller's default

- **GIVEN** a provider holding the flag `new-checkout`, state disabled, whose default variant
  holds `true`
- **WHEN** a consumer requests a boolean evaluation of `new-checkout` with default `false`
- **THEN** the returned value is `false`
- **AND** the reason is `DISABLED` and no error is reported
- **AND** the variant is the empty string

#### Scenario: Disabled is distinguishable from not found

- **GIVEN** a provider holding the disabled flag `new-checkout` and no flag named `absent`
- **WHEN** a consumer evaluates each key with the same default value
- **THEN** both return that default value
- **AND** `new-checkout` reports the `DISABLED` reason with no error, while `absent` reports the
  `ERROR` reason with the `FLAG_NOT_FOUND` error code

### Requirement: A type the resolved variant does not hold reports TYPE_MISMATCH

When a requested flag key resolves to a variant whose value is not of the type the caller
requested, the evaluation SHALL return the caller's default value and report the `TYPE_MISMATCH`
error code with the `ERROR` reason, rather than a zero value, a converted value or a panic.

#### Scenario: Requesting a string from a boolean flag is a type mismatch

- **GIVEN** a provider holding the enabled flag `new-checkout` whose default variant holds
  `true`
- **WHEN** a consumer requests a string evaluation of `new-checkout` with default `"fallback"`
- **THEN** the returned value is `"fallback"`
- **AND** the error code is `TYPE_MISMATCH` and the reason is `ERROR`

#### Scenario: The mismatch error names the flag key

- **WHEN** a consumer requests an evaluation whose type the resolved variant does not hold, for
  the flag key `new-checkout`
- **THEN** the resolution detail carries an error whose message includes `new-checkout`

#### Scenario: A mismatch is reported for every value type

- **GIVEN** a provider holding one enabled flag per value type, each defaulting to a variant
  holding a value of that type
- **WHEN** a consumer requests, for each flag, an evaluation of a type the flag does not hold
- **THEN** every such request returns the caller's default with the `TYPE_MISMATCH` error code

#### Scenario: An object evaluation accepts any resolved value

- **GIVEN** a provider holding the enabled flag `new-checkout` whose default variant holds
  `true`
- **WHEN** a consumer requests an object evaluation of `new-checkout`
- **THEN** the returned value is `true` with the `STATIC` reason, because an object evaluation
  constrains the value to no particular type

### Requirement: Integral variant values satisfy both numeric request types

A variant value that is a whole number SHALL resolve for both an integer request and a float
request. A variant value with a fractional part SHALL resolve for a float request and SHALL
report `TYPE_MISMATCH` for an integer request.

#### Scenario: A whole number resolves as an integer

- **GIVEN** a provider holding the enabled flag `max-items` whose default variant holds `50`
- **WHEN** a consumer requests an integer evaluation of `max-items` with default `10`
- **THEN** the returned value is `50` with the `STATIC` reason

#### Scenario: A whole number resolves as a float

- **GIVEN** a provider holding the enabled flag `max-items` whose default variant holds `50`
- **WHEN** a consumer requests a float evaluation of `max-items` with default `1.5`
- **THEN** the returned value is `50.0` with the `STATIC` reason

#### Scenario: A fractional number is not an integer

- **GIVEN** a provider holding the enabled flag `sample-rate` whose default variant holds `2.5`
- **WHEN** a consumer requests an integer evaluation of `sample-rate` with default `1`
- **THEN** the returned value is `1`
- **AND** the error code is `TYPE_MISMATCH`

#### Scenario: A fractional number resolves as a float

- **GIVEN** a provider holding the enabled flag `sample-rate` whose default variant holds `2.5`
- **WHEN** a consumer requests a float evaluation of `sample-rate` with default `1.0`
- **THEN** the returned value is `2.5` with the `STATIC` reason

### Requirement: An unknown flag key resolves as not found

When a requested flag key names no flag in the provider's flag set, the evaluation SHALL return
the caller's default value and report the `FLAG_NOT_FOUND` error code with the `ERROR` reason,
so that an absent flag stays distinguishable from one that resolved.

#### Scenario: An unknown key is not found even when other flags exist

- **GIVEN** a provider holding the enabled flag `new-checkout`
- **WHEN** a consumer requests a boolean evaluation of `absent` with default `true`
- **THEN** the returned value is `true`
- **AND** the error code is `FLAG_NOT_FOUND` and the reason is `ERROR`

#### Scenario: Flag keys are matched exactly

- **GIVEN** a provider holding the enabled flag `new-checkout`
- **WHEN** a consumer evaluates the key `New-Checkout`
- **THEN** the error code is `FLAG_NOT_FOUND`, because keys are compared exactly and not folded
