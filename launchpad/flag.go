package launchpad

import (
	"errors"
	"fmt"
	"maps"
	"math"
	"slices"
)

// State says whether a flag participates in evaluation. A flag that is not
// Enabled resolves to the caller's default value with the DISABLED reason,
// which stays distinguishable from a flag the provider does not hold.
type State string

const (
	Enabled  State = "ENABLED"
	Disabled State = "DISABLED"
)

// Flag is the definition of one feature flag: its state, its named variants,
// and the variant selected when nothing else selects one.
//
// A flag does not declare its value type. The variant values carry it, so a
// flag whose variants hold booleans is a boolean flag. Asking for a type the
// resolved variant does not hold is a TYPE_MISMATCH at evaluation time, which
// is why [NewFlagSet] rejects variants that disagree about their type.
type Flag struct {
	// State says whether the flag participates in evaluation. The zero value
	// is not Enabled, so a Flag literal that omits it is disabled.
	State State

	// DefaultVariant names the variant to select. It must name one of
	// Variants; [NewFlagSet] rejects a flag where it does not.
	DefaultVariant string

	// Variants maps a variant name to the value that variant holds. It must
	// be non-empty, and every value must be of the same type.
	Variants map[string]any
}

// FlagSet is a set of flag definitions that is well-formed by construction:
// the only way to obtain a non-empty one is [NewFlagSet], which validates it.
// Pass one to [WithFlags] to build a provider that serves it.
//
// The zero FlagSet is a valid empty set, so a caller who ignores an error from
// [NewFlagSet] gets a provider that resolves nothing rather than one holding
// half-checked flags.
type FlagSet struct {
	flags map[string]Flag
}

// category groups variant values by the type a flag's variants must agree on.
// Every number is one category: the integer-versus-float distinction is
// reconciled when the flag is evaluated, not when it is defined.
type category string

const (
	categoryBool   category = "boolean"
	categoryString category = "string"
	categoryNumber category = "number"
	categoryObject category = "object"
)

func categoryOf(value any) category {
	switch value.(type) {
	case bool:
		return categoryBool
	case string:
		return categoryString
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return categoryNumber
	default:
		return categoryObject
	}
}

// normalizeValue returns the canonical form of a variant value: every whole
// number becomes an int64 and every number with a fractional part becomes a
// float64, so an evaluation needs one type assertion rather than a search
// through every Go numeric type a caller or a decoder might have produced.
//
// Values that are not one of these numeric types are returned unchanged, and
// are categorised as objects: an object evaluation constrains the value to no
// particular type, so there is nothing to normalize. A whole number outside the
// range an integer evaluation can return is the one numeric value with no
// correct resolution, and is rejected rather than silently losing precision.
func normalizeValue(value any) (any, error) {
	switch v := value.(type) {
	case int:
		return int64(v), nil
	case int8:
		return int64(v), nil
	case int16:
		return int64(v), nil
	case int32:
		return int64(v), nil
	case int64:
		return v, nil
	case uint:
		return normalizeUint(uint64(v))
	case uint8:
		return int64(v), nil
	case uint16:
		return int64(v), nil
	case uint32:
		return int64(v), nil
	case uint64:
		return normalizeUint(v)
	case float32:
		return normalizeFloat(float64(v)), nil
	case float64:
		return normalizeFloat(v), nil
	default:
		return value, nil
	}
}

func normalizeUint(v uint64) (any, error) {
	if v > math.MaxInt64 {
		return nil, fmt.Errorf("holds %d, which exceeds the largest integer an evaluation can return", v)
	}

	return int64(v), nil
}

// normalizeFloat collapses a whole-valued float onto int64 so it satisfies an
// integer request as well as a float one. NaN and the infinities are not whole
// and are left as they are.
func normalizeFloat(v float64) any {
	if math.Trunc(v) == v && v >= math.MinInt64 && v <= math.MaxInt64 {
		return int64(v)
	}

	return v
}

// normalized returns the flag with every variant value in canonical form, or
// the first invariant it violates. The variant checks run before the
// default-variant check, because a flag with no variants cannot have a default
// variant that names one and reporting both would say the same thing twice.
func (f Flag) normalized() (Flag, error) {
	if len(f.Variants) == 0 {
		return Flag{}, errors.New("has no variants")
	}

	variants := make(map[string]any, len(f.Variants))
	var first category
	var firstName string
	for _, name := range slices.Sorted(maps.Keys(f.Variants)) {
		value, err := normalizeValue(f.Variants[name])
		if err != nil {
			return Flag{}, fmt.Errorf("variant %q %w", name, err)
		}
		variants[name] = value

		got := categoryOf(value)
		if firstName == "" {
			first, firstName = got, name
			continue
		}
		if got != first {
			return Flag{}, fmt.Errorf(
				"variants %q and %q hold different types (%s and %s)",
				firstName, name, first, got,
			)
		}
	}

	if _, ok := variants[f.DefaultVariant]; !ok {
		return Flag{}, fmt.Errorf("default variant %q names no variant of this flag", f.DefaultVariant)
	}

	return Flag{State: f.State, DefaultVariant: f.DefaultVariant, Variants: variants}, nil
}

// NewFlagSet validates a set of flag definitions, keyed by flag key, and
// returns it as a value that is well-formed by construction.
//
// A flag is rejected when it has no variants, when its variants hold more than
// one type, or when its default variant names no variant of that flag. Every
// violation in the set is reported, in flag-key order, so the error does not
// depend on Go's map iteration order. Checking here rather than at evaluation
// time means a malformed definition is a startup error instead of a per-caller
// surprise.
//
// Every number is one type for the type check: a flag holding both 50 and 2.5
// is a well-formed numeric flag, and the integer-versus-float distinction is
// reconciled when the flag is evaluated.
//
// The returned set holds its own copy of the map and of each flag's variants,
// so mutating what was passed in afterwards does not change what the provider
// serves. The copy of a variant's value is not deep: a variant holding a map or
// a slice is still shared with the caller, who hands over ownership of it.
func NewFlagSet(from map[string]Flag) (FlagSet, error) {
	var errs []error
	flags := make(map[string]Flag, len(from))
	for _, key := range slices.Sorted(maps.Keys(from)) {
		flag, err := from[key].normalized()
		if err != nil {
			errs = append(errs, fmt.Errorf("flag %q: %w", key, err))

			continue
		}
		flags[key] = flag
	}
	if len(errs) > 0 {
		return FlagSet{}, errors.Join(errs...)
	}

	return FlagSet{flags: flags}, nil
}
