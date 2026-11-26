.DEFAULT_GOAL := help
.PHONY: records llvm-clang build build-tinygo build-wasm-tinygo release-archives release-notes lint vulncheck build-gui build-debug build-wasm test-wasm build-tests test test-all test-ptr64 test-go vet vet-gui check-tiers check-inits check-ifaces check-json check-recovers check-objmodel check-names check-strhot check-strhot-gui test-gui generate-objmodel names-csv check build-parser bench-compare clean help

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

# The golangci-lint of skycoin/skycoin and skycoin/skywire, and govulncheck.
GOLANGCI_LINT_VERSION := v2.15.0
GOVULNCHECK_VERSION := v1.10.0

# names-csv output file and extra repository paths to scan.
NAMES_CSV ?= bin/names.csv
REPOS ?=

# release-archives writes one archive per RELEASE_PLATFORMS entry, and
# SHA256SUMS, into RELEASE_DIR for the tag VERSION.
VERSION ?=
RELEASE_DIR ?= bin/release
RELEASE_PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64

# CORE_BUILD is recipe shell that builds every cx build core into $$coredir,
# with the file suffix $$exe and the go build flags $$goflags.
CORE_BUILD = for set in $(CORE_LIBRARY_SETS); do \
		suffix=$$(echo "-$$set" | sed "s/^-none$$//; s/,/-/g"); \
		tags=$$(echo "$$set" | sed "s/^none$$//; s/[a-z][a-z]*/cxcorelibrary&/g; s/,/ /g"); \
		echo go build $$goflags -tags=\"$(TAGS) cxcore $$tags\" -o $$coredir/cx-core$$suffix$$exe ./t6-cx-core-run; \
		go build $$goflags -tags="$(TAGS) cxcore $$tags" -o $$coredir/cx-core$$suffix$$exe ./t6-cx-core-run || exit 1; \
	done

build: ## Build cx and the cx build cores, cx-core and one per system library set, into ./bin
	go build -tags="$(TAGS)" -o ./bin/$(CXEXE) ./t11-cx
	@coredir=./bin; exe=$(EXE); goflags=; $(CORE_BUILD)

release-archives: ## Build the archives of release VERSION (a v tag) for RELEASE_PLATFORMS with cgo off, and SHA256SUMS, into RELEASE_DIR
	@case "$(VERSION)" in v[0-9]*) ;; *) echo "give the tag, as in make release-archives VERSION=v0.9.0"; exit 1 ;; esac
	rm -rf $(RELEASE_DIR)
	@set -e; for platform in $(RELEASE_PLATFORMS); do \
		export GOOS=$${platform%/*} GOARCH=$${platform#*/} CGO_ENABLED=0; \
		exe=; if test $$GOOS = windows; then exe=.exe; fi; \
		name=cx-$(VERSION)-$$GOOS-$$GOARCH; coredir=$(RELEASE_DIR)/$$name/bin; goflags="-trimpath -ldflags=-s"; \
		mkdir -p $$coredir; \
		echo go build -trimpath -ldflags=\"-s -X main.VERSION=$(VERSION:v%=%)\" -tags=\"$(TAGS)\" -o $$coredir/cx$$exe ./t11-cx; \
		go build -trimpath -ldflags="-s -X main.VERSION=$(VERSION:v%=%)" -tags="$(TAGS)" -o $$coredir/cx$$exe ./t11-cx; \
		$(CORE_BUILD); \
		cp LICENSE README.md CHANGELOG.md $(RELEASE_DIR)/$$name/; \
		if test $$GOOS = windows; then (cd $(RELEASE_DIR) && zip -qr $$name.zip $$name); \
		else tar -C $(RELEASE_DIR) -czf $(RELEASE_DIR)/$$name.tar.gz $$name; fi; \
		rm -r $(RELEASE_DIR)/$$name; \
	done
	cd $(RELEASE_DIR) && sha256sum cx-* > SHA256SUMS && cat SHA256SUMS

release-notes: ## Print the release notes of VERSION, its CHANGELOG.md section or else the Unreleased section
	@echo "The GUI build of cx (the cxfx tag with sys/gl, sys/glfw and sys/gltext) is not included, since it needs cgo. Build it from source with make build-gui. \`cx build --target=llvm\` needs clang on your machine, see docs/LlvmTarget.md."
	@echo
	@awk -v version="### $(VERSION)" ' \
		/^### / { inversion = ($$0 == version || index($$0, version " ") == 1); inunreleased = ($$0 == "### Unreleased"); next } \
		inversion { found = found $$0 "\n" } \
		inunreleased { unreleased = unreleased $$0 "\n" } \
		END { printf "%s", (found != "" ? found : unreleased) }' CHANGELOG.md

build-gui: ## Build cx with the cxfx GL/GLFW/OpenAL package (needs cgo and X11/GL headers)
	go build -tags="$(TAGS) $(GUI_TAGS)" -o ./bin/$(CXEXE) ./t11-cx

build-debug: ## Build cx without optimizations
	go build -gcflags="all=-N -l" -tags="$(TAGS)" -o ./bin/$(CXEXE) ./t11-cx

build-wasm: ## Compile check cx for js/wasm with the default tags, as the playground builds it
	GOOS=js GOARCH=wasm go build -tags="$(TAGS)" -o /dev/null ./t11-cx

test-wasm: ## Build cx for js/wasm and run the hello world example under Node
	GOOS=js GOARCH=wasm go build -tags="$(TAGS)" -o ./bin/cx.wasm ./t11-cx
	node "$$(go env GOROOT)/lib/wasm/wasm_exec_node.js" ./bin/cx.wasm tests/examples/hello-world.cx | grep -qx "Hello World!"

# The TinyGo targets are opt in, the Go builds stay the default, see
# docs/Microcontrollers.md.
TINYGO ?= tinygo

build-tinygo: ## Build cx and cx-core with TinyGo, ./bin/cx-tinygo and ./bin/cx-core-tinygo (needs tinygo)
	$(TINYGO) build -tags="$(TAGS)" -o ./bin/cx-tinygo$(EXE) ./t11-cx
	$(TINYGO) build -tags="$(TAGS) cxcore" -o ./bin/cx-core-tinygo$(EXE) ./t6-cx-core-run

build-wasm-tinygo: ## Build cx for js/wasm with TinyGo, ./bin/cx-tinygo.wasm, which runs with the wasm_exec.js of TinyGo
	$(TINYGO) build -target=wasm -tags="$(TAGS)" -o ./bin/cx-tinygo.wasm ./t11-cx

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

lint: ## golangci-lint with .golangci.yml, the version the lint job of CI pins
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION) run -c .golangci.yml ./...

vulncheck: ## govulncheck of the module and the Go standard library
	go run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION) ./...

llvm-clang: ## Print the clang that cx build --target=llvm and the native form of the execution records use, see docs/LlvmTarget.md
	@command -v "$${CX_CLANG:-clang}" >/dev/null && "$${CX_CLANG:-clang}" --version | head -n 1 || echo "no clang, the native form of the execution records is not run"

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

check-names: ## Check the Go names against the Naming section of AGENTS.md, see names-allowlist.txt
	go run ./t0-cx-tools-names -check names-allowlist.txt .

names-csv: ## Write one CSV row per declared Go and CX name to NAMES_CSV, for skycoin-cx and the repositories in REPOS
	mkdir -p $(dir $(NAMES_CSV))
	go run ./t0-cx-tools-names -out $(NAMES_CSV) . $(REPOS)

generate-opcodes: ## Regenerate the opcode and library signature tables and switches from t1-constants/opcodes_definitions.go and libraries_definitions.go, decisions D21 and D96
	go run ./t2-cx-core-opcodegen -root .

check-opcodes: ## Check that the opcode and library tables and switches are up to date with t1-constants/opcodes_definitions.go and libraries_definitions.go
	go run ./t2-cx-core-opcodegen -root . -check

generate-objmodel: ## Regenerate the Gid, manager and manager test files of t3-ast and the site counts of docs/ObjectModel.md
	go run ./t0-cx-core-objmodelgen -dir t3-ast

check: check-tiers llvm-clang check-inits check-ifaces check-names check-json check-recovers check-objmodel check-opcodes check-gcreach check-strhot vet test-go test test-ptr64 ## tier, init, interface, naming, JSON, recover, object model, opcode table, GC reach and STRHOT checks, vet, Go tests and CX program tests (default and ptr64)

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
