# Spec Delta

## Purpose

Defines what a LaunchPad flag definition is and what a well-formed set of definitions looks
like: the key, the enabled state, the named variants that carry the flag's value and type, and
the variant selected when nothing else selects one. It owns the invariants checked when a flag
set is supplied and the guarantee that a supplied set cannot change under a running provider, so
these hold identically whether the set came from Go code, a file or a sync source.

## ADDED Requirements

### Requirement: Flag definition structure

A flag definition SHALL consist of an enabled state, a set of named variants mapping a variant
name to a value, and the name of the variant to select when nothing else selects one. A flag
SHALL NOT declare its value type separately; the variant values carry it.

#### Scenario: A flag is defined by its variants and default variant

- **GIVEN** a flag whose variants are `on` holding `true` and `off` holding `false`, with `off`
  as its default variant
- **WHEN** that flag is supplied to the provider under the key `new-checkout`
- **THEN** the flag set is accepted

#### Scenario: Two flags of different types coexist in one set

- **GIVEN** one flag whose variants hold boolean values and another whose variants hold strings
- **WHEN** both are supplied in the same flag set
- **THEN** the flag set is accepted, because type is a property of a flag and not of the set

### Requirement: Variants of one flag hold a single value type

A flag's variants SHALL all hold values of the same type. A flag set containing a flag with
mixed-type variants SHALL be rejected, so that a flag's type cannot depend on which variant a
given evaluation selects.

#### Scenario: Mixed-type variants are rejected

- **GIVEN** a flag whose variants are `on` holding `true` and `off` holding the string
  `"disabled"`
- **WHEN** that flag set is supplied
- **THEN** supplying it fails with an error
- **AND** the error names the flag key whose variants are mixed

#### Scenario: A single-variant flag is well-formed

- **GIVEN** a flag with exactly one variant
- **WHEN** that flag set is supplied
- **THEN** the flag set is accepted, because one variant cannot disagree with itself

#### Scenario: Whole and fractional numbers are one type

- **GIVEN** a flag whose variants are `standard` holding `50` and `precise` holding `2.5`
- **WHEN** that flag set is supplied
- **THEN** the flag set is accepted, because every number is one type for this purpose and a
  request's numeric type is reconciled when the flag is evaluated

### Requirement: A numeric variant value must be representable by a numeric evaluation

A variant value that is a number SHALL be rejected unless an integer or a float evaluation can
return it without losing information. A whole number beyond the range an integer evaluation can
return is the one numeric value with no correct resolution, so it is refused when the flag set is
supplied rather than losing precision at every evaluation.

#### Scenario: A whole number beyond the integer range is rejected

- **GIVEN** a flag whose only variant holds an unsigned whole number larger than the largest
  value an integer evaluation can return
- **WHEN** that flag set is supplied
- **THEN** supplying it fails with an error
- **AND** the error names the flag key

#### Scenario: A number beyond the integer range but within the float range is accepted

- **GIVEN** a flag whose only variant holds a number larger than any whole number an integer
  evaluation can return, but within the range a float evaluation can return
- **WHEN** that flag set is supplied
- **THEN** the flag set is accepted, because a float evaluation can still return it

#### Scenario: A value that is not a number is not subject to the numeric range

- **GIVEN** a flag whose only variant holds a value of a type no format could serialize, such as
  a channel or a function
- **WHEN** that flag set is supplied
- **THEN** the flag set is accepted and the value is treated as an object variant, because an
  object evaluation constrains the value to no particular type

### Requirement: The default variant names an existing variant

A flag's default variant SHALL name one of that flag's variants. A flag set containing a flag
whose default variant names no existing variant SHALL be rejected.

#### Scenario: A dangling default variant is rejected

- **GIVEN** a flag whose variants are `on` and `off` and whose default variant is `enabled`
- **WHEN** that flag set is supplied
- **THEN** supplying it fails with an error
- **AND** the error names both the flag key and the missing variant name

#### Scenario: A flag with no variants is rejected

- **GIVEN** a flag whose variant set is empty
- **WHEN** that flag set is supplied
- **THEN** supplying it fails with an error, because no default variant can name an existing
  variant

### Requirement: Invariants are reported at supply time, not at evaluation time

A flag set that violates any invariant SHALL be rejected when it is supplied, before any
evaluation occurs. A provider SHALL NOT be constructed holding a flag set that was rejected.

#### Scenario: Every violation in a set is reported together

- **GIVEN** a flag set containing one flag with mixed-type variants and another with a dangling
  default variant
- **WHEN** that flag set is supplied
- **THEN** supplying it fails with an error naming both offending flag keys, rather than only
  the first one encountered

#### Scenario: An evaluation never reports an invariant violation

- **GIVEN** a flag set that violates an invariant
- **WHEN** the consumer ignores the supply error and evaluates any flag key
- **THEN** no resolution detail reports the invariant violation, because the provider never
  held the rejected set

### Requirement: A supplied flag set cannot change under a running provider

The provider SHALL resolve against the flag set as it was at supply time. A caller that mutates
the collection it supplied, after supplying it, SHALL NOT change what any evaluation returns.

#### Scenario: Mutating the supplied collection does not affect evaluation

- **GIVEN** a provider supplied with a flag set containing the enabled boolean flag
  `new-checkout` whose default variant holds `true`
- **WHEN** the caller removes `new-checkout` from the collection it passed in, and then
  evaluates `new-checkout`
- **THEN** the evaluation still returns `true`

#### Scenario: Concurrent evaluations are safe

- **GIVEN** a provider holding a flag set
- **WHEN** many goroutines evaluate flags from that set at the same time
- **THEN** every evaluation returns the value its flag defines
- **AND** the race detector reports no data race

### Requirement: A provider may hold no flags

Supplying no flag set SHALL be valid and SHALL produce a provider that holds an empty flag set,
so that constructing a provider without flags remains possible.

#### Scenario: A provider constructed with no flags resolves nothing

- **WHEN** a consumer constructs a provider without supplying any flag set
- **THEN** construction succeeds
- **AND** every flag key evaluates as not found

#### Scenario: An explicitly empty flag set is accepted

- **WHEN** a consumer supplies an empty flag set
- **THEN** the flag set is accepted, because emptiness violates no invariant
