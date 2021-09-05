SHELL=bash

PKG_LIST := $(shell go list ./...)
GIT_TAG := $(shell git describe --tags --abbrev=0 2>/dev/null)
BLDTAGS := stdlib,adaptiveservice
LDFLAGS = -X 'github.com/godevsig/gshellos.version=$(GIT_TAG)' -X 'github.com/godevsig/gshellos.buildTags=$(BLDTAGS)'

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

dep:
	@mkdir -p bin .test

build: dep
	@go build -tags $(BLDTAGS) -ldflags="$(LDFLAGS)" -o bin ./cmd/gshell

debug: BLDTAGS := $(BLDTAGS),debug
debug: build ## Build debug binary to bin dir

lite: BLDTAGS := $(BLDTAGS)
lite: build ## Build lite release binary to bin dir

full: BLDTAGS := $(BLDTAGS),echart,database
full: build ## Build full release binary to bin dir

clean: ## Remove previous build and test files
	@rm -rf bin `find -name "\.test"`
	@rm -rf bin `find -name "test"`

help: ## Display this help screen
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
