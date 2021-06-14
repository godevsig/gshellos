SHELL=bash

PKG_LIST := $(shell go list ./...)
GIT_TAG := $(shell git describe --tags --abbrev=0 2>/dev/null)
LDFLAGS = -X 'github.com/godevsig/gshellos.version=$(GIT_TAG)'

.PHONY: all dep format build clean test coverage lint vet race help

all: format lint vet test build

format: ## Check coding style
	@DIFF=$$(gofmt -d .); echo -n "$$DIFF"; test -z "$$DIFF"

lint: ## Lint the files
	@golint -set_exit_status ${PKG_LIST}

vet: ## Examine and report suspicious constructs
	@go vet ${PKG_LIST}

test: dep ## Run unittests
	@set -o pipefail; go test -v -short ${PKG_LIST} | tee .test/test.log
	@ERRORS=$$(grep "no test files" .test/test.log); echo "$$ERRORS"; test -z "$$ERRORS"

race:  ## Run data race detector
	@go test -race -short ${PKG_LIST}

COVER_GOAL := 80
coverage: dep ## Generate global code coverage report
	@go test -covermode=count -coverpkg="./..." -coverprofile .test/l1_coverage.cov $(PKG_LIST)
	@echo "mode: count" > .test/final_coverage.out
	@cat `find -name "*.cov"` | grep -v "mode: count" >> .test/final_coverage.out
	@go tool cover -func=.test/final_coverage.out | tee .test/final_coverage.log
	@tail .test/final_coverage.log -n1 | awk -F"\t*| *|%" '{if ($$3<${COVER_GOAL}) {print "Coverage goal: ${COVER_GOAL}% not reached"; exit 1}}'

dep: ## Get the dependencies
	@mkdir -p bin .test

build: release

release: dep ## Build release binary file to bin dir
	@go build -ldflags="$(LDFLAGS)" -o bin ./cmd/gshell

debug: dep ## Build debug binary file to bin dir
	@go build -ldflags="$(LDFLAGS)" -o bin -tags debug ./cmd/gshell 

clean: ## Remove previous build and test files
	@rm -rf bin `find -name "\.test"`
	@rm -rf bin `find -name "test"`

help: ## Display this help screen
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
