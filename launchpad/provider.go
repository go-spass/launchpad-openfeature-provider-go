package launchpad

import (
	"context"
	"fmt"

	"github.com/open-feature/go-sdk/openfeature"
)

// Provider must satisfy the SDK's provider contract. Asserting it here rather
// than in a test means an SDK upgrade that changes the interface fails the
// build, so a consumer running `go build` sees it without running our tests.
var _ openfeature.FeatureProvider = (*Provider)(nil)

// ProviderName is the name LaunchPad reports in its OpenFeature metadata. It
// identifies LaunchPad as the source of an evaluation and is stable across
// instances, so consumers can match it against evaluation details.
const ProviderName = "LaunchPad"

// Provider is the LaunchPad OpenFeature provider. Register it with the
// OpenFeature SDK rather than calling it directly:
//
//	err := openfeature.SetProviderAndWait(launchpad.NewProvider())
//
// Supply flags with [WithFlags]; a provider built without them holds an empty
// flag set, so every key resolves as not found. A Provider does not change
// after construction and is safe for concurrent use.
type Provider struct {
	flags FlagSet
}

// Option configures a Provider at construction. Options exist so that
// supplying flags, and whatever is configurable later, stays additive.
type Option func(*Provider)

// WithFlags serves the given flag set. The set is already valid by
// construction, so there is nothing here that can fail.
func WithFlags(set FlagSet) Option {
	return func(p *Provider) {
		p.flags = set
	}
}

// NewProvider returns a ready-to-use LaunchPad provider. With no options it
// holds no flags and resolves nothing, which is what it did before flag
// storage existed.
func NewProvider(opts ...Option) *Provider {
	p := &Provider{}
	for _, opt := range opts {
		opt(p)
	}

	return p
}

// flagNotFound builds the resolution detail returned for a flag the provider
// cannot resolve. Every evaluation method shares it so the reason, error code
// and variant cannot drift apart between value types. Variant is deliberately
// left at its zero value: no variant was selected.
func flagNotFound(flag string) openfeature.ProviderResolutionDetail {
	return openfeature.ProviderResolutionDetail{
		ResolutionError: openfeature.NewFlagNotFoundResolutionError(
			fmt.Sprintf("flag %q not found", flag),
		),
		Reason: openfeature.ErrorReason,
	}
}

// typeMismatch builds the resolution detail returned when a flag resolved but
// the value its variant holds is not of the requested type. Variant is left at
// its zero value: a value the caller cannot use is not a variant it selected.
func typeMismatch(flag string, value any, want string) openfeature.ProviderResolutionDetail {
	return openfeature.ProviderResolutionDetail{
		ResolutionError: openfeature.NewTypeMismatchResolutionError(
			fmt.Sprintf("flag %q holds %T, not %s", flag, value, want),
		),
		Reason: openfeature.ErrorReason,
	}
}

// resolve selects the value a flag evaluation should return, without regard to
// the type the caller asked for. Every typed evaluation method shares it, so
// lookup, state and variant selection cannot drift apart between value types,
// and targeting can later be added in one place.
//
// ok reports whether a variant was selected. When it is false the caller must
// return its own default value with the detail as given; when it is true the
// caller still has to check that value is of the type it was asked for.
func (p *Provider) resolve(flag string) (value any, ok bool, detail openfeature.ProviderResolutionDetail) {
	definition, found := p.flags.flags[flag]
	if !found {
		return nil, false, flagNotFound(flag)
	}

	if definition.State != Enabled {
		return nil, false, openfeature.ProviderResolutionDetail{Reason: openfeature.DisabledReason}
	}

	return definition.Variants[definition.DefaultVariant], true, openfeature.ProviderResolutionDetail{
		Reason:  openfeature.StaticReason,
		Variant: definition.DefaultVariant,
	}
}

// Metadata reports the provider's identity to the OpenFeature SDK.
func (p *Provider) Metadata() openfeature.Metadata {
	return openfeature.Metadata{Name: ProviderName}
}

// Hooks reports the evaluation hooks the provider contributes.
func (p *Provider) Hooks() []openfeature.Hook {
	return []openfeature.Hook{}
}

// BooleanEvaluation resolves a boolean flag.
func (p *Provider) BooleanEvaluation(
	_ context.Context, flag string, defaultValue bool, _ openfeature.FlattenedContext,
) openfeature.BoolResolutionDetail {
	value, ok, detail := p.resolve(flag)
	if !ok {
		return openfeature.BoolResolutionDetail{Value: defaultValue, ProviderResolutionDetail: detail}
	}

	typed, ok := value.(bool)
	if !ok {
		return openfeature.BoolResolutionDetail{
			Value:                    defaultValue,
			ProviderResolutionDetail: typeMismatch(flag, value, "a boolean"),
		}
	}

	return openfeature.BoolResolutionDetail{Value: typed, ProviderResolutionDetail: detail}
}

// StringEvaluation resolves a string flag.
func (p *Provider) StringEvaluation(
	_ context.Context, flag string, defaultValue string, _ openfeature.FlattenedContext,
) openfeature.StringResolutionDetail {
	value, ok, detail := p.resolve(flag)
	if !ok {
		return openfeature.StringResolutionDetail{Value: defaultValue, ProviderResolutionDetail: detail}
	}

	typed, ok := value.(string)
	if !ok {
		return openfeature.StringResolutionDetail{
			Value:                    defaultValue,
			ProviderResolutionDetail: typeMismatch(flag, value, "a string"),
		}
	}

	return openfeature.StringResolutionDetail{Value: typed, ProviderResolutionDetail: detail}
}

// IntEvaluation resolves an integer flag.
func (p *Provider) IntEvaluation(
	_ context.Context, flag string, defaultValue int64, _ openfeature.FlattenedContext,
) openfeature.IntResolutionDetail {
	value, ok, detail := p.resolve(flag)
	if !ok {
		return openfeature.IntResolutionDetail{Value: defaultValue, ProviderResolutionDetail: detail}
	}

	// A whole number is normalized to int64 when the flag set is built, so a
	// float64 here always carries a fractional part and cannot be an integer.
	typed, ok := value.(int64)
	if !ok {
		return openfeature.IntResolutionDetail{
			Value:                    defaultValue,
			ProviderResolutionDetail: typeMismatch(flag, value, "a whole number"),
		}
	}

	return openfeature.IntResolutionDetail{Value: typed, ProviderResolutionDetail: detail}
}

// FloatEvaluation resolves a floating-point flag.
func (p *Provider) FloatEvaluation(
	_ context.Context, flag string, defaultValue float64, _ openfeature.FlattenedContext,
) openfeature.FloatResolutionDetail {
	value, ok, detail := p.resolve(flag)
	if !ok {
		return openfeature.FloatResolutionDetail{Value: defaultValue, ProviderResolutionDetail: detail}
	}

	// A whole number is held as int64, and widening it loses nothing, so an
	// integral variant serves a float request as well as an integer one.
	switch typed := value.(type) {
	case float64:
		return openfeature.FloatResolutionDetail{Value: typed, ProviderResolutionDetail: detail}
	case int64:
		return openfeature.FloatResolutionDetail{Value: float64(typed), ProviderResolutionDetail: detail}
	default:
		return openfeature.FloatResolutionDetail{
			Value:                    defaultValue,
			ProviderResolutionDetail: typeMismatch(flag, value, "a number"),
		}
	}
}

// ObjectEvaluation resolves a structured flag.
func (p *Provider) ObjectEvaluation(
	_ context.Context, flag string, defaultValue any, _ openfeature.FlattenedContext,
) openfeature.InterfaceResolutionDetail {
	value, ok, detail := p.resolve(flag)
	if !ok {
		return openfeature.InterfaceResolutionDetail{Value: defaultValue, ProviderResolutionDetail: detail}
	}

	// An object evaluation constrains the value to no particular type, so every
	// resolved value satisfies it and there is no mismatch to report.
	return openfeature.InterfaceResolutionDetail{Value: value, ProviderResolutionDetail: detail}
}
