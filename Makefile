.DEFAULT_GOAL := help
.PHONY: build build-gui build-debug build-tests test test-all test-ptr64 test-go vet vet-gui check-tiers check-inits check-ifaces check build-parser clean help

CXEXE := cx
EXE :=
ifeq ($(OS),Windows_NT)
CXEXE := cx.exe
EXE := .exe
endif

# Default build includes cipher, cxos and regexp. Opt out with nocipher,
# nocxos, noregexp. ptr64 switches the VM to 64 bit pointers.
TAGS ?=
GUI_TAGS := cxfx

build: ## Build cx into ./bin
	go build -tags="$(TAGS)" -o ./bin/$(CXEXE) ./t11-cx

build-gui: ## Build cx with the cxfx GL/GLFW/OpenAL package (needs cgo and X11/GL headers)
	go build -tags="$(TAGS) $(GUI_TAGS)" -o ./bin/$(CXEXE) ./t11-cx

build-debug: ## Build cx without optimizations
	go build -gcflags="all=-N -l" -tags="$(TAGS)" -o ./bin/$(CXEXE) ./t11-cx

build-tests: build ## Build the cx program test runner
	go build -o ./bin/cxtest ./t1-cxtest

test: build-tests ## Run the stable CX program tests
	./bin/cxtest --cxpath=$(CURDIR)/bin/$(CXEXE) --wdir=./tests --log=fail,stderr --disable-tests=gui,issue

test-all: build-tests ## Run all non-GUI CX program tests, including known issues
	./bin/cxtest --cxpath=$(CURDIR)/bin/$(CXEXE) --wdir=./tests --log=fail,stderr --disable-tests=gui

test-ptr64: ## Run the stable CX program tests with a ptr64 build
	go build -tags="$(TAGS) ptr64" -o ./bin/cx64$(EXE) ./t11-cx
	go build -o ./bin/cxtest ./t1-cxtest
	./bin/cxtest --cxpath=$(CURDIR)/bin/cx64$(EXE) --wdir=./tests --log=fail,stderr --disable-tests=gui,issue

test-go: ## Run the Go unit tests
	go test ./...

vet: ## go vet with default and ptr64 tags
	go vet ./...
	go vet -tags ptr64 ./t0-types ./t3-ast

vet-gui: ## go vet with the cxfx tag (needs cgo and X11/GL/OpenAL headers)
	go vet -tags cxfx ./t6-cxfx ./t7-init ./t11-cx

check-tiers: ## Check that packages only import lower tiers and sit at <dir>/<file>.go
	./check-tiers.sh

check-inits: ## Check that no package outside main has func init(), see init-allowlist.txt
	./check-inits.sh

check-ifaces: ## Check that no interface embeds another interface, see iface-allowlist.txt
	./check-ifaces.sh

check: check-tiers check-inits check-ifaces vet test-go test test-ptr64 ## tier, init and interface checks, vet, Go tests and CX program tests (default and ptr64)

build-parser: ## Regenerate the CX parsers from the .y grammars
	go tool goyacc -o t5-cxpartialparsing/cxpartialparsing.go -v /dev/null t5-cxpartialparsing/cxpartialparsing.y
	go tool goyacc -o t8-cxparsingcompletor/parsingcompletor.go -v /dev/null t8-cxparsingcompletor/parsingcompletor.y
	gofmt -w t5-cxpartialparsing t8-cxparsingcompletor

clean: ## Remove binaries
	rm -rf ./bin

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-14s\033[0m %s\n", $$1, $$2}'
