// Package launchpad implements an [OpenFeature] provider backed by an
// in-memory feature flag store.
//
// The provider holds no flags yet. It satisfies the OpenFeature provider
// contract in full, but no flag key resolves: every evaluation returns the
// default value the caller supplied, with the FLAG_NOT_FOUND error code and the
// ERROR reason. Flag storage and evaluation arrive in a later change.
//
// [OpenFeature]: https://openfeature.dev
package launchpad
