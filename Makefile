# Run `make` for the list. `make check` is what CI runs.
.DEFAULT_GOAL := help
SHELL := /bin/bash

STORE ?= memory
FOLLOW ?= --follow
COMPOSE := docker compose -f compose.yaml \
	$(if $(filter mysql,$(STORE)),-f compose.mysql.yaml) \
	$(if $(filter mongodb,$(STORE)),-f compose.mongodb.yaml) \
	$(if $(OBSERVABILITY),-f compose.observability.yaml)
TAGS := persistence_mysql persistence_mongodb
BUF := go run github.com/bufbuild/buf/cmd/buf@v1.75.0
STATICCHECK := go run honnef.co/go/tools/cmd/staticcheck@v0.8.2
GOVULNCHECK := go run golang.org/x/vuln/cmd/govulncheck@v1.9.0
KUBECONFORM := go run github.com/yannh/kubeconform/cmd/kubeconform@v0.8.0
CHART := deploy/helm/go-api-boilerplate

.PHONY: help
help: ## Show this list
	@grep -hE '^[a-z0-9-]+:.*## ' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  %-16s %s\n", $$1, $$2}'

.PHONY: check
check: check-go check-contracts check-web check-deploy ## Everything CI runs, except containers and e2e

.PHONY: check-go
check-go: ## Format, vet and lint every build, then unit tests (no Docker)
	@test -z "$$(gofmt -l .)" || (gofmt -l . && echo "run gofmt -w ." && false)
	go fix -diff ./...
	go vet ./...
	$(STATICCHECK) ./...
	@for tag in $(TAGS); do echo "vet and staticcheck -tags $$tag"; go vet -tags $$tag ./... && $(STATICCHECK) -tags $$tag ./... || exit 1; done
	go test -race -shuffle=on -short ./...

.PHONY: test
test: ## All Go tests, including the MySQL and MongoDB containers
	go test -race -shuffle=on ./...

.PHONY: vuln
vuln: ## Check dependencies for known vulnerabilities
	$(GOVULNCHECK) ./...

.PHONY: check-contracts
check-contracts: gen ## Lint the proto and fail if generated files are stale
	$(BUF) lint
	@git diff --exit-code -- gen website/src/content/docs/reference || (echo "generated files changed: commit the output of make gen" && false)

.PHONY: check-web
check-web: ## Typecheck, lint, test and build the web app
	cd web && npm ci && npm run check && npm run build

.PHONY: check-deploy
check-deploy: ## Lint the Helm chart and validate it for each store
	@for store in memory mysql mongodb; do \
		helm lint $(CHART) --strict --quiet -f $(CHART)/ci/$$store-values.yaml && \
		helm template ci $(CHART) -f $(CHART)/ci/$$store-values.yaml | $(KUBECONFORM) -strict -summary \
			-schema-location default \
			-schema-location 'https://raw.githubusercontent.com/datreeio/CRDs-catalog/main/{{.Group}}/{{.ResourceKind}}_{{.ResourceAPIVersion}}.json' || exit 1; \
	done

.PHONY: gen
gen: ## Regenerate Connect stubs and the configuration reference
	$(BUF) generate
	go run ./cmd/auth env > website/src/content/docs/reference/auth-configuration.md
	go run ./cmd/user env > website/src/content/docs/reference/user-configuration.md

.PHONY: up
up: ## Start the stack (STORE=memory|mysql|mongodb, OBSERVABILITY=1 adds Grafana)
	$(COMPOSE) up --build --wait

.PHONY: down
down: ## Stop the stack and remove its volumes
	$(COMPOSE) down --volumes

.PHONY: logs
logs: ## Follow the services' logs (FOLLOW= prints them and exits)
	$(COMPOSE) logs $(FOLLOW) auth user

.PHONY: e2e
e2e: ## End-to-end tests against a running stack (make up first)
	go test -tags e2e -count=1 ./e2e/...

.PHONY: e2e-web
e2e-web: ## Browser tests against a running stack
	cd web && E2E_STACK_UP=1 npm run e2e
