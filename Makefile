SHELL=bash

PLUGIN :=
LDFLAGS :=
CGO := 0
PKG_ALL = $(shell go list ./... | grep -v unsafe)
GIT_TAG ?= $(shell git describe --tags --abbrev=0 2>/dev/null)
BUILD_TIME = $(shell date "+%Y.%m.%d %H:%M:%S")
STDTAGS := stdbase,stdcommon,stdruntime
EXTTAGS := adaptiveservice,shell,log,pidinfo,asbench

format: ## Check coding style
	@DIFF=$$(gofmt -d .); echo -n "$$DIFF"; test -z "$$DIFF"

lint: dep ## Lint the files
	@golint -set_exit_status ${PKG_ALL}

vet: dep ## Examine and report suspicious constructs
	@go vet ${PKG_ALL}

testbin: PLUGIN := ,plugin
testbin: EXTTAGS := debug,$(EXTTAGS)
testbin: STDTAGS := $(STDTAGS),stdhttp,stdlog
testbin: LDFLAGS += -X github.com/godevsig/gshellos.updateInterval=5
testbin: dep ## Generate test version of main binary
	@CGO_ENABLED=1 go test -tags $(STDTAGS),$(EXTTAGS)$(PLUGIN) -ldflags="$(LDFLAGS)" -covermode=count -coverpkg="./..." -c -o bin/gshell.tester .
	@ln -snf gshell.tester bin/gshell.test

pluginfiles: gsh-symbol-tools
	@mkdir -p .plugins
	@cd .plugins; PATH=`pwd`/../bin:$$PATH gsh-gen-symbols -plugin github.com/godevsig/grepo/echo; \
		CGO_ENABLED=1 go build -buildmode=plugin github_com-godevsig-grepo-echo.go; \
		mv github_com-godevsig-grepo-echo.so github_com-godevsig-grepo-echo.gp
	@cd .plugins; PATH=`pwd`/../bin:$$PATH gsh-gen-symbols -plugin github.com/godevsig/grepo/topidchart; \
		CGO_ENABLED=1 go build -buildmode=plugin github_com-godevsig-grepo-topidchart.go; \
		mv github_com-godevsig-grepo-topidchart.so github_com-godevsig-grepo-topidchart.gp
	@cd .plugins; PATH=`pwd`/../bin:$$PATH gsh-gen-symbols -plugin github.com/godevsig/grepo/docit; \
		CGO_ENABLED=1 go build -buildmode=plugin github_com-godevsig-grepo-docit.go; \
		mv github_com-godevsig-grepo-docit.so github_com-godevsig-grepo-docit.gp
	@cd .plugins; cp github_com-godevsig-grepo-echo.go wrong-format-test.gp

rmtestfiles:
	@rm -rf .working .test .plugins; rm -f default.joblist.yaml

test: clean pluginfiles testbin ## Run unit tests
	@PATH=`pwd`/bin:$$PATH gshell.test -test.v -test.run TestCmd
	@PATH=`pwd`/bin:$$PATH gshell.test -test.v -test.run TestClientServer
	@PATH=`pwd`/bin:$$PATH gshell.test -test.v -test.run TestAutoUpdate

COVER_GOAL := 80
coverage: clean pluginfiles testbin ## Generate global code coverage report
	@PATH=`pwd`/bin:$$PATH gshell.test -test.v -test.run TestCmd -test.coverprofile .test/gshell_coverage.cov
	@PATH=`pwd`/bin:$$PATH gshell.test -test.v -test.run TestClientServer -test.coverprofile .test/gshell_clientserver_coverage.cov
	@PATH=`pwd`/bin:$$PATH gshell.test -test.v -test.run TestAutoUpdate -test.coverprofile .test/gshell_update_coverage.cov
	@echo "mode: count" > .test/final_coverage.out
	@cat `find -name "*.cov"` | grep -E -v "mode: count|/extension/|/stdlib/" >> .test/final_coverage.out
	@go tool cover -func=.test/final_coverage.out | tee .test/final_coverage.log
	@tail .test/final_coverage.log -n1 | awk -F"\t*| *|%" '{if ($$3<${COVER_GOAL}) {print "Coverage goal: ${COVER_GOAL}% not reached"; exit 1}}'

dep:
	@mkdir -p bin .test
	@echo -n $(GIT_TAG) > bin/version
	@echo -n $(STDTAGS),$(EXTTAGS) > bin/buildtag
	@echo -n $(BUILD_TIME) > bin/buildtime

build: dep
	@CGO_ENABLED=$(CGO) go build -tags $(STDTAGS),$(EXTTAGS)$(PLUGIN) -ldflags="$(LDFLAGS)" -o bin ./cmd/gshell

lite: EXTTAGS := $(EXTTAGS),echomsg,topidchartmsg,recordermsg
lite: build ## Build with lite feature set, no cgo, no plugin

lite-strip: LDFLAGS += -s -w
lite-strip: lite ## Same as lite, but with symbols stripped

full: EXTTAGS := debug,$(EXTTAGS),echo,fileserver,topidchart,docit,recorder
full: STDTAGS := $(STDTAGS),stdarchive,stdcompress,stdcontainer,stdcrypto,stddatabase,stdencoding
full: STDTAGS := $(STDTAGS),stdhash,stdhtml,stdlog,stdmath,stdhttp,stdmail,stdrpc,stdregexp,stdtext,stdunicode
full: build ## Build with full feature set, no cgo, no plugin

full-plugin: CGO := 1
full-plugin: PLUGIN := ,plugin
full-plugin: full ## Similar to full, with plugin support, dynamically linked

generate: gen-extlib gen-stdlib ## Generate libraries

gen-extlib: gsh-symbol-tools
	@PATH=`pwd`/bin:$$PATH go generate github.com/godevsig/gshellos/extension

check-extlib: gen-extlib
	@echo Checking if the generated files were forgotten to commit...
	@DIFF=$$(git diff); echo -n "$$DIFF"; test -z "$$DIFF"

gen-stdlib: gsh-symbol-tools
	@PATH=`pwd`/bin:$$PATH go generate github.com/godevsig/gshellos/stdlib

gsh-symbol-tools:
	@mkdir -p bin
	@go build -ldflags="-s -w" -o cmd/extract/extract ./cmd/extract
	@cp cmd/extract/extract bin/gsh-extract
	@cp extension/gsh-gen-symbols bin/

clean: rmtestfiles ## Remove previous build and test files
	@rm -rf bin `find -name "\.test"` `find -name "test"`
	@rm -f cmd/extract/extract
	@rm -f *.log

help: ## Display this help screen
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
