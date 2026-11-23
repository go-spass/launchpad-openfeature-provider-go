.DEFAULT_GOAL := check

GO ?= go
GOFMT ?= gofmt
GOLANGCI_LINT ?= golangci-lint

.PHONY: build vet test lint fmt fmt-check tidy check

build:
	$(GO) build ./...

vet:
	$(GO) vet ./...

test:
	$(GO) test -v -race -cover ./...

# golangci-lint is an external prerequisite, not a module dependency: fail with
# an install hint rather than skipping the lint step.
lint:
	@command -v $(GOLANGCI_LINT) >/dev/null 2>&1 || { \
		echo "$(GOLANGCI_LINT) not found on PATH. Install it with: brew install golangci-lint" >&2; \
		exit 1; \
	}
	$(GOLANGCI_LINT) run ./...

# fmt is the only target that edits files; the gate uses fmt-check instead.
fmt:
	$(GOFMT) -l -w .

# gofmt -l exits 0 even when it lists unformatted files, so assert on output.
fmt-check:
	@files=$$($(GOFMT) -l .); \
	if [ -n "$$files" ]; then \
		echo "not gofmt-ed:" >&2; \
		echo "$$files" >&2; \
		echo "run 'make fmt'" >&2; \
		exit 1; \
	fi

tidy:
	$(GO) mod tidy

# Recipe rather than prerequisites, so the order holds under `make -j`:
# cheapest and most localized signal first.
check:
	$(MAKE) fmt-check
	$(MAKE) vet
	$(MAKE) lint
	$(MAKE) test
