// Package launchpad implements an [OpenFeature] provider backed by an
// in-memory feature flag store.
//
// Flags are supplied as a validated set:
//
//	set, err := launchpad.NewFlagSet(map[string]launchpad.Flag{
//		"new-checkout": {
//			State:          launchpad.Enabled,
//			DefaultVariant: "on",
//			Variants:       map[string]any{"on": true, "off": false},
//		},
//	})
//	if err != nil {
//		return err
//	}
//	err = openfeature.SetProviderAndWait(launchpad.NewProvider(launchpad.WithFlags(set)))
//
// A flag does not declare its value type; its variant values carry it. An
// enabled flag resolves to its default variant's value with the STATIC reason,
// a disabled flag returns the caller's default with the DISABLED reason, a key
// the provider does not hold reports FLAG_NOT_FOUND, and asking for a type the
// resolved variant does not hold reports TYPE_MISMATCH. A provider built
// without flags holds an empty set, so every key reports FLAG_NOT_FOUND.
//
// Loading a flag set from a file, targeting and percentage rollout arrive in
// later changes.
//
// [OpenFeature]: https://openfeature.dev
package launchpad
