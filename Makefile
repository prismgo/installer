GO ?= go
GOFMT ?= gofmt
PACKAGES ?= ./...
BINARY ?= bin/prismgo
CMD_DIR ?= ./cmd/prismgo
SMOKE_ROOT ?= tmp/_smoke-new
APP_NAME ?= myapp
APP_MODULE ?= github.com/prismgo/$(APP_NAME)
APP_DIR ?= $(SMOKE_ROOT)/$(APP_NAME)
PRISMGO_REPOSITORY ?= https://github.com/prismgo/prismgo
SMOKE_CREATE_TIMEOUT ?= 300
SMOKE_RUN_TIMEOUT ?= 30
LINT_ARGS ?=
GO_BUILD_CACHE ?= $(CURDIR)/tmp/gocache
GO_MOD_CACHE ?= $(CURDIR)/tmp/_gomodcache
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
	@echo "  make covdata    Generate Go coverage data when go.mod exists"
	@echo "  make lint       Run golangci-lint when go.mod exists"
	@echo "  make build      Build prismgo when cmd/prismgo exists"
	@echo "  make install    Install prismgo when cmd/prismgo exists"
	@echo "  make smoke-new  Run prismgo new $(APP_NAME) against the live PrismGo skeleton"
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

.PHONY: covdata
covdata:
	@if [ -z "$(HAS_GOMOD)" ]; then \
		echo "Skipping coverage data: go.mod is not present yet."; \
	else \
		bash ./.github/scripts/coverage.sh $(PACKAGES); \
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
	@if [ -z "$(HAS_CMD)" ]; then \
		echo "Skipping smoke-new: $(CMD_DIR) is not present yet."; \
	else \
		set -eu; \
		ref_dir="$(SMOKE_ROOT)/prismgo-reference"; \
		bin_path="$(SMOKE_ROOT)/prismgo"; \
		new_log="$(SMOKE_ROOT)/prismgo-new.log"; \
		run_log="$(SMOKE_ROOT)/go-run.log"; \
		run_pid="$(SMOKE_ROOT)/go-run.pid"; \
		expected_top="$(SMOKE_ROOT)/expected-top-level.txt"; \
		actual_top="$(SMOKE_ROOT)/actual-top-level.txt"; \
		rm -rf "$(SMOKE_ROOT)"; \
		mkdir -p "$(SMOKE_ROOT)" "$(GO_BUILD_CACHE)" "$(GO_MOD_CACHE)"; \
		echo "Building PrismGo installer smoke binary"; \
		env GOCACHE="$(GO_BUILD_CACHE)" GOMODCACHE="$(GO_MOD_CACHE)" $(GO) build -o "$$bin_path" $(CMD_DIR); \
		echo "Cloning live PrismGo skeleton from $(PRISMGO_REPOSITORY)"; \
		git clone --depth=1 "$(PRISMGO_REPOSITORY)" "$$ref_dir"; \
		old_module="$$(sed -n 's/^module[[:space:]][[:space:]]*//p' "$$ref_dir/go.mod" | head -n 1)"; \
		echo "Generating $(APP_NAME) with module $(APP_MODULE)"; \
		env GOCACHE="$(GO_BUILD_CACHE)" GOMODCACHE="$(GO_MOD_CACHE)" "$$bin_path" new "$(APP_DIR)" --module "$(APP_MODULE)" > "$$new_log" 2>&1 & \
		new_pid="$$!"; \
		i=0; \
		while kill -0 "$$new_pid" 2>/dev/null && [ "$$i" -lt "$(SMOKE_CREATE_TIMEOUT)" ]; do \
			i="$$((i + 1))"; \
			sleep 1; \
		done; \
		if kill -0 "$$new_pid" 2>/dev/null; then \
			kill "$$new_pid"; \
			wait "$$new_pid" 2>/dev/null || true; \
			echo "prismgo new did not finish within $(SMOKE_CREATE_TIMEOUT) seconds" >&2; \
			cat "$$new_log" >&2; \
			exit 1; \
		fi; \
		if ! wait "$$new_pid"; then \
			cat "$$new_log" >&2; \
			exit 1; \
		fi; \
		( cd "$$ref_dir" && find . -mindepth 1 -maxdepth 1 ! -name .git -exec basename {} \; | sort ) > "$$expected_top"; \
		( cd "$(APP_DIR)" && find . -mindepth 1 -maxdepth 1 ! -name .git ! -name .env -exec basename {} \; | sort ) > "$$actual_top"; \
		echo "Checking generated top-level skeleton structure"; \
		diff -u "$$expected_top" "$$actual_top"; \
		echo "Checking module path replacement"; \
		grep -qx 'module $(APP_MODULE)' "$(APP_DIR)/go.mod"; \
		if [ -n "$$old_module" ] && grep -R "\"$$old_module/" "$(APP_DIR)" --include='*.go' --include='go.mod'; then \
			echo "Generated app still contains old import prefix $$old_module/" >&2; \
			exit 1; \
		fi; \
		echo "Checking github.com/prismgo/framework module resolution"; \
		framework="$$(cd "$(APP_DIR)" && env GOCACHE="$(GO_BUILD_CACHE)" GOMODCACHE="$(GO_MOD_CACHE)" $(GO) list -m -f '{{if .Replace}}replace {{.Replace.Path}}{{else}}{{.Version}}{{end}}' github.com/prismgo/framework)"; \
		case "$$framework" in \
			""|replace*) echo "github.com/prismgo/framework resolved incorrectly: $$framework" >&2; exit 1 ;; \
			*) echo "github.com/prismgo/framework $$framework" ;; \
		esac; \
		echo "Checking generated app startup output"; \
		( cd "$(APP_DIR)" && env GOCACHE="$(GO_BUILD_CACHE)" GOMODCACHE="$(GO_MOD_CACHE)" $(GO) run . ) > "$$run_log" 2>&1 & \
		echo "$$!" > "$$run_pid"; \
		i=0; \
		while [ "$$i" -lt "$(SMOKE_RUN_TIMEOUT)" ]; do \
			if grep -q 'PrismGo' "$$run_log"; then \
				break; \
			fi; \
			if ! kill -0 "$$(cat "$$run_pid")" 2>/dev/null; then \
				break; \
			fi; \
			i="$$((i + 1))"; \
			sleep 1; \
		done; \
		if ! grep -q 'PrismGo' "$$run_log"; then \
			echo "go run . did not print PrismGo within $(SMOKE_RUN_TIMEOUT) seconds" >&2; \
			cat "$$run_log" >&2; \
			exit 1; \
		fi; \
		if kill -0 "$$(cat "$$run_pid")" 2>/dev/null; then \
			kill "$$(cat "$$run_pid")"; \
			wait "$$(cat "$$run_pid")" 2>/dev/null || true; \
		fi; \
		echo "smoke-new passed"; \
	fi

.PHONY: ci
ci: fmt-check vet test lint
