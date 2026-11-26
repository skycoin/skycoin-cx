//go:build cximagemarkers

package main

import (
	"os"
	"strconv"
)

// coreImageMarkerBegin and coreImageMarkerExit frame the output of the run
// for a script that reads it from an emulator, which may print boot text
// first and may not pass the exit code on, see docs/Microcontrollers.md.
func coreImageMarkerBegin() {
	coreImageWrite(os.Stdout, "\n"+coreImageMarkerBeginLine+"\n")
}

func coreImageMarkerExit(code int32) {
	coreImageWrite(os.Stdout, "\n"+coreImageMarkerExitPrefix+strconv.Itoa(int(code))+"\n")
}

const (
	coreImageMarkerBeginLine  = "cx-core-image-begin"
	coreImageMarkerExitPrefix = "cx-core-image-exit "
)
