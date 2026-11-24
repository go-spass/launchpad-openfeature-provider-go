# Spec Delta

## MODIFIED Requirements

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
