package launchpad_test

import (
	"context"
	"math"
	"strings"
	"testing"

	"github.com/go-spass/launchpad-openfeature-provider-go/launchpad"
	"github.com/open-feature/go-sdk/openfeature"
)

// The flag below is deliberately malformed, and in the way a hand-written flag
// definition really goes wrong: "off" holds the string "disabled" where a
// boolean was meant. Left alone, the flag would have no single type — selecting
// "on" would serve a boolean and selecting "off" would report TYPE_MISMATCH —
// so which one a caller got would decide whether the flag worked.
func TestNewFlagSet_RejectsMixedTypeVariants(t *testing.T) {
	t.Parallel()

	_, err := launchpad.NewFlagSet(map[string]launchpad.Flag{
		"new-checkout": {
			State:          launchpad.Enabled,
			DefaultVariant: "on",
			Variants:       map[string]any{"on": true, "off": "disabled"}, // want false
		},
	})

	if err == nil {
		t.Fatal("NewFlagSet() error = nil, want an error for mixed-type variants")
	}
	if !strings.Contains(err.Error(), "new-checkout") {
		t.Errorf("NewFlagSet() error = %q, want it to name the flag key %q", err, "new-checkout")
	}
}

func TestNewFlagSet_AcceptsWellFormedSet(t *testing.T) {
	t.Parallel()

	tests := map[string]map[string]launchpad.Flag{
		"two flags of different value types in one set": {
			"new-checkout": {
				State:          launchpad.Enabled,
				DefaultVariant: "on",
				Variants:       map[string]any{"on": true, "off": false},
			},
			"greeting": {
				State:          launchpad.Enabled,
				DefaultVariant: "formal",
				Variants:       map[string]any{"formal": "Good day", "casual": "Hey"},
			},
		},
		"a single-variant flag": {
			"new-checkout": {
				State:          launchpad.Enabled,
				DefaultVariant: "on",
				Variants:       map[string]any{"on": true},
			},
		},
		"an empty set": {},
	}

	for name, flags := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := launchpad.NewFlagSet(flags); err != nil {
				t.Errorf("NewFlagSet() error = %v, want nil", err)
			}
		})
	}
}

func TestNewFlagSet_TreatsEveryNumberAsOneType(t *testing.T) {
	t.Parallel()

	tests := map[string]map[string]any{
		"whole and fractional together": {"standard": 50, "precise": 2.5},
		"mixed Go numeric widths":       {"a": int8(1), "b": uint64(2), "c": float32(2.5)},
	}

	for name, variants := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := launchpad.NewFlagSet(map[string]launchpad.Flag{
				"max-items": {
					State:          launchpad.Enabled,
					DefaultVariant: "standard",
					Variants:       withVariant(variants, "standard", 50),
				},
			})
			if err != nil {
				t.Errorf("NewFlagSet() error = %v, want nil", err)
			}
		})
	}
}

// withVariant returns variants with name set to value, so a table case can
// supply the variants it cares about and still have a valid default variant.
func withVariant(variants map[string]any, name string, value any) map[string]any {
	out := make(map[string]any, len(variants)+1)
	for k, v := range variants {
		out[k] = v
	}
	out[name] = value

	return out
}

func TestNewFlagSet_RejectsDanglingDefaultVariant(t *testing.T) {
	t.Parallel()

	_, err := launchpad.NewFlagSet(map[string]launchpad.Flag{
		"new-checkout": {
			State:          launchpad.Enabled,
			DefaultVariant: "enabled",
			Variants:       map[string]any{"on": true, "off": false},
		},
	})

	if err == nil {
		t.Fatal("NewFlagSet() error = nil, want an error for a dangling default variant")
	}
	for _, want := range []string{"new-checkout", "enabled"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("NewFlagSet() error = %q, want it to name %q", err, want)
		}
	}
}

func TestNewFlagSet_RejectsFlagWithNoVariants(t *testing.T) {
	t.Parallel()

	_, err := launchpad.NewFlagSet(map[string]launchpad.Flag{
		"new-checkout": {State: launchpad.Enabled, DefaultVariant: "on"},
	})

	if err == nil {
		t.Fatal("NewFlagSet() error = nil, want an error for a flag with no variants")
	}
	if !strings.Contains(err.Error(), "new-checkout") {
		t.Errorf("NewFlagSet() error = %q, want it to name the flag key %q", err, "new-checkout")
	}
}

// Map iteration order is random, so an error that reported only the first
// violation it met would make this assertion flaky. Both keys must appear.
func TestNewFlagSet_ReportsEveryViolation(t *testing.T) {
	t.Parallel()

	_, err := launchpad.NewFlagSet(map[string]launchpad.Flag{
		"mixed-variants": {
			State:          launchpad.Enabled,
			DefaultVariant: "on",
			Variants:       map[string]any{"on": true, "off": "disabled"}, // want false
		},
		"dangling-default": {
			State:          launchpad.Enabled,
			DefaultVariant: "missing",
			Variants:       map[string]any{"on": true},
		},
	})

	if err == nil {
		t.Fatal("NewFlagSet() error = nil, want an error naming both offending flags")
	}
	for _, want := range []string{"mixed-variants", "dangling-default"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("NewFlagSet() error = %q, want it to name %q", err, want)
		}
	}
}

func TestNewFlagSet_IgnoresLaterMutationOfTheCallersMap(t *testing.T) {
	t.Parallel()

	variants := map[string]any{"on": true, "off": false}
	flags := map[string]launchpad.Flag{
		"new-checkout": {State: launchpad.Enabled, DefaultVariant: "on", Variants: variants},
	}

	set, err := launchpad.NewFlagSet(flags)
	if err != nil {
		t.Fatalf("NewFlagSet() error = %v, want nil", err)
	}

	delete(flags, "new-checkout")
	variants["on"] = false

	provider := launchpad.NewProvider(launchpad.WithFlags(set))
	got := provider.BooleanEvaluation(context.Background(), "new-checkout", false, nil)

	if !got.Value {
		t.Errorf("BooleanEvaluation() value = %v, want true: the supplied set must not change", got.Value)
	}
	if got.Variant != "on" {
		t.Errorf("BooleanEvaluation() variant = %q, want %q", got.Variant, "on")
	}
}

// Normalization is only observable through evaluation, so each case asserts the
// value an integer request returns. Every accepted Go numeric type is listed,
// so an arm dropped from the switch shows up here rather than as a mismatch in
// a consumer.
func TestNewFlagSet_NormalizesEveryAcceptedNumericType(t *testing.T) {
	t.Parallel()

	whole := map[string]any{
		"int":     int(7),
		"int8":    int8(7),
		"int16":   int16(7),
		"int32":   int32(7),
		"int64":   int64(7),
		"uint":    uint(7),
		"uint8":   uint8(7),
		"uint16":  uint16(7),
		"uint32":  uint32(7),
		"uint64":  uint64(7),
		"float32": float32(7),
		"float64": float64(7),
	}

	for name, value := range whole {
		t.Run("whole "+name, func(t *testing.T) {
			t.Parallel()

			set, err := launchpad.NewFlagSet(map[string]launchpad.Flag{
				"max-items": {
					State:          launchpad.Enabled,
					DefaultVariant: "only",
					Variants:       map[string]any{"only": value},
				},
			})
			if err != nil {
				t.Fatalf("NewFlagSet() error = %v, want nil", err)
			}

			provider := launchpad.NewProvider(launchpad.WithFlags(set))
			if got := provider.IntEvaluation(context.Background(), "max-items", 0, nil); got.Value != 7 {
				t.Errorf("IntEvaluation() value = %d, want 7", got.Value)
			}
			if got := provider.FloatEvaluation(context.Background(), "max-items", 0, nil); got.Value != 7 {
				t.Errorf("FloatEvaluation() value = %v, want 7", got.Value)
			}
		})
	}

	t.Run("fractional float32 stays fractional", func(t *testing.T) {
		t.Parallel()

		set, err := launchpad.NewFlagSet(map[string]launchpad.Flag{
			"sample-rate": {
				State:          launchpad.Enabled,
				DefaultVariant: "only",
				Variants:       map[string]any{"only": float32(2.5)},
			},
		})
		if err != nil {
			t.Fatalf("NewFlagSet() error = %v, want nil", err)
		}

		provider := launchpad.NewProvider(launchpad.WithFlags(set))
		if got := provider.FloatEvaluation(context.Background(), "sample-rate", 0, nil); got.Value != 2.5 {
			t.Errorf("FloatEvaluation() value = %v, want 2.5", got.Value)
		}
		if got := provider.IntEvaluation(context.Background(), "sample-rate", 1, nil); got.Value != 1 {
			t.Errorf("IntEvaluation() value = %d, want the caller's default 1", got.Value)
		}
	})

	t.Run("an unsigned value too large for an integer evaluation is rejected", func(t *testing.T) {
		t.Parallel()

		_, err := launchpad.NewFlagSet(map[string]launchpad.Flag{
			"max-items": {
				State:          launchpad.Enabled,
				DefaultVariant: "only",
				Variants:       map[string]any{"only": uint64(math.MaxUint64)},
			},
		})
		if err == nil {
			t.Fatal("NewFlagSet() error = nil, want an error for a value above the int64 range")
		}
		if !strings.Contains(err.Error(), "max-items") {
			t.Errorf("NewFlagSet() error = %q, want it to name the flag key", err)
		}
	})

	t.Run("a float too large for an integer evaluation stays a float", func(t *testing.T) {
		t.Parallel()

		set, err := launchpad.NewFlagSet(map[string]launchpad.Flag{
			"huge": {
				State:          launchpad.Enabled,
				DefaultVariant: "only",
				Variants:       map[string]any{"only": math.MaxFloat64},
			},
		})
		if err != nil {
			t.Fatalf("NewFlagSet() error = %v, want nil", err)
		}

		provider := launchpad.NewProvider(launchpad.WithFlags(set))
		if got := provider.FloatEvaluation(context.Background(), "huge", 0, nil); got.Value != math.MaxFloat64 {
			t.Errorf("FloatEvaluation() value = %v, want %v", got.Value, math.MaxFloat64)
		}
		if got := provider.IntEvaluation(context.Background(), "huge", 1, nil); got.Value != 1 {
			t.Errorf("IntEvaluation() value = %d, want the caller's default 1", got.Value)
		}
	})
}

// Only numbers are range-checked. A value of any other type is an object
// variant, because an object evaluation constrains the value to no particular
// type — so there is no denylist of exotic types to keep in step with Go.
func TestNewFlagSet_AcceptsNonNumericValueOfAnyType(t *testing.T) {
	t.Parallel()

	tests := map[string]any{
		"a channel":  make(chan int),
		"a function": func() {},
		"a complex":  complex128(1 + 2i),
		"a uintptr":  uintptr(42),
	}

	for name, value := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			set, err := launchpad.NewFlagSet(map[string]launchpad.Flag{
				"exotic": {
					State:          launchpad.Enabled,
					DefaultVariant: "only",
					Variants:       map[string]any{"only": value},
				},
			})
			if err != nil {
				t.Fatalf("NewFlagSet() error = %v, want nil", err)
			}

			got := launchpad.NewProvider(launchpad.WithFlags(set)).
				ObjectEvaluation(context.Background(), "exotic", nil, nil)

			if got.Variant != "only" {
				t.Errorf("ObjectEvaluation() variant = %q, want %q", got.Variant, "only")
			}
			if got.ResolutionError != (openfeature.ResolutionError{}) {
				t.Errorf("ObjectEvaluation() error = %v, want no error", got.ResolutionError)
			}
		})
	}
}
