.DEFAULT_GOAL := help
.PHONY: build build-gui build-debug build-wasm test-wasm build-tests test test-all test-ptr64 test-go vet vet-gui check-tiers check-inits check-ifaces check-recovers check-objmodel generate-objmodel check build-parser bench-compare clean help

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

build-wasm: ## Compile check cx for js/wasm with the default tags, as the playground builds it
	GOOS=js GOARCH=wasm go build -tags="$(TAGS)" -o /dev/null ./t11-cx

test-wasm: ## Build cx for js/wasm and run the hello world example under Node
	GOOS=js GOARCH=wasm go build -tags="$(TAGS)" -o ./bin/cx.wasm ./t11-cx
	node "$$(go env GOROOT)/lib/wasm/wasm_exec_node.js" ./bin/cx.wasm tests/examples/hello-world.cx | grep -qx "Hello World!"

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

vet: ## go vet with default and ptr64 tags, and gofmt
	go vet ./...
	@files=$$(gofmt -l $$(git ls-files '*.go' | grep -v '^vendor/')); test -z "$$files" || { echo "gofmt -l lists: $$files"; exit 1; }
	go vet -tags ptr64 ./t0-types ./t3-ast

vet-gui: ## go vet with the cxfx tag (needs cgo and X11/GL/OpenAL headers)
	go vet -tags cxfx ./t6-cxfx ./t7-init ./t11-cx

check-tiers: ## Check that packages only import lower tiers and sit at <dir>/<file>.go
	./check-tiers.sh

check-inits: ## Check that no package outside main has func init(), see init-allowlist.txt
	./check-inits.sh

check-ifaces: ## Check that no interface embeds another interface, see iface-allowlist.txt
	./check-ifaces.sh

check-recovers: ## Check that no non-vendor code calls recover(, see recover-allowlist.txt
	./check-recovers.sh

check-objmodel: ## Check the object model rules of docs/ObjectModel.md and that the generated managers are up to date
	./check-objmodel.sh

generate-objmodel: ## Regenerate the Gid, manager and manager test files of t3-ast and the site counts of docs/ObjectModel.md
	go run ./t0-cx-core-objmodelgen -dir t3-ast

check: check-tiers check-inits check-ifaces check-recovers check-objmodel vet test-go test test-ptr64 ## tier, init, interface, recover and object model checks, vet, Go tests and CX program tests (default and ptr64)

bench-compare: ## Time the t0-benchmarks/compare programs in cx, C, Python 3 and Bash (needs clang, python3, taskset)
	go build -tags="$(TAGS)" -o ./bin/$(CXEXE) ./t11-cx
	go build -tags="$(TAGS) cxopcount" -o ./bin/cx-opcount$(EXE) ./t11-cx
	./t0-benchmarks/compare/run.sh

build-parser: ## Regenerate the CX parsers from the .y grammars
	go tool goyacc -o t5-cxpartialparsing/cxpartialparsing.go -v /dev/null t5-cxpartialparsing/cxpartialparsing.y
	go tool goyacc -o t8-cxparsingcompletor/parsingcompletor.go -v /dev/null t8-cxparsingcompletor/parsingcompletor.y
	gofmt -w t5-cxpartialparsing t8-cxparsingcompletor

clean: ## Remove binaries
	rm -rf ./bin

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-14s\033[0m %s\n", $$1, $$2}'
