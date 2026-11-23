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
// It currently holds no flags, so no flag key resolves. Every evaluation
// returns the default value the caller supplied, with the FLAG_NOT_FOUND error
// code and the ERROR reason. A Provider carries no state and is safe for
// concurrent use.
type Provider struct{}

// NewProvider returns a ready-to-use LaunchPad provider.
func NewProvider() *Provider {
	return &Provider{}
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
	return openfeature.BoolResolutionDetail{
		Value:                    defaultValue,
		ProviderResolutionDetail: flagNotFound(flag),
	}
}

// StringEvaluation resolves a string flag.
func (p *Provider) StringEvaluation(
	_ context.Context, flag string, defaultValue string, _ openfeature.FlattenedContext,
) openfeature.StringResolutionDetail {
	return openfeature.StringResolutionDetail{
		Value:                    defaultValue,
		ProviderResolutionDetail: flagNotFound(flag),
	}
}

// IntEvaluation resolves an integer flag.
func (p *Provider) IntEvaluation(
	_ context.Context, flag string, defaultValue int64, _ openfeature.FlattenedContext,
) openfeature.IntResolutionDetail {
	return openfeature.IntResolutionDetail{
		Value:                    defaultValue,
		ProviderResolutionDetail: flagNotFound(flag),
	}
}

// FloatEvaluation resolves a floating-point flag.
func (p *Provider) FloatEvaluation(
	_ context.Context, flag string, defaultValue float64, _ openfeature.FlattenedContext,
) openfeature.FloatResolutionDetail {
	return openfeature.FloatResolutionDetail{
		Value:                    defaultValue,
		ProviderResolutionDetail: flagNotFound(flag),
	}
}

// ObjectEvaluation resolves a structured flag.
func (p *Provider) ObjectEvaluation(
	_ context.Context, flag string, defaultValue any, _ openfeature.FlattenedContext,
) openfeature.InterfaceResolutionDetail {
	return openfeature.InterfaceResolutionDetail{
		Value:                    defaultValue,
		ProviderResolutionDetail: flagNotFound(flag),
	}
}
