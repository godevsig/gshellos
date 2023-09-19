SHELL=bash

PKG_ALL = $(shell go list ./...)
GIT_TAG = $(shell git describe --tags --abbrev=0 2>/dev/null)
COMMIT_REV = $(shell git rev-parse HEAD)
STDTAGS := stdbase,stdcommon,stdruntime
EXTTAGS := adaptiveservice,shell,log,pidinfo,asbench

all: format lint vet test build

format: ## Check coding style
	@DIFF=$$(gofmt -d .); echo -n "$$DIFF"; test -z "$$DIFF"

lint: dep ## Lint the files
	@golint -set_exit_status ${PKG_ALL}

vet: dep ## Examine and report suspicious constructs
	@go vet ${PKG_ALL}

testbin: STDTAGS := $(STDTAGS),stdhttp,stdlog
testbin: LDFLAGS += -X 'github.com/godevsig/gshellos.updateInterval=5'
testbin: dep ## Generate test version of main binary
	@go test -tags $(STDTAGS),$(EXTTAGS) -ldflags="$(LDFLAGS)" -covermode=count -coverpkg="./..." -c -o bin/gshell.tester .
	@ln -snf gshell.tester bin/gshell.test

rmtestfiles:
	@rm -rf .working; rm -rf .test; rm -f default.joblist.yaml

test: rmtestfiles testbin ## Run unit tests
	@PATH=$$PATH:`pwd`/bin gshell.test -test.v -test.run TestCmd
	@PATH=$$PATH:`pwd`/bin gshell.test -test.v -test.run TestAutoUpdate

COVER_GOAL := 78
coverage: rmtestfiles testbin ## Generate global code coverage report
	@PATH=$$PATH:`pwd`/bin gshell.test -test.v -test.run TestCmd -test.coverprofile .test/gshell_coverage.cov
	@PATH=$$PATH:`pwd`/bin gshell.test -test.v -test.run TestAutoUpdate -test.coverprofile .test/gshell_update_coverage.cov
	@echo "mode: count" > .test/final_coverage.out
	@cat `find -name "*.cov"` | grep -E -v "mode: count|/extension/|/stdlib/" >> .test/final_coverage.out
	@go tool cover -func=.test/final_coverage.out | tee .test/final_coverage.log
	@tail .test/final_coverage.log -n1 | awk -F"\t*| *|%" '{if ($$3<${COVER_GOAL}) {print "Coverage goal: ${COVER_GOAL}% not reached"; exit 1}}'

dep:
	@mkdir -p bin .test
	@echo -n $(GIT_TAG) > bin/gittag
	@echo -n $(STDTAGS),$(EXTTAGS) > bin/buildtag
	@echo -n $(COMMIT_REV) > bin/rev

build: dep
	@go build -tags $(STDTAGS),$(EXTTAGS) -ldflags="$(LDFLAGS)" -o bin ./cmd/gshell

lite: LDFLAGS += -s -w
lite: EXTTAGS := $(EXTTAGS),echomsg,topidchartmsg,recordermsg
lite: build ## Build lite release binary to bin dir

full: EXTTAGS := debug,$(EXTTAGS),echo,fileserver,topidchart,docit,recorder
full: STDTAGS := $(STDTAGS),stdext,stdarchive,stdcompress,stdcontainer,stdcrypto,stddatabase,stdencoding
full: STDTAGS := $(STDTAGS),stdhash,stdhtml,stdlog,stdmath,stdhttp,stdmail,stdrpc,stdregexp,stdtext,stdunicode
full: build ## Build full release binary to bin dir

generate: gen-extlib gen-stdlib ## Generate libraries

gen-extlib: extractbin
	@go generate github.com/godevsig/gshellos/extension

check-extlib: gen-extlib
	@echo Checking if the generated files were forgotten to commit...
	@DIFF=$$(git diff); echo -n "$$DIFF"; test -z "$$DIFF"

gen-stdlib: extractbin
	@go generate github.com/godevsig/gshellos/stdlib

extractbin:
	@go build -o cmd/extract ./cmd/extract

clean: rmtestfiles ## Remove previous build and test files
	@rm -rf bin `find -name "\.test"` `find -name "test"`
	@rm -f cmd/extract/extract

help: ## Display this help screen
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
