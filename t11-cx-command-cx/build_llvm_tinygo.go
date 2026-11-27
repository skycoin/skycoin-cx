package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/skycoin/skycoin-cx/t0-cx-core-valuetypes"
	llvm "github.com/skycoin/skycoin-cx/t3-cx-compiler-llvm"
)

// The linkers of --llvm-linker. clang compiles the module and the C runtime
// with the clang on the PATH. tinygo compiles them with the clang inside
// TinyGo and links them into a TinyGo program, which brings the C library,
// compiler-rt, startup code and linker script of a TinyGo target, see
// docs/Microcontrollers.md.
const (
	buildLlvmLinkerClang  = "clang"
	buildLlvmLinkerTinygo = "tinygo"
)

// buildLlvmTinygoEnv names the tinygo cx build runs, tinygo when unset.
const buildLlvmTinygoEnv = "CX_TINYGO"

// buildLlvmTinygoFlagsEnv holds more flags for tinygo build, such as
// -serial=uart for an ESP32-C3 in QEMU.
const buildLlvmTinygoFlagsEnv = "CX_TINYGO_FLAGS"

// buildLlvmTinygoEntryName is the name of the emitted entry function in a
// TinyGo program, whose main calls it.
const buildLlvmTinygoEntryName = "cx_program_main"

// buildLlvmTinygo returns the tinygo cx build runs, or "" when there is none.
func buildLlvmTinygo() string {
	tinygo := os.Getenv(buildLlvmTinygoEnv)
	if tinygo == "" {
		tinygo = "tinygo"
	}
	path, err := exec.LookPath(tinygo)
	if err != nil {
		return ""
	}
	return path
}

// buildLlvmLinkerChoose returns the linker of a build, or a problem. A
// TinyGo target needs tinygo. Without --llvm-linker it is clang, or tinygo
// when there is no clang and there is a tinygo.
func buildLlvmLinkerChoose(options buildOptions) (string, string) {
	linker := options.buildOptionsLlvmLinker
	switch {
	case linker != "" && linker != buildLlvmLinkerClang && linker != buildLlvmLinkerTinygo:
		return "", fmt.Sprintf("unknown --llvm-linker %q, want %s or %s", linker, buildLlvmLinkerClang, buildLlvmLinkerTinygo)
	case options.buildOptionsLlvmTinygoTarget != "" && linker == buildLlvmLinkerClang:
		return "", "--llvm-tinygo-target needs --llvm-linker=tinygo"
	case options.buildOptionsLlvmTinygoTarget != "":
		return buildLlvmLinkerTinygo, ""
	case linker != "":
		return linker, ""
	case buildLlvmClang() == "" && buildLlvmTinygo() != "":
		return buildLlvmLinkerTinygo, ""
	}
	return buildLlvmLinkerClang, ""
}

// buildLlvmTinygoTarget is the code generation of a TinyGo target, from
// tinygo info -json.
type buildLlvmTinygoTarget struct {
	buildLlvmTinygoTargetTriple   string
	buildLlvmTinygoTargetCpu      string
	buildLlvmTinygoTargetFeatures string
}

// buildLlvmTinygoTargetRead runs tinygo info -json for target, the host when
// it is empty. The three fields are plain strings, so they are cut from the
// text and encoding/json stays out of cx (skycoin-cx#509).
func buildLlvmTinygoTargetRead(tinygo string, target string) (buildLlvmTinygoTarget, error) {
	args := []string{"info", "-json"}
	if target != "" {
		args = append(args, "-target="+target)
	}
	var stderr bytes.Buffer
	command := exec.Command(tinygo, args...)
	command.Stderr = &stderr
	out, err := command.Output()
	if err != nil {
		return buildLlvmTinygoTarget{}, fmt.Errorf("%s info %s: %v: %s", tinygo, target, err, strings.TrimSpace(stderr.String()))
	}
	field := func(key string) string {
		marker := []byte(`"` + key + `": "`)
		start := bytes.Index(out, marker)
		if start < 0 {
			return ""
		}
		value := out[start+len(marker):]
		if end := bytes.IndexByte(value, '"'); end >= 0 {
			return string(value[:end])
		}
		return ""
	}
	codeGeneration := buildLlvmTinygoTarget{
		buildLlvmTinygoTargetTriple:   field("llvm-target"),
		buildLlvmTinygoTargetCpu:      field("cpu"),
		buildLlvmTinygoTargetFeatures: field("features"),
	}
	if codeGeneration.buildLlvmTinygoTargetTriple == "" {
		return codeGeneration, fmt.Errorf("%s info -json %s gives no llvm-target", tinygo, target)
	}
	if codeGeneration.buildLlvmTinygoTargetTriple == buildLlvmTinygoTripleWasi {
		version, err := exec.Command(tinygo, "version").Output()
		if err != nil {
			return codeGeneration, fmt.Errorf("%s version: %v", tinygo, err)
		}
		codeGeneration.buildLlvmTinygoTargetTriple = buildLlvmTinygoTripleClang(codeGeneration.buildLlvmTinygoTargetTriple, buildLlvmTinygoLlvmMajorParse(version))
	}
	return codeGeneration, nil
}

// The wasm triples of TinyGo. LLVM 22 deprecated the first for the second.
const (
	buildLlvmTinygoTripleWasi   = "wasm32-unknown-wasi"
	buildLlvmTinygoTripleWasip1 = "wasm32-unknown-wasip1"
)

// buildLlvmTinygoTripleClang returns triple as TinyGo passes it to clang for
// LLVM llvmMajor, as compileopts.LLVMTriple of tinygo-org/tinygo#6309 does.
func buildLlvmTinygoTripleClang(triple string, llvmMajor int) string {
	if triple == buildLlvmTinygoTripleWasi && llvmMajor >= 22 {
		return buildLlvmTinygoTripleWasip1
	}
	return triple
}

// buildLlvmTinygoLlvmMajorParse returns the LLVM major version in the output
// of tinygo version, or 0 when it has none.
func buildLlvmTinygoLlvmMajorParse(version []byte) int {
	marker := []byte("LLVM version ")
	start := bytes.Index(version, marker)
	if start < 0 {
		return 0
	}
	major := 0
	for _, digit := range version[start+len(marker):] {
		if digit < '0' || digit > '9' {
			break
		}
		major = major*10 + int(digit-'0')
	}
	return major
}

// buildLlvmTinygoMain is the Go side of the TinyGo program: main runs the
// emitted entry, and the runtime ends the process through os.Exit.
const buildLlvmTinygoMain = `package main

/*
#cgo CFLAGS: -DCX_RT_TINYGO -O2 -w
#cgo LDFLAGS: %s
#include <stdint.h>
int32_t %s(void);
*/
import "C"

import (
	"os"
	"strconv"
	"unsafe"
)

//export cx_rt_tinygo_exit
func cx_rt_tinygo_exit(code int32) {
	os.Exit(int(code))
}

//export cx_rt_tinygo_float_append
func cx_rt_tinygo_float_append(v float64, format int32, precision int32, bits int32, out *byte, room int32) int32 {
	var text [400]byte
	return int32(copy(unsafe.Slice(out, room), strconv.AppendFloat(text[:0], v, byte(format), int(precision), int(bits))))
}

func main() {
	os.Exit(int(C.%s()))
}
`

// buildLlvmTinygoNative compiles the module at modulePath with the clang of
// TinyGo for target, the host when it is empty, and builds it with the C
// runtime into a TinyGo program at output. The extension of output picks
// the file format of an embedded target, as tinygo build does.
func buildLlvmTinygoNative(modulePath string, output string, target string, debug bool) int {
	tinygo := buildLlvmTinygo()
	if tinygo == "" {
		return buildFail(fmt.Sprintf("wrote %s but found no tinygo to build it into %s, install TinyGo or set %s", modulePath, output, buildLlvmTinygoEnv))
	}
	codeGeneration, err := buildLlvmTinygoTargetRead(tinygo, target)
	if err != nil {
		return buildFail(err.Error())
	}
	directory, err := os.MkdirTemp("", "cx-tinygo-")
	if err != nil {
		return buildFail(err.Error())
	}
	defer func() {
		if err := os.RemoveAll(directory); err != nil {
			fmt.Fprintf(os.Stderr, "cx build: cannot remove %s: %v\n", directory, err)
		}
	}()
	object := filepath.Join(directory, "program.o")
	args := []string{"clang", "-c", "-O2", "-w", "--target=" + codeGeneration.buildLlvmTinygoTargetTriple}
	if codeGeneration.buildLlvmTinygoTargetCpu != "" && codeGeneration.buildLlvmTinygoTargetCpu != "generic" {
		args = append(args, "-Xclang", "-target-cpu", "-Xclang", codeGeneration.buildLlvmTinygoTargetCpu)
	}
	for _, feature := range strings.Split(codeGeneration.buildLlvmTinygoTargetFeatures, ",") {
		if feature != "" {
			args = append(args, "-Xclang", "-target-feature", "-Xclang", feature)
		}
	}
	if debug {
		args = append(args, "-g")
	}
	args = append(args, "-o", object, modulePath)
	if err := buildLlvmTinygoRun(tinygo, "", args); err != nil {
		return buildFail(err.Error())
	}
	files := map[string][]byte{
		"go.mod":             []byte("module cxprogram\n\ngo 1.22\n"),
		"main.go":            []byte(fmt.Sprintf(buildLlvmTinygoMain, object, buildLlvmTinygoEntryName, buildLlvmTinygoEntryName)),
		llvm.RuntimeFileName: llvm.RuntimeC,
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(directory, name), content, 0o644); err != nil {
			return buildFail(err.Error())
		}
	}
	output, err = filepath.Abs(output)
	if err != nil {
		return buildFail(err.Error())
	}
	args = []string{"build", "-o", output}
	if target != "" {
		args = append(args, "-target="+target)
	}
	// TinyGo keeps DWARF by default, which made fib for wasip1 1.34 MB instead
	// of 74 KB. CX_TINYGO_FLAGS comes after it, so -no-debug=false still wins.
	if !debug {
		args = append(args, "-no-debug")
	}
	args = append(args, strings.Fields(os.Getenv(buildLlvmTinygoFlagsEnv))...)
	if err := buildLlvmTinygoRun(tinygo, directory, append(args, ".")); err != nil {
		return buildFail(err.Error())
	}
	return valuetypes.CXSuccess
}

// buildLlvmTinygoRun runs tinygo with args in directory, its output on
// stderr. GOFLAGS and GOWORK of the caller would apply to the generated
// module, so they are cleared.
func buildLlvmTinygoRun(tinygo string, directory string, args []string) error {
	command := exec.Command(tinygo, args...)
	command.Dir = directory
	command.Env = append(os.Environ(), "GOFLAGS=", "GOWORK=off")
	command.Stdout, command.Stderr = os.Stderr, os.Stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("%s %s: %v", tinygo, args[0], err)
	}
	return nil
}
