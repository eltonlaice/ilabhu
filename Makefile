# ilabhu — common dev tasks.
#
# Usage: `make <target>` from the repo root. `make` (no target) prints help.
#
# Self-documenting: each public target has a `## description` comment that the
# `help` target parses and renders.

.DEFAULT_GOAL := help
.PHONY: help build test lint vet fmt run web-install web-dev web-build web-lint smoke check clean

# ----- Backend (control-plane) -----

build: ## Build the ilabhud binary into control-plane/bin/
	cd control-plane && go build -o bin/ilabhud ./cmd/ilabhud

test: ## Run all Go tests
	cd control-plane && go test ./...

vet: ## Run `go vet`
	cd control-plane && go vet ./...

lint: ## Run golangci-lint over the control plane
	cd control-plane && golangci-lint run ./...

fmt: ## Format Go sources
	cd control-plane && gofmt -w .

run: ## Run ilabhud against ./exams on :8080 (Ctrl-C to stop)
	cd control-plane && go run ./cmd/ilabhud -addr :8080 -exams-dir ../exams

# ----- Frontend (web) -----

web-install: ## Install web dependencies (npm ci)
	cd web && npm ci

web-dev: ## Run the Next.js dev server on :3000
	cd web && npm run dev

web-build: ## Build the web app for production
	cd web && npm run build

web-lint: ## Lint the web app
	cd web && npm run lint

# ----- Composite -----

check: vet lint test web-lint web-build ## Everything CI checks (no run)

smoke: ## Start ilabhud, hit /healthz and /v1/exams, then stop
	@echo "starting ilabhud on :18080…"
	@cd control-plane && go run ./cmd/ilabhud -addr :18080 -exams-dir ../exams -state-dir /tmp/ilabhu-smoke > /tmp/ilabhu-smoke.log 2>&1 & \
	  pid=$$!; \
	  trap "kill $$pid 2>/dev/null; rm -rf /tmp/ilabhu-smoke /tmp/ilabhu-smoke.log" EXIT INT TERM; \
	  until curl -sf http://127.0.0.1:18080/healthz >/dev/null; do sleep 1; done; \
	  echo "/healthz:" && curl -s http://127.0.0.1:18080/healthz && echo && \
	  echo "/v1/exams:" && curl -s http://127.0.0.1:18080/v1/exams

clean: ## Remove build artefacts
	rm -rf control-plane/bin web/.next web/out

help: ## Show this help
	@awk 'BEGIN {FS = ":.*?## "} \
	     /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}' \
	     $(MAKEFILE_LIST) | sort
