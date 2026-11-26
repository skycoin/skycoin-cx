package main

import (
	"fmt"
	"os"

	"github.com/skycoin/skycoin-cx/t0-types"
	"github.com/skycoin/skycoin-cx/t1-constants"
	"github.com/skycoin/skycoin-cx/t2-cx-runtime-ir"
	"github.com/skycoin/skycoin-cx/t4-actions"
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
	prgrm := actions.AST
	core.CoreLibrariesEnabledResolve(prgrm)
	for library, enabled := range prgrm.ProgramLibrariesEnabled {
		prgrm.ProgramLibrariesEnabled[library] = enabled && prgrm.ProgramLibrariesUsed[library]
		if !prgrm.ProgramLibrariesUsed[library] {
			prgrm.ProgramOperationsEnabled.OpcodeSetLibraryRemove(library)
		}
	}
	prgrm.CxIrState.CxIrStateManagerGrowBoundary()
	prgrm.CxIrState.StringPool.StringPoolFreeze()
	if code := core.CoreOperationRefused(prgrm); code != 0 {
		fmt.Fprintf(os.Stderr, "cx build: %s\n", core.CoreOperationRefusalLine(prgrm, code))
		return types.CX_RUNTIME_LIBRARY_DISABLED
	}
	image, _, outside := core.ExecutorLower(prgrm)
	if outside != "" {
		fmt.Fprintf(os.Stderr, "cx build: cannot run on vm: %s\n", outside)
		return types.CX_RUNTIME_NOT_IMPLEMENTED
	}
	for _, library := range [...]int{constants.LibraryNumberFile, constants.LibraryNumberNet} {
		if image.ImageOperationsEnabled.OpcodeSetLibraryAny(library) {
			return buildFail(fmt.Sprintf("the program uses the %s library, which cx-core-image does not link", constants.LibraryTable[library-1].LibraryName))
		}
	}
	if err := os.WriteFile(options.buildOptionsOutput, ir.ImageFileEncode(&image), 0o644); err != nil {
		return buildFail(err.Error())
	}
	return types.CX_SUCCESS
}
