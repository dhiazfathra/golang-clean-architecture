.PHONY: infra-up infra-down build test cover migrate seed generate lint run vet install-hooks sql-validate

SERVICE_NAME ?= golang-clean-arch
ENV          ?= development

infra-up:
	docker compose -f deployments/docker-compose.yaml up -d

infra-down:
	docker compose -f deployments/docker-compose.yaml down

build:
	go build ./...

migrate:
	@echo "Applying migrations..."
	@for f in $$(ls migrations/*.up.sql | sort); do \
		echo "  $$f"; \
		psql "$(DATABASE_URL)" -f "$$f" 2>/dev/null || true; \
	done

seed:
	SEED_ONLY=true go run cmd/server/main.go

generate:
	go run cmd/generate/main.go -module=$(module) -fields=$(fields)
# Usage: make generate module=product fields="name:string,price:float64,sku:string,active:bool"

lint:
	golangci-lint run ./...

run:
	go run cmd/server/main.go

vet:
	go vet ./...

install-hooks:
	@bash scripts/install-hooks.sh

sql-validate:
	@bash -c 'for f in migrations/*.up.sql; do \
		base=$$(basename "$$f"); \
		if ! echo "$$base" | grep -qE "^[0-9]{14}_[a-z_]+\.up\.sql$$"; then \
			echo "Invalid: $$base"; exit 1; \
		fi; \
	done && echo "All migration filenames valid"'

# =============================================================================
# Go test & coverage targets aligned with codecov.yml
# =============================================================================
# Mirrors the ignore patterns in codecov.yml so that what is excluded in
# Codecov is also excluded from local coverage reports.
#
# Usage:
#   make test            # run all tests (no coverage)
#   make cover           # run tests + generate coverage report
#   make cover-html      # open coverage report in browser
#   make cover-check     # fail if total coverage is below COVERAGE_THRESHOLD
#   make cover-clean     # remove generated coverage artifacts
# =============================================================================

# --- Tunables (override on the command line or in CI env) -------------------
COVERAGE_THRESHOLD := 80        # Must match codecov.yml project target
COVERAGE_OUT       := coverage.out
COVERAGE_HTML      := coverage.html
GOTEST_TIMEOUT     := 5m
#GOTEST_TIMEOUT    := 10m       # Increase for slow integration suites
RACE               := -race     # Remove if your platform doesn't support -race

# --- Derived -----------------------------------------------------------------
MODULE := $(shell go list -m)

# =============================================================================
# EXCLUDE_PATTERNS — mirrors codecov.yml `ignore:` list exactly.
#
# `go test -coverpkg` and `go tool cover` work on package import paths, not
# file globs, so we build a grep pattern that removes matching import paths
# from the coverage report instead.
#
# Pattern strategy:
#   *_test.go      → Go never instruments test files; no action needed.
#   mock_*.go      → excluded via package path containing "/mock"
#   mocks/**       → excluded via package path containing "/mocks"
#   testdata/**    → Go skips testdata/ by convention; no action needed.
#   *.pb.go        → files live in packages we mark as generated below
#   *.gen.go       → same
#   vendor/**      → `go list ./...` never returns vendor packages
#   cmd/*/main.go  → excluded via package path matching "MODULE/cmd/"
# =============================================================================
EXCLUDE_PKG_PATTERN := \
	/mock[s/]|\
	\.pb\.go|\
	\.gen\.go|\
	$(MODULE)/cmd/

# All non-excluded packages (used for -coverpkg and go test ./...)
ALL_PKGS := $(shell go list ./... | grep -Ev '$(EXCLUDE_PKG_PATTERN)')

# =============================================================================
# Targets
# =============================================================================

.PHONY: test
## test: run all tests without coverage instrumentation
test:
	go test $(RACE) -timeout $(GOTEST_TIMEOUT) $(ALL_PKGS)

# -----------------------------------------------------------------------------
.PHONY: cover
## cover: run tests with coverage; produce coverage.out aligned with codecov.yml
cover:
	@echo "==> Running tests with coverage (excluded packages mirror codecov.yml)"
	go test $(RACE) \
		-timeout $(GOTEST_TIMEOUT) \
		-covermode=atomic \
		-coverprofile=$(COVERAGE_OUT) \
		-coverpkg=$(shell echo $(ALL_PKGS) | tr ' ' ',') \
		$(ALL_PKGS)
	@echo ""
	@echo "==> Removing generated/vendored lines from $(COVERAGE_OUT)"
	@$(MAKE) --no-print-directory _strip-coverage
	@echo ""
	@echo "==> Coverage summary"
	go tool cover -func=$(COVERAGE_OUT) | tail -1

# -----------------------------------------------------------------------------
.PHONY: cover-html
## cover-html: open an HTML coverage report in the default browser
cover-html: cover
	go tool cover -html=$(COVERAGE_OUT) -o $(COVERAGE_HTML)
	@echo "==> Report written to $(COVERAGE_HTML)"
	@case "$$(uname -s)" in \
		Darwin) open $(COVERAGE_HTML) ;; \
		Linux)  xdg-open $(COVERAGE_HTML) ;; \
		MINGW*|CYGWIN*|MSYS*) start $(COVERAGE_HTML) ;; \
		*) echo "==> Unsupported OS: open $(COVERAGE_HTML) manually" ;; \
	esac

# -----------------------------------------------------------------------------
.PHONY: cover-check
## cover-check: fail the build if total coverage is below COVERAGE_THRESHOLD
cover-check: cover
	@echo "==> Checking coverage threshold (>= $(COVERAGE_THRESHOLD)%)"
	@TOTAL=$$(go tool cover -func=$(COVERAGE_OUT) \
		| awk '/^total:/ { gsub(/%/, "", $$NF); printf "%d", $$NF }'); \
	echo "    Total coverage: $${TOTAL}%"; \
	if [ "$${TOTAL}" -lt "$(COVERAGE_THRESHOLD)" ]; then \
		echo "    FAIL: coverage $${TOTAL}% is below threshold $(COVERAGE_THRESHOLD)%"; \
		exit 1; \
	else \
		echo "    PASS"; \
	fi

# -----------------------------------------------------------------------------
.PHONY: cover-clean
## cover-clean: remove generated coverage artifacts
cover-clean:
	@rm -f $(COVERAGE_OUT) $(COVERAGE_HTML)
	@echo "==> Coverage artifacts removed"

# =============================================================================
# Internal helpers
# =============================================================================

# _strip-coverage: post-processes coverage.out to remove lines that match the
# same file-level patterns as codecov.yml `ignore:`.
# This ensures `go tool cover -func` totals match what Codecov displays.
#
# NOTE: awk program lives in scripts/strip_coverage.awk — avoids shell
# line-continuation parsing that breaks character classes like [^/] when
# Make flattens the backslash-newline sequences before passing to awk.
.PHONY: _strip-coverage
_strip-coverage:
	@awk -f scripts/strip_coverage.awk $(COVERAGE_OUT) > $(COVERAGE_OUT).tmp
	@mv $(COVERAGE_OUT).tmp $(COVERAGE_OUT)