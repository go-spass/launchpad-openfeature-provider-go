# Tasks

## 1. Branch and dependency setup

- [x] 1.1 Create and switch to the branch `feat/provider-skeleton` off an up-to-date `main`;
      verify `git rev-parse --abbrev-ref HEAD` reports `feat/provider-skeleton`
- [x] 1.2 Add `github.com/open-feature/go-sdk v1.20.0` with `go get`; verify `go.mod` contains the
      pinned `require` line, `go.sum` now exists, and `go build ./...` succeeds
- [x] 1.3 Run `make tidy` and verify the module files are tidy-stable: a second `make tidy`
      changes neither `go.mod` nor `go.sum`. Verified after task 2.2, because until
      `launchpad/provider.go` imports the SDK, `go get` records the requirement as `// indirect`
      and `go mod tidy` correctly drops it, so the check is only meaningful once the import
      exists. The first `tidy` after that import is expected to change `go.mod` — it promotes the
      SDK to a direct requirement and adds `go.uber.org/mock v0.6.0 // indirect`, pulled in by the
      SDK's own test dependencies. The original wording of this task also asked for a clean
      `git status` for both files, which cannot hold: this change is what creates `go.sum` and
      first adds a `require` line, and nothing is committed during apply

## 2. Provider type and interface conformance

- [x] 2.1 Write `launchpad/provider_test.go` in package `launchpad_test` with
      `TestProviderSatisfiesFeatureProvider`, assigning `launchpad.NewProvider()` to a variable of
      type `openfeature.FeatureProvider`; verify `go test ./launchpad` fails to build because
      `NewProvider` and `Provider` do not exist
- [x] 2.2 Add `Provider` and `NewProvider() *Provider` to `launchpad/provider.go` plus the five
      evaluation methods, `Metadata()` and `Hooks()` as stubs returning zero values, so the test
      compiles; verify `go test ./launchpad -run TestProviderSatisfiesFeatureProvider` passes and
      the build error is gone
- [x] 2.3 Add `var _ openfeature.FeatureProvider = (*Provider)(nil)` to `launchpad/provider.go`;
      verify `go build ./...` succeeds and that temporarily renaming one method makes `go build`
      fail rather than only `go test`

## 3. Provider identity and hooks

- [x] 3.1 Write `TestMetadata` asserting the reported name is non-empty and identical across two
      separately constructed providers read twice each; verify it fails because the stub returns an
      empty name
- [x] 3.2 Implement `Metadata()` to return a constant non-empty provider name; verify
      `go test ./launchpad -run TestMetadata` passes
- [x] 3.3 Write `TestHooks` asserting the returned slice has length 0 and is not nil; verify it
      fails on the non-nil assertion because the stub returns `nil`
- [x] 3.4 Implement `Hooks()` to return `[]openfeature.Hook{}`; verify
      `go test ./launchpad -run TestHooks` passes

## 4. Unresolved evaluation outcome

- [x] 4.1 Write `TestEvaluation_ReturnsCallerDefault` with one typed subtest per value type —
      boolean asserted with both `true` and `false` defaults, plus string, int64, float64 and
      object; verify it fails because the stubs return zero values rather than the caller's default
- [x] 4.2 Implement the five evaluation methods to return the caller's `defaultValue` in the
      detail's `Value` field; verify `go test ./launchpad -run TestEvaluation_ReturnsCallerDefault`
      passes
- [x] 4.3 Write `TestEvaluation_ReportsFlagNotFound` driven by a
      `map[string]func(*launchpad.Provider) openfeature.ProviderResolutionDetail` table with one
      entry per value type, asserting error code `openfeature.FlagNotFoundCode`, reason
      `openfeature.ErrorReason`, and an error message containing the flag key `missing-flag`;
      verify it fails because the stubs report an empty reason and no resolution error
- [x] 4.4 Add the unexported `flagNotFound(flag string) openfeature.ProviderResolutionDetail`
      helper and return it from all five evaluation methods; verify
      `go test ./launchpad -run TestEvaluation_ReportsFlagNotFound` passes
- [x] 4.5 Write `TestEvaluation_ReportsNoVariant` over the same closure table asserting `Variant`
      is the empty string; verify it passes against the helper from 4.4 without further code
      changes, confirming the helper leaves `Variant` at its zero value
- [x] 4.6 Write `TestEvaluation_AllTypesAnswered` covering the closure table plus a subtest that
      supplies a nil `openfeature.FlattenedContext`; verify
      `go test ./launchpad -run TestEvaluation_AllTypesAnswered` passes with no panic

## 5. End-to-end evaluation through an OpenFeature client

- [x] 5.1 Write `TestClientEvaluation_MissingFlag`, registering the provider with
      `openfeature.SetNamedProviderAndWait("launchpad-test", p)`, evaluating via
      `openfeature.NewClient("launchpad-test")`, and asserting `BooleanValue` returns the `true`
      default together with a non-nil error, and that `BooleanValueDetails` reports the provider
      name, reason `ERROR` and error code `FLAG_NOT_FOUND`; the subtest must not call
      `t.Parallel()` and must register `openfeature.Shutdown` with `t.Cleanup`; verify it passes
      against the implementation from groups 3 and 4 without further code changes
- [x] 5.2 Run `go test -race ./launchpad` twice in a row and verify both runs pass, confirming the
      SDK-global registration does not leak between runs

## 6. Documentation

- [x] 6.1 Update `launchpad/doc.go` and add a `Provider` doc comment stating that the provider
      currently holds no flags and that every evaluation returns the caller's default with
      `FLAG_NOT_FOUND`; verify `go doc ./launchpad` renders both comments
- [x] 6.2 Update the "Project status" section of `CLAUDE.md`, which currently states the
      repository has no provider code, no tests and no `go.sum`; verify the section describes the
      `launchpad` package's provider, its tests and the pinned SDK dependency

## 7. Verification gate

- [x] 7.1 Run the gate and verify all of it is clean: `gofmt -l .` prints nothing,
      `go vet ./...` exits 0, `go test -race -cover ./...` passes, and `make check` succeeds
      end to end
