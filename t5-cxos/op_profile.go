//go:build !nocxos

package cxos

import (
	"fmt"
	"os"

	"github.com/skycoin/skycoin-cx/t1-profiling" // thin: only StartCPUProfileFile; no depth gain
	"github.com/skycoin/skycoin-cx/t3-ast"
)

// VIOLATION: state outside State. Owner: t5-cxos; moves to OsFileMng in phase 1.
var openProfiles = make(map[string]func() error)

// opStartProfile writes a CPU profile to the path it is given.
func opStartProfile(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	profilePath := inputs[0].Get_str(prgrm)
	stop, err := profiling.StartCPUProfileFile(profilePath, int(inputs[1].Get_i32(prgrm)))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	openProfiles[profilePath] = stop
}

func opStopProfile(prgrm *ast.CXProgram, inputs []ast.CXValue, outputs []ast.CXValue) {
	profilePath := inputs[0].Get_str(prgrm)
	stop, ok := openProfiles[profilePath]
	if !ok {
		return
	}
	delete(openProfiles, profilePath)
	if err := stop(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}
