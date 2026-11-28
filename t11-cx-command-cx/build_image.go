package main

import (
	"fmt"
	"os"

	"github.com/skycoin/skycoin-cx/t0-cx-core-valuetypes"
	"github.com/skycoin/skycoin-cx/t1-cx-core-opcodetable"
	"github.com/skycoin/skycoin-cx/t2-cx-runtime-ir"
	"github.com/skycoin/skycoin-cx/t4-cx-compiler-astbuild"
	core "github.com/skycoin/skycoin-cx/t5-cx-runtime-core"
)

// buildTargetImage writes the image of the program for cx-core-image, the
// vm runner that builds with TinyGo for microcontrollers, see
// docs/Microcontrollers.md.
const buildTargetImage = "image"

// buildImageExtension ends the file name of an image.
const buildImageExtension = ".cximage"

// buildImage writes the image vm runs to the output. A program outside the
// vm subset, or one that uses a library cx-core-image leaves out, is
// refused. -stack-size sets the stack of the image, which a microcontroller
// needs far below the 1 MB default.
func buildImage(options buildOptions) int {
	image, problem, code := core.ExecutorImageBuild(astbuild.AST)
	if problem != "" {
		fmt.Fprintf(os.Stderr, "cx build: %s\n", problem)
		return int(code)
	}
	for _, library := range [...]int{opcodetable.LibraryNumberFile, opcodetable.LibraryNumberNet, opcodetable.LibraryNumberCompiler} {
		if image.ImageOperationsEnabled.OpcodeSetLibraryAny(library) {
			return buildFail(fmt.Sprintf("the program uses the %s library, which cx-core-image does not link", opcodetable.LibraryTable[library-1].LibraryName))
		}
	}
	fmt.Fprint(os.Stderr, buildCallGraphReport(&image, options.buildOptionsDepthQuiet))
	if err := os.WriteFile(options.buildOptionsOutput, ir.ImageFileEncode(&image), 0o644); err != nil {
		return buildFail(err.Error())
	}
	return valuetypes.CXSuccess
}
