package launchpad_test

import (
	"context"
	"reflect"
	"strings"
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
