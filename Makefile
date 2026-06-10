GO ?= go
GOFMT ?= gofmt
PACKAGES ?= ./...
BINARY ?= bin/prismgo
CMD_DIR ?= ./cmd/prismgo
APP_DIR ?= tmp/app
LINT_ARGS ?=
GO_BUILD_CACHE ?= $(CURDIR)/tmp/gocache
GOLANGCI_LINT_CACHE_DIR ?= $(CURDIR)/tmp/golangci-lint

HAS_GOMOD := $(wildcard go.mod)
HAS_CMD := $(wildcard cmd/prismgo)

.PHONY: help
help:
	@echo "PrismGo Installer targets:"
	@echo "  make fmt        Format Go files when go.mod exists"
	@echo "  make fmt-check  Check Go formatting when go.mod exists"
	@echo "  make vet        Run go vet when go.mod exists"
	@echo "  make test       Run Go tests when go.mod exists"
	@echo "  make lint       Run golangci-lint when go.mod exists"
	@echo "  make build      Build prismgo when cmd/prismgo exists"
	@echo "  make install    Install prismgo when cmd/prismgo exists"
	@echo "  make smoke-new  Reserved for prismgo new $(APP_DIR)"
	@echo "  make ci         Run checks valid for the current repository state"

.PHONY: require-go-module
require-go-module:
	@if [ -z "$(HAS_GOMOD)" ]; then \
		echo "Skipping Go check: go.mod is not present yet."; \
		exit 0; \
	fi

.PHONY: require-prismgo-command
require-prismgo-command:
	@if [ -z "$(HAS_CMD)" ]; then \
		echo "Skipping PrismGo CLI command: $(CMD_DIR) is not present yet."; \
		exit 0; \
	fi

.PHONY: test
test:
	@if [ -z "$(HAS_GOMOD)" ]; then \
		echo "Skipping tests: go.mod is not present yet."; \
	else \
		$(GO) test -v $(PACKAGES); \
	fi

.PHONY: vet
vet:
	@if [ -z "$(HAS_GOMOD)" ]; then \
		echo "Skipping vet: go.mod is not present yet."; \
	else \
		$(GO) vet $(PACKAGES); \
	fi

.PHONY: fmt
fmt:
	@if [ -z "$(HAS_GOMOD)" ]; then \
		echo "Skipping format: go.mod is not present yet."; \
	else \
		files="$$(find . -name '*.go' -not -path './tmp/*')"; \
		if [ -n "$$files" ]; then $(GOFMT) -w $$files; fi; \
	fi

.PHONY: fmt-check
fmt-check:
	@if [ -z "$(HAS_GOMOD)" ]; then \
		echo "Skipping format check: go.mod is not present yet."; \
	else \
		tmp="$$(mktemp -d)"; \
		trap 'rm -rf "$$tmp"' EXIT; \
		status=0; \
		for file in $$(find . -name '*.go' -not -path './tmp/*'); do \
			current="$$tmp/current"; \
			formatted="$$tmp/formatted"; \
			sed 's/\r$$//' "$$file" > "$$current"; \
			$(GOFMT) "$$file" > "$$formatted"; \
			if ! cmp -s "$$current" "$$formatted"; then \
				echo "Please run 'make fmt' and commit the result for $$file:"; \
				diff -u "$$current" "$$formatted" || true; \
				status=1; \
			fi; \
		done; \
		exit "$$status"; \
	fi

.PHONY: lint
lint:
	@if [ -z "$(HAS_GOMOD)" ]; then \
		echo "Skipping lint: go.mod is not present yet."; \
	else \
		mkdir -p "$(GO_BUILD_CACHE)" "$(GOLANGCI_LINT_CACHE_DIR)"; \
		env GOCACHE="$(GO_BUILD_CACHE)" GOLANGCI_LINT_CACHE="$(GOLANGCI_LINT_CACHE_DIR)" golangci-lint run $(LINT_ARGS) $(PACKAGES); \
	fi

.PHONY: build
build:
	@if [ -z "$(HAS_CMD)" ]; then \
		echo "Skipping build: $(CMD_DIR) is not present yet."; \
	else \
		mkdir -p bin; \
		$(GO) build -o $(BINARY) $(CMD_DIR); \
	fi

.PHONY: install
install:
	@if [ -z "$(HAS_CMD)" ]; then \
		echo "Skipping install: $(CMD_DIR) is not present yet."; \
	else \
		$(GO) install $(CMD_DIR); \
	fi

.PHONY: smoke-new
smoke-new:
	@echo "Skipping smoke-new: prismgo new $(APP_DIR) is intentionally deferred until the installer implementation is ready."

.PHONY: ci
ci: fmt-check vet test lint
