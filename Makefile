.DEFAULT_GOAL := help
.PHONY: records build build-gui build-debug build-wasm test-wasm build-tests test test-all test-ptr64 test-go vet vet-gui check-tiers check-inits check-ifaces check-json check-recovers check-objmodel check-strhot check-strhot-gui test-gui generate-objmodel names-csv check build-parser bench-compare clean help

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

# CORE_LIBRARY_SETS are the system library sets of the cx build cores, one
# core each, cx-core for none and cx-core-file-net for file,net.
CORE_LIBRARY_SETS := none file net process file,net file,process net,process file,net,process \
	cipher file,cipher net,cipher process,cipher file,net,cipher file,process,cipher net,process,cipher file,net,process,cipher

# names-csv output file and extra repository paths to scan.
NAMES_CSV ?= bin/names.csv
REPOS ?=

build: ## Build cx and the cx build cores, cx-core and one per system library set, into ./bin
	go build -tags="$(TAGS)" -o ./bin/$(CXEXE) ./t11-cx
	@for set in $(CORE_LIBRARY_SETS); do \
		suffix=$$(echo "-$$set" | sed "s/^-none$$//; s/,/-/g"); \
		tags=$$(echo "$$set" | sed "s/^none$$//; s/[a-z][a-z]*/cxcorelibrary&/g; s/,/ /g"); \
		echo go build -tags=\"$(TAGS) $$tags\" -o ./bin/cx-core$$suffix$(EXE) ./t7-cx-core-run; \
		go build -tags="$(TAGS) $$tags" -o ./bin/cx-core$$suffix$(EXE) ./t7-cx-core-run || exit 1; \
	done

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

test-all: build-tests ## Run all non-GUI CX program tests, including any marked as known issues (issue-217, a GUI test, so none runs)
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

test-gui: ## Build cx with cxfx and run the gui CX program tests, under xvfb-run when DISPLAY is unset (needs cgo, X11/GL/OpenAL headers and a GL driver)
	go build -tags="$(TAGS) $(GUI_TAGS)" -o ./bin/cx-gui$(EXE) ./t11-cx
	go build -o ./bin/cxtest ./t1-cxtest
	$(if $(DISPLAY),,xvfb-run -a) ./bin/cxtest --cxpath=$(CURDIR)/bin/cx-gui$(EXE) --wdir=./tests --log=fail,stderr --enable-tests=gui

check-tiers: ## Check that packages only import lower tiers and sit at <dir>/<file>.go
	./check-tiers.sh

check-inits: ## Check that no package outside main has func init(), see init-allowlist.txt
	./check-inits.sh

check-ifaces: ## Check that no interface embeds another interface, see iface-allowlist.txt
	./check-ifaces.sh

check-json: ## Check that no core package imports encoding/json, every package but vendor and t0-cx-tools-*, see skycoin-cx#509
	./check-json.sh

check-recovers: ## Check that no non-vendor code calls recover(, see recover-allowlist.txt
	./check-recovers.sh

check-objmodel: ## Check the object model rules of docs/ObjectModel.md and that the generated managers are up to date
	./check-objmodel.sh

check-gcreach: ## Check that the vm and IR base stay GC free and every other GC reach has a GCREACH comment, see docs/GarbageCollector.md
	./check-gcreach.sh

check-strhot: ## Check that every string operation on the run path of the old loop has a STRHOT code, see docs/StringsHotPath.md. Checks t6-cxfx too when the cxfx build compiles
	./check-strhot.sh

check-strhot-gui: ## check-strhot, failing when the cxfx build of t6-cxfx does not compile (needs cgo and X11/GL/OpenAL headers)
	CX_STRHOT_CXFX=1 ./check-strhot.sh

names-csv: ## Write one CSV row per declared Go and CX name to NAMES_CSV, for skycoin-cx and the repositories in REPOS
	mkdir -p $(dir $(NAMES_CSV))
	go run ./t0-cx-tools-names -out $(NAMES_CSV) . $(REPOS)

generate-opcodes: ## Regenerate the opcode signature tables from t1-constants/opcodes_definitions.go, decision D21
	go run ./t2-cx-core-opcodegen -root .

check-opcodes: ## Check that the opcode signature tables are up to date with t1-constants/opcodes_definitions.go
	go run ./t2-cx-core-opcodegen -root . -check

generate-objmodel: ## Regenerate the Gid, manager and manager test files of t3-ast and the site counts of docs/ObjectModel.md
	go run ./t0-cx-core-objmodelgen -dir t3-ast

check: check-tiers check-inits check-ifaces check-json check-recovers check-objmodel check-opcodes check-gcreach check-strhot vet test-go test test-ptr64 ## tier, init, interface, JSON, recover, object model, opcode table, GC reach and STRHOT checks, vet, Go tests and CX program tests (default and ptr64)

bench-compare: ## Time the t0-benchmarks/compare programs in cx, C, Python 3 and Bash (needs clang, python3, taskset)
	go build -tags="$(TAGS)" -o ./bin/$(CXEXE) ./t11-cx
	go build -tags="$(TAGS) cxopcount" -o ./bin/cx-opcount$(EXE) ./t11-cx
	./t0-benchmarks/compare/run.sh

records: ## Write tests/execution-records.txt from the old loop, see docs/ExecutionRecords.md. Review the diff before committing it
	CX_EXECUTION_RECORDS_WRITE=1 go test -count=1 -run '^TestExecutionRecords$$' ./t11-cx

build-parser: ## Regenerate the CX parsers from the .y grammars
	go tool goyacc -o t5-cxpartialparsing/cxpartialparsing.go -v /dev/null t5-cxpartialparsing/cxpartialparsing.y
	go tool goyacc -o t8-cxparsingcompletor/parsingcompletor.go -v /dev/null t8-cxparsingcompletor/parsingcompletor.y
	gofmt -w t5-cxpartialparsing t8-cxparsingcompletor

clean: ## Remove binaries
	rm -rf ./bin

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-14s\033[0m %s\n", $$1, $$2}'
