package launchpad_test

import (
	"context"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/go-spass/launchpad-openfeature-provider-go/launchpad"
	"github.com/open-feature/go-sdk/openfeature"
)

func TestProviderSatisfiesFeatureProvider(t *testing.T) {
	t.Parallel()

	// The assignment is the assertion: this does not compile unless
	// *launchpad.Provider implements every method of the interface. Reading
	// metadata back through the interface value proves it is usable there, not
	// merely assignable.
	var provider openfeature.FeatureProvider = launchpad.NewProvider()

	if got := provider.Metadata().Name; got != launchpad.ProviderName {
		t.Errorf("name read through the interface = %q, want %q", got, launchpad.ProviderName)
	}
}

func TestMetadata(t *testing.T) {
	t.Parallel()

	first, second := launchpad.NewProvider(), launchpad.NewProvider()

	name := first.Metadata().Name
	if name == "" {
		t.Fatal("Metadata().Name is empty; the provider must report a name")
	}

	reads := []openfeature.Metadata{
		first.Metadata(), first.Metadata(),
		second.Metadata(), second.Metadata(),
	}
	for i, got := range reads {
		if got.Name != name {
			t.Errorf("read %d: Metadata().Name = %q, want %q (the name must not vary)", i, got.Name, name)
		}
	}
}

func TestHooks(t *testing.T) {
	t.Parallel()

	hooks := launchpad.NewProvider().Hooks()

	if hooks == nil {
		t.Error("Hooks() returned nil; it must return an empty, non-nil slice")
	}

	if len(hooks) != 0 {
		t.Errorf("len(Hooks()) = %d, want 0; the provider contributes no hooks", len(hooks))
	}

	for range hooks {
		t.Error("Hooks() yielded an entry; the slice must be empty")
	}
}

func TestEvaluation_ReturnsCallerDefault(t *testing.T) {
	t.Parallel()

	const flag = "missing-flag"

	t.Run("boolean true", func(t *testing.T) {
		t.Parallel()
		got := launchpad.NewProvider().BooleanEvaluation(context.Background(), flag, true, nil)
		if !got.Value {
			t.Errorf("BooleanEvaluation value = %v, want true", got.Value)
		}
	})

	t.Run("boolean false", func(t *testing.T) {
		t.Parallel()
		got := launchpad.NewProvider().BooleanEvaluation(context.Background(), flag, false, nil)
		if got.Value {
			t.Errorf("BooleanEvaluation value = %v, want false", got.Value)
		}
	})

	t.Run("string", func(t *testing.T) {
		t.Parallel()
		got := launchpad.NewProvider().StringEvaluation(context.Background(), flag, "fallback", nil)
		if got.Value != "fallback" {
			t.Errorf("StringEvaluation value = %q, want %q", got.Value, "fallback")
		}
	})

	t.Run("int", func(t *testing.T) {
		t.Parallel()
		got := launchpad.NewProvider().IntEvaluation(context.Background(), flag, 42, nil)
		if got.Value != 42 {
			t.Errorf("IntEvaluation value = %d, want 42", got.Value)
		}
	})

	t.Run("float", func(t *testing.T) {
		t.Parallel()
		got := launchpad.NewProvider().FloatEvaluation(context.Background(), flag, 3.5, nil)
		if got.Value != 3.5 {
			t.Errorf("FloatEvaluation value = %v, want 3.5", got.Value)
		}
	})

	t.Run("object", func(t *testing.T) {
		t.Parallel()
		want := map[string]any{"tier": "gold"}
		got := launchpad.NewProvider().ObjectEvaluation(context.Background(), flag, want, nil)
		if !reflect.DeepEqual(got.Value, want) {
			t.Errorf("ObjectEvaluation value = %#v, want %#v", got.Value, want)
		}
	})
}

// missingFlag is the flag key every evaluation test asks for. The provider holds
// no flags, so no key resolves and this one stands in for all of them.
const missingFlag = "missing-flag"

// evaluations drives the assertions that are identical across value types. Each
// entry discards the typed value and returns only the shared half of the
// resolution detail, so one table can cover all five evaluation methods. A sixth
// value type added by a future SDK shows up here as a missing entry.
func evaluations() map[string]func(*launchpad.Provider) openfeature.ProviderResolutionDetail {
	ctx := context.Background()

	return map[string]func(*launchpad.Provider) openfeature.ProviderResolutionDetail{
		"boolean": func(p *launchpad.Provider) openfeature.ProviderResolutionDetail {
			return p.BooleanEvaluation(ctx, missingFlag, true, nil).ProviderResolutionDetail
		},
		"string": func(p *launchpad.Provider) openfeature.ProviderResolutionDetail {
			return p.StringEvaluation(ctx, missingFlag, "fallback", nil).ProviderResolutionDetail
		},
		"int": func(p *launchpad.Provider) openfeature.ProviderResolutionDetail {
			return p.IntEvaluation(ctx, missingFlag, 42, nil).ProviderResolutionDetail
		},
		"float": func(p *launchpad.Provider) openfeature.ProviderResolutionDetail {
			return p.FloatEvaluation(ctx, missingFlag, 3.5, nil).ProviderResolutionDetail
		},
		"object": func(p *launchpad.Provider) openfeature.ProviderResolutionDetail {
			return p.ObjectEvaluation(ctx, missingFlag, map[string]any{"tier": "gold"}, nil).ProviderResolutionDetail
		},
	}
}

func TestEvaluation_ReportsFlagNotFound(t *testing.T) {
	t.Parallel()

	for name, evaluate := range evaluations() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := evaluate(launchpad.NewProvider())

			if got.Reason != openfeature.ErrorReason {
				t.Errorf("Reason = %q, want %q", got.Reason, openfeature.ErrorReason)
			}

			err := got.Error()
			if err == nil {
				t.Fatal("resolution detail carries no error; a missing flag must report one")
			}

			if code := got.ResolutionDetail().ErrorCode; code != openfeature.FlagNotFoundCode {
				t.Errorf("error code = %q, want %q", code, openfeature.FlagNotFoundCode)
			}

			if !strings.Contains(err.Error(), missingFlag) {
				t.Errorf("error message %q does not name the flag key %q", err.Error(), missingFlag)
			}
		})
	}
}

func TestEvaluation_ReportsNoVariant(t *testing.T) {
	t.Parallel()

	for name, evaluate := range evaluations() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := evaluate(launchpad.NewProvider()); got.Variant != "" {
				t.Errorf("Variant = %q, want empty; no variant was selected", got.Variant)
			}
		})
	}
}

func TestEvaluation_AllTypesAnswered(t *testing.T) {
	t.Parallel()

	table := evaluations()

	if len(table) != 5 {
		t.Fatalf("the evaluation table covers %d value types, want 5 (boolean, string, int, float, object)", len(table))
	}

	for name, evaluate := range table {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := evaluate(launchpad.NewProvider())

			if got.Reason == "" {
				t.Error("evaluation returned a detail with no reason; every answer must carry one")
			}
		})
	}

	t.Run("nil evaluation context", func(t *testing.T) {
		t.Parallel()

		provider := launchpad.NewProvider()
		ctx := context.Background()

		var flatCtx openfeature.FlattenedContext

		if got := provider.BooleanEvaluation(ctx, missingFlag, true, flatCtx); !got.Value {
			t.Error("BooleanEvaluation with a nil context did not return the default")
		}

		if got := provider.StringEvaluation(ctx, missingFlag, "fallback", flatCtx); got.Value != "fallback" {
			t.Error("StringEvaluation with a nil context did not return the default")
		}

		if got := provider.IntEvaluation(ctx, missingFlag, 42, flatCtx); got.Value != 42 {
			t.Error("IntEvaluation with a nil context did not return the default")
		}

		if got := provider.FloatEvaluation(ctx, missingFlag, 3.5, flatCtx); got.Value != 3.5 {
			t.Error("FloatEvaluation with a nil context did not return the default")
		}

		if got := provider.ObjectEvaluation(ctx, missingFlag, "fallback", flatCtx); got.Value != "fallback" {
			t.Error("ObjectEvaluation with a nil context did not return the default")
		}
	})
}

// TestClientEvaluation_MissingFlag exercises the provider the way a consumer
// does: registered with the SDK and reached through a client. It registers under
// a named domain rather than the default provider slot, because registration
// writes SDK-global state and the default slot would make this test clobber, and
// depend on, whatever else has registered. It must not run in parallel for the
// same reason.
func TestClientEvaluation_MissingFlag(t *testing.T) {
	const domain = "launchpad-test"

	t.Cleanup(openfeature.Shutdown)

	if err := openfeature.SetNamedProviderAndWait(domain, launchpad.NewProvider()); err != nil {
		t.Fatalf("SetNamedProviderAndWait: %v", err)
	}

	client := openfeature.NewClient(domain)
	ctx := context.Background()

	value, err := client.BooleanValue(ctx, missingFlag, true, openfeature.EvaluationContext{})
	if !value {
		t.Errorf("BooleanValue = %v, want true (the caller's default)", value)
	}

	if err == nil {
		t.Fatal("BooleanValue returned no error; a missing flag must surface one")
	}

	if !strings.Contains(err.Error(), string(openfeature.FlagNotFoundCode)) {
		t.Errorf("error %q does not identify the flag as not found", err.Error())
	}

	details, err := client.BooleanValueDetails(ctx, missingFlag, true, openfeature.EvaluationContext{})
	if err == nil {
		t.Fatal("BooleanValueDetails returned no error; a missing flag must surface one")
	}

	if !details.Value {
		t.Errorf("details.Value = %v, want true (the caller's default)", details.Value)
	}

	if details.FlagKey != missingFlag {
		t.Errorf("details.FlagKey = %q, want %q", details.FlagKey, missingFlag)
	}

	if details.Reason != openfeature.ErrorReason {
		t.Errorf("details.Reason = %q, want %q", details.Reason, openfeature.ErrorReason)
	}

	if details.ErrorCode != openfeature.FlagNotFoundCode {
		t.Errorf("details.ErrorCode = %q, want %q", details.ErrorCode, openfeature.FlagNotFoundCode)
	}

	if got := openfeature.NamedProviderMetadata(domain).Name; got != launchpad.ProviderName {
		t.Errorf("provider name for domain %q = %q, want %q", domain, got, launchpad.ProviderName)
	}
}

func TestNewProvider_AcceptsAFlagSet(t *testing.T) {
	t.Parallel()

	set, err := launchpad.NewFlagSet(map[string]launchpad.Flag{
		"new-checkout": {
			State:          launchpad.Enabled,
			DefaultVariant: "on",
			Variants:       map[string]any{"on": true, "off": false},
		},
	})
	if err != nil {
		t.Fatalf("NewFlagSet() error = %v, want nil", err)
	}

	if got := launchpad.NewProvider(launchpad.WithFlags(set)); got == nil {
		t.Error("NewProvider(WithFlags(set)) = nil, want a provider")
	}
	if got := launchpad.NewProvider(); got == nil {
		t.Error("NewProvider() = nil, want a provider")
	}
}

func TestNewProvider_ZeroFlagSetResolvesNothing(t *testing.T) {
	t.Parallel()

	// A caller who ignores the error from NewFlagSet passes the zero value.
	var set launchpad.FlagSet
	provider := launchpad.NewProvider(launchpad.WithFlags(set))

	got := provider.BooleanEvaluation(context.Background(), "new-checkout", true, nil)

	if got.Value != true {
		t.Errorf("BooleanEvaluation() value = %v, want the caller's default true", got.Value)
	}
	if code := got.ResolutionError.Error(); !strings.Contains(code, string(openfeature.FlagNotFoundCode)) {
		t.Errorf("BooleanEvaluation() error = %q, want it to carry %v", code, openfeature.FlagNotFoundCode)
	}
}

// enabledSet is a flag set holding one enabled flag, for tests that care about
// a single flag's resolution rather than about validation.
func enabledSet(t *testing.T, key string, defaultVariant string, variants map[string]any) launchpad.FlagSet {
	t.Helper()

	set, err := launchpad.NewFlagSet(map[string]launchpad.Flag{
		key: {State: launchpad.Enabled, DefaultVariant: defaultVariant, Variants: variants},
	})
	if err != nil {
		t.Fatalf("NewFlagSet() error = %v, want nil", err)
	}

	return set
}

func TestEvaluation_ResolvesEnabledFlag(t *testing.T) {
	t.Parallel()

	set := enabledSet(t, "new-checkout", "on", map[string]any{"on": true, "off": false})
	provider := launchpad.NewProvider(launchpad.WithFlags(set))

	got := provider.BooleanEvaluation(context.Background(), "new-checkout", false, nil)

	if !got.Value {
		t.Errorf("BooleanEvaluation() value = %v, want true", got.Value)
	}
	if got.Reason != openfeature.StaticReason {
		t.Errorf("BooleanEvaluation() reason = %v, want %v", got.Reason, openfeature.StaticReason)
	}
	if got.Variant != "on" {
		t.Errorf("BooleanEvaluation() variant = %q, want %q", got.Variant, "on")
	}
	if got.ResolutionError != (openfeature.ResolutionError{}) {
		t.Errorf("BooleanEvaluation() error = %v, want no error", got.ResolutionError)
	}
}

func TestEvaluation_ResolvesEveryValueType(t *testing.T) {
	t.Parallel()

	object := map[string]any{"theme": "dark"}

	tests := map[string]struct {
		variant string
		value   any
		call    func(*launchpad.Provider, string) (any, openfeature.ProviderResolutionDetail)
	}{
		"boolean": {variant: "on", value: true, call: func(p *launchpad.Provider, k string) (any, openfeature.ProviderResolutionDetail) {
			got := p.BooleanEvaluation(context.Background(), k, false, nil)

			return got.Value, got.ProviderResolutionDetail
		}},
		"string": {variant: "formal", value: "Good day", call: func(p *launchpad.Provider, k string) (any, openfeature.ProviderResolutionDetail) {
			got := p.StringEvaluation(context.Background(), k, "fallback", nil)

			return got.Value, got.ProviderResolutionDetail
		}},
		"int": {variant: "standard", value: int64(50), call: func(p *launchpad.Provider, k string) (any, openfeature.ProviderResolutionDetail) {
			got := p.IntEvaluation(context.Background(), k, 10, nil)

			return got.Value, got.ProviderResolutionDetail
		}},
		"float": {variant: "precise", value: 2.5, call: func(p *launchpad.Provider, k string) (any, openfeature.ProviderResolutionDetail) {
			got := p.FloatEvaluation(context.Background(), k, 1.5, nil)

			return got.Value, got.ProviderResolutionDetail
		}},
		"object": {variant: "dark", value: object, call: func(p *launchpad.Provider, k string) (any, openfeature.ProviderResolutionDetail) {
			got := p.ObjectEvaluation(context.Background(), k, nil, nil)

			return got.Value, got.ProviderResolutionDetail
		}},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			set := enabledSet(t, "flag", tt.variant, map[string]any{tt.variant: tt.value})
			value, detail := tt.call(launchpad.NewProvider(launchpad.WithFlags(set)), "flag")

			if !reflect.DeepEqual(value, tt.value) {
				t.Errorf("value = %#v, want %#v", value, tt.value)
			}
			if detail.Reason != openfeature.StaticReason {
				t.Errorf("reason = %v, want %v", detail.Reason, openfeature.StaticReason)
			}
			if detail.Variant != tt.variant {
				t.Errorf("variant = %q, want %q", detail.Variant, tt.variant)
			}
			if detail.ResolutionError != (openfeature.ResolutionError{}) {
				t.Errorf("error = %v, want no error", detail.ResolutionError)
			}
		})
	}
}

func TestEvaluation_DisabledFlagReturnsCallerDefault(t *testing.T) {
	t.Parallel()

	set, err := launchpad.NewFlagSet(map[string]launchpad.Flag{
		"new-checkout": {
			State:          launchpad.Disabled,
			DefaultVariant: "on",
			Variants:       map[string]any{"on": true, "off": false},
		},
	})
	if err != nil {
		t.Fatalf("NewFlagSet() error = %v, want nil", err)
	}
	provider := launchpad.NewProvider(launchpad.WithFlags(set))

	got := provider.BooleanEvaluation(context.Background(), "new-checkout", false, nil)

	if got.Value {
		t.Errorf("BooleanEvaluation() value = %v, want the caller's default false", got.Value)
	}
	if got.Reason != openfeature.DisabledReason {
		t.Errorf("BooleanEvaluation() reason = %v, want %v", got.Reason, openfeature.DisabledReason)
	}
	if got.ResolutionError != (openfeature.ResolutionError{}) {
		t.Errorf("BooleanEvaluation() error = %v, want no error", got.ResolutionError)
	}
	if got.Variant != "" {
		t.Errorf("BooleanEvaluation() variant = %q, want the empty string", got.Variant)
	}
}

func TestEvaluation_DisabledIsDistinguishableFromNotFound(t *testing.T) {
	t.Parallel()

	set, err := launchpad.NewFlagSet(map[string]launchpad.Flag{
		"new-checkout": {
			State:          launchpad.Disabled,
			DefaultVariant: "on",
			Variants:       map[string]any{"on": true},
		},
	})
	if err != nil {
		t.Fatalf("NewFlagSet() error = %v, want nil", err)
	}
	provider := launchpad.NewProvider(launchpad.WithFlags(set))

	disabled := provider.BooleanEvaluation(context.Background(), "new-checkout", true, nil)
	absent := provider.BooleanEvaluation(context.Background(), "absent", true, nil)

	if !disabled.Value || !absent.Value {
		t.Errorf("values = %v and %v, want both to be the caller's default true", disabled.Value, absent.Value)
	}
	if disabled.Reason != openfeature.DisabledReason {
		t.Errorf("disabled reason = %v, want %v", disabled.Reason, openfeature.DisabledReason)
	}
	if disabled.ResolutionError != (openfeature.ResolutionError{}) {
		t.Errorf("disabled error = %v, want no error", disabled.ResolutionError)
	}
	if absent.Reason != openfeature.ErrorReason {
		t.Errorf("absent reason = %v, want %v", absent.Reason, openfeature.ErrorReason)
	}
	if code := absent.ResolutionError.Error(); !strings.Contains(code, string(openfeature.FlagNotFoundCode)) {
		t.Errorf("absent error = %q, want it to carry %v", code, openfeature.FlagNotFoundCode)
	}
}

func TestEvaluation_WrongTypeReportsTypeMismatch(t *testing.T) {
	t.Parallel()

	// Each case resolves a flag whose variant holds a value of a different
	// type than the evaluation asks for.
	tests := map[string]struct {
		held any
		call func(*launchpad.Provider, string) (any, openfeature.ProviderResolutionDetail)
		want any
	}{
		"string request against a boolean flag": {held: true, want: "fallback",
			call: func(p *launchpad.Provider, k string) (any, openfeature.ProviderResolutionDetail) {
				got := p.StringEvaluation(context.Background(), k, "fallback", nil)

				return got.Value, got.ProviderResolutionDetail
			}},
		"boolean request against a string flag": {held: "Good day", want: false,
			call: func(p *launchpad.Provider, k string) (any, openfeature.ProviderResolutionDetail) {
				got := p.BooleanEvaluation(context.Background(), k, false, nil)

				return got.Value, got.ProviderResolutionDetail
			}},
		"int request against a string flag": {held: "Good day", want: int64(10),
			call: func(p *launchpad.Provider, k string) (any, openfeature.ProviderResolutionDetail) {
				got := p.IntEvaluation(context.Background(), k, 10, nil)

				return got.Value, got.ProviderResolutionDetail
			}},
		"float request against a boolean flag": {held: true, want: 1.5,
			call: func(p *launchpad.Provider, k string) (any, openfeature.ProviderResolutionDetail) {
				got := p.FloatEvaluation(context.Background(), k, 1.5, nil)

				return got.Value, got.ProviderResolutionDetail
			}},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			set := enabledSet(t, "new-checkout", "only", map[string]any{"only": tt.held})
			value, detail := tt.call(launchpad.NewProvider(launchpad.WithFlags(set)), "new-checkout")

			if !reflect.DeepEqual(value, tt.want) {
				t.Errorf("value = %#v, want the caller's default %#v", value, tt.want)
			}
			if code := detail.ResolutionError.Error(); !strings.Contains(code, string(openfeature.TypeMismatchCode)) {
				t.Errorf("error = %q, want it to carry %v", code, openfeature.TypeMismatchCode)
			}
			if detail.Reason != openfeature.ErrorReason {
				t.Errorf("reason = %v, want %v", detail.Reason, openfeature.ErrorReason)
			}
			if !strings.Contains(detail.ResolutionError.Error(), "new-checkout") {
				t.Errorf("error = %q, want it to name the flag key", detail.ResolutionError.Error())
			}
			if detail.Variant != "" {
				t.Errorf("variant = %q, want the empty string", detail.Variant)
			}
		})
	}
}

func TestEvaluation_ObjectAcceptsAnyResolvedValue(t *testing.T) {
	t.Parallel()

	set := enabledSet(t, "new-checkout", "on", map[string]any{"on": true})
	provider := launchpad.NewProvider(launchpad.WithFlags(set))

	got := provider.ObjectEvaluation(context.Background(), "new-checkout", nil, nil)

	if got.Value != true {
		t.Errorf("ObjectEvaluation() value = %#v, want true", got.Value)
	}
	if got.Reason != openfeature.StaticReason {
		t.Errorf("ObjectEvaluation() reason = %v, want %v", got.Reason, openfeature.StaticReason)
	}
	if got.ResolutionError != (openfeature.ResolutionError{}) {
		t.Errorf("ObjectEvaluation() error = %v, want no error", got.ResolutionError)
	}
}

func TestEvaluation_NumericRule(t *testing.T) {
	t.Parallel()

	t.Run("a whole number resolves as an integer", func(t *testing.T) {
		t.Parallel()

		set := enabledSet(t, "max-items", "standard", map[string]any{"standard": 50})
		got := launchpad.NewProvider(launchpad.WithFlags(set)).
			IntEvaluation(context.Background(), "max-items", 10, nil)

		if got.Value != 50 {
			t.Errorf("IntEvaluation() value = %d, want 50", got.Value)
		}
		if got.Reason != openfeature.StaticReason {
			t.Errorf("IntEvaluation() reason = %v, want %v", got.Reason, openfeature.StaticReason)
		}
	})

	t.Run("a whole number resolves as a float", func(t *testing.T) {
		t.Parallel()

		set := enabledSet(t, "max-items", "standard", map[string]any{"standard": 50})
		got := launchpad.NewProvider(launchpad.WithFlags(set)).
			FloatEvaluation(context.Background(), "max-items", 1.5, nil)

		if got.Value != 50.0 {
			t.Errorf("FloatEvaluation() value = %v, want 50", got.Value)
		}
		if got.Reason != openfeature.StaticReason {
			t.Errorf("FloatEvaluation() reason = %v, want %v", got.Reason, openfeature.StaticReason)
		}
	})

	t.Run("a fractional number is not an integer", func(t *testing.T) {
		t.Parallel()

		set := enabledSet(t, "sample-rate", "precise", map[string]any{"precise": 2.5})
		got := launchpad.NewProvider(launchpad.WithFlags(set)).
			IntEvaluation(context.Background(), "sample-rate", 1, nil)

		if got.Value != 1 {
			t.Errorf("IntEvaluation() value = %d, want the caller's default 1", got.Value)
		}
		if code := got.ResolutionError.Error(); !strings.Contains(code, string(openfeature.TypeMismatchCode)) {
			t.Errorf("IntEvaluation() error = %q, want it to carry %v", code, openfeature.TypeMismatchCode)
		}
	})

	t.Run("a fractional number resolves as a float", func(t *testing.T) {
		t.Parallel()

		set := enabledSet(t, "sample-rate", "precise", map[string]any{"precise": 2.5})
		got := launchpad.NewProvider(launchpad.WithFlags(set)).
			FloatEvaluation(context.Background(), "sample-rate", 1.0, nil)

		if got.Value != 2.5 {
			t.Errorf("FloatEvaluation() value = %v, want 2.5", got.Value)
		}
		if got.Reason != openfeature.StaticReason {
			t.Errorf("FloatEvaluation() reason = %v, want %v", got.Reason, openfeature.StaticReason)
		}
	})
}

func TestEvaluation_MatchesFlagKeysExactly(t *testing.T) {
	t.Parallel()

	set := enabledSet(t, "new-checkout", "on", map[string]any{"on": true})
	provider := launchpad.NewProvider(launchpad.WithFlags(set))

	got := provider.BooleanEvaluation(context.Background(), "New-Checkout", false, nil)

	if got.Value {
		t.Errorf("BooleanEvaluation() value = %v, want the caller's default false", got.Value)
	}
	if code := got.ResolutionError.Error(); !strings.Contains(code, string(openfeature.FlagNotFoundCode)) {
		t.Errorf("BooleanEvaluation() error = %q, want it to carry %v", code, openfeature.FlagNotFoundCode)
	}
}

func TestEvaluation_IsSafeUnderConcurrentUse(t *testing.T) {
	t.Parallel()

	set := enabledSet(t, "new-checkout", "on", map[string]any{"on": true, "off": false})
	provider := launchpad.NewProvider(launchpad.WithFlags(set))

	var wg sync.WaitGroup
	for range 64 {
		wg.Add(1)
		go func() {
			defer wg.Done()

			got := provider.BooleanEvaluation(context.Background(), "new-checkout", false, nil)
			if !got.Value || got.Variant != "on" {
				t.Errorf("BooleanEvaluation() = (%v, %q), want (true, \"on\")", got.Value, got.Variant)
			}
		}()
	}
	wg.Wait()
}

// unresolvableSet holds a disabled flag and a boolean flag, which between them
// reach every outcome where the provider resolves no usable value for a key it
// does hold.
func unresolvableSet(t *testing.T) launchpad.FlagSet {
	t.Helper()

	set, err := launchpad.NewFlagSet(map[string]launchpad.Flag{
		"disabled-flag": {
			State:          launchpad.Disabled,
			DefaultVariant: "on",
			Variants:       map[string]any{"on": true},
		},
		"boolean-flag": {
			State:          launchpad.Enabled,
			DefaultVariant: "on",
			Variants:       map[string]any{"on": true},
		},
	})
	if err != nil {
		t.Fatalf("NewFlagSet() error = %v, want nil", err)
	}

	return set
}

func TestEvaluation_DefaultPassesThroughForEveryUnresolvableOutcome(t *testing.T) {
	t.Parallel()

	provider := launchpad.NewProvider(launchpad.WithFlags(unresolvableSet(t)))

	disabled := provider.BooleanEvaluation(context.Background(), "disabled-flag", false, nil)
	if disabled.Value {
		t.Errorf("disabled value = %v, want the caller's default false", disabled.Value)
	}

	mismatch := provider.StringEvaluation(context.Background(), "boolean-flag", "fallback", nil)
	if mismatch.Value != "fallback" {
		t.Errorf("mismatch value = %q, want the caller's default %q", mismatch.Value, "fallback")
	}
}

func TestEvaluation_HeldFlagNeverReportsFlagNotFound(t *testing.T) {
	t.Parallel()

	provider := launchpad.NewProvider(launchpad.WithFlags(unresolvableSet(t)))

	disabled := provider.BooleanEvaluation(context.Background(), "disabled-flag", false, nil)
	if disabled.ResolutionError != (openfeature.ResolutionError{}) {
		t.Errorf("disabled error = %v, want no error at all", disabled.ResolutionError)
	}

	mismatch := provider.StringEvaluation(context.Background(), "boolean-flag", "fallback", nil)
	if code := mismatch.ResolutionError.Error(); strings.Contains(code, string(openfeature.FlagNotFoundCode)) {
		t.Errorf("mismatch error = %q, want it not to carry %v", code, openfeature.FlagNotFoundCode)
	}
}

func TestEvaluation_VariantIsEmptyWheneverNoVariantWasSelected(t *testing.T) {
	t.Parallel()

	provider := launchpad.NewProvider(launchpad.WithFlags(unresolvableSet(t)))

	if got := provider.BooleanEvaluation(context.Background(), "disabled-flag", false, nil); got.Variant != "" {
		t.Errorf("disabled variant = %q, want the empty string", got.Variant)
	}
	if got := provider.StringEvaluation(context.Background(), "boolean-flag", "fallback", nil); got.Variant != "" {
		t.Errorf("mismatch variant = %q, want the empty string", got.Variant)
	}
}

// Not parallel, and with its own domain: provider registration is SDK-global,
// so sharing a domain with another test would make both order-dependent.
func TestClientEvaluation_ResolvedFlag(t *testing.T) {
	const domain = "launchpad-test-resolved"

	t.Cleanup(openfeature.Shutdown)

	set, err := launchpad.NewFlagSet(map[string]launchpad.Flag{
		"new-checkout": {
			State:          launchpad.Enabled,
			DefaultVariant: "on",
			Variants:       map[string]any{"on": true, "off": false},
		},
	})
	if err != nil {
		t.Fatalf("NewFlagSet() error = %v, want nil", err)
	}

	provider := launchpad.NewProvider(launchpad.WithFlags(set))
	if err := openfeature.SetNamedProviderAndWait(domain, provider); err != nil {
		t.Fatalf("SetNamedProviderAndWait: %v", err)
	}

	client := openfeature.NewClient(domain)
	ctx := context.Background()

	value, err := client.BooleanValue(ctx, "new-checkout", false, openfeature.EvaluationContext{})
	if err != nil {
		t.Fatalf("BooleanValue returned %v, want no error for a flag that resolves", err)
	}
	if !value {
		t.Errorf("BooleanValue = %v, want true (the flag's own value, not the default)", value)
	}

	details, err := client.BooleanValueDetails(ctx, "new-checkout", false, openfeature.EvaluationContext{})
	if err != nil {
		t.Fatalf("BooleanValueDetails returned %v, want no error", err)
	}

	if !details.Value {
		t.Errorf("details.Value = %v, want true", details.Value)
	}
	if details.Variant != "on" {
		t.Errorf("details.Variant = %q, want %q", details.Variant, "on")
	}
	if details.Reason != openfeature.StaticReason {
		t.Errorf("details.Reason = %q, want %q", details.Reason, openfeature.StaticReason)
	}
	if details.ErrorCode != "" {
		t.Errorf("details.ErrorCode = %q, want no error code", details.ErrorCode)
	}
	if got := openfeature.NamedProviderMetadata(domain).Name; got != launchpad.ProviderName {
		t.Errorf("provider name for domain %q = %q, want %q", domain, got, launchpad.ProviderName)
	}
}
