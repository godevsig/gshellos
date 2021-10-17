SHELL=bash

PKG_ALL := $(shell go list ./...)
PKG_LIST := $(shell go list ./... | grep -E -v "gshellos$$|/cmd|extension$$|stdlib$$|scriptlib$$")
GIT_TAG := $(shell git describe --tags --abbrev=0 2>/dev/null)
COMMIT_REV := $(shell git rev-parse HEAD)
BLDTAGS := adaptiveservice
BLDTAGS := $(BLDTAGS),stdbase,stdcommon
LDFLAGS = -X 'github.com/godevsig/gshellos.version=$(GIT_TAG)' -X 'github.com/godevsig/gshellos.buildTags=$(BLDTAGS)'
LDFLAGS += -X 'github.com/godevsig/gshellos.commitRev=$(COMMIT_REV)'

all: format lint vet test build

format: ## Check coding style
	@DIFF=$$(gofmt -d .); echo -n "$$DIFF"; test -z "$$DIFF"

lint: ## Lint the files
	@golint -set_exit_status ${PKG_ALL}

vet: ## Examine and report suspicious constructs
	@go vet ${PKG_ALL}

testbin: LDFLAGS += -X 'github.com/godevsig/gshellos.updateInterval=5'
testbin: dep ## Generate test version of main binary
	@go test -tags $(BLDTAGS) -ldflags="$(LDFLAGS)" -covermode=count -coverpkg="./..." -c -o bin/gshell.tester .
	@ln -snf gshell.tester bin/gshell.test

test: testbin ## Run unit tests
	@PATH=$$PATH:`pwd`/bin gshell.test -test.v -test.run TestCmd
	@PATH=$$PATH:`pwd`/bin gshell.test -test.v -test.run TestAutoUpdate
	@set -o pipefail; go test -v -short ${PKG_LIST} | tee .test/test.log
	@ERRORS=$$(grep "no test files" .test/test.log); echo "$$ERRORS"; test -z "$$ERRORS"

race:  ## Run data race detector
	@go test -race -short ${PKG_LIST}

COVER_GOAL := 80
coverage: testbin ## Generate global code coverage report
	@PATH=$$PATH:`pwd`/bin gshell.test -test.v -test.run TestCmd -test.coverprofile .test/gshell_coverage.cov
	@PATH=$$PATH:`pwd`/bin gshell.test -test.v -test.run TestAutoUpdate -test.coverprofile .test/gshell_update_coverage.cov
	@go test -covermode=count -coverpkg="./..." -coverprofile .test/l1_coverage.cov $(PKG_LIST)
	@echo "mode: count" > .test/final_coverage.out
	@cat `find -name "*.cov"` | grep -E -v "mode: count|/extension/|/stdlib/|/scriptlib/" >> .test/final_coverage.out
	@go tool cover -func=.test/final_coverage.out | tee .test/final_coverage.log
	@tail .test/final_coverage.log -n1 | awk -F"\t*| *|%" '{if ($$3<${COVER_GOAL}) {print "Coverage goal: ${COVER_GOAL}% not reached"; exit 1}}'

dep:
	@mkdir -p bin .test

build: dep
	@go build -tags $(BLDTAGS) -ldflags="$(LDFLAGS)" -o bin ./cmd/gshell

lite: LDFLAGS += -s -w
lite: build ## Build lite release binary to bin dir

FULLTAGS := $(BLDTAGS),stdext,stdarchive,stdcompress,stdcontainer,stdcrypto,stddatabase,stdencoding
FULLTAGS := $(FULLTAGS),stdhash,stdhtml,stdlog,stdmath,stdhttp,stdmail,stdrpc,stdregexp,stdruntime,stdtext,stdunicode
FULLTAGS := $(FULLTAGS),debug
full: BLDTAGS := $(FULLTAGS)
full: build ## Build full release binary to bin dir

generate: gen-extlib gen-stdlib ## Generate libraries

gen-extlib: extractbin
	@go generate github.com/godevsig/gshellos/extension

gen-stdlib: extractbin
	@go generate github.com/godevsig/gshellos/stdlib

extractbin:
	@go build -o cmd/extract ./cmd/extract

clean: ## Remove previous build and test files
	@rm -rf bin `find -name "\.test"` `find -name "test"`

help: ## Display this help screen
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
