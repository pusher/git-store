include .env

RED := \033[31m
GREEN := \033[32m
NC := \033[0m

.NOTPARALLEL:

.PHONY: all
all: distclean test

.PHONY: distclean
distclean:
	rm -rf vendor

.PHONY: fmt
fmt:
	$(GO) fmt ./...

.PHONY: vet
vet: vendor
	$(GO) vet ./...

.PHONY: lint
lint:
	@ echo "$(GREEN)Linting code$(NC)"
	$(LINTER) run --disable-all \
		--exclude-use-default=false \
		--enable=govet \
		--enable=ineffassign \
		--enable=deadcode \
		--enable=golint \
		--enable=goconst \
		--enable=gofmt \
		--enable=goimports \
		--skip-dirs=pkg/client/ \
		--deadline=120s \
		--tests ./...
	@ echo

vendor:
	@ echo "$(GREEN)Pulling dependencies$(NC)"
	$(DEP) ensure --vendor-only
	@ echo

.PHONY: test
test: vendor
	@ echo "$(GREEN)Running test suite$(NC)"
	$(GINKGO) -v -race -randomizeAllSpecs ./... -- -report-dir=$$ARTIFACTS
	@ echo

.PHONY: check
check: fmt lint vet test
