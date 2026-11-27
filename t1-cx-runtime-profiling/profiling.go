package profiling

import (
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"time"

	filelog "github.com/skycoin/skycoin-cx/t0-cx-runtime-filelog"
)

// VIOLATION: state outside State. Owner: t1-cx-runtime-profiling; moves to ConfigMng in phase 2.
var DebugProfile bool

// VIOLATION: state outside State. Owner: t1-cx-runtime-profiling; moves to ConfigMng in phase 2.
var DebugProfileRate int

// VIOLATION: state outside State. Owner: t1-cx-runtime-profiling; moves to ProfileMng in phase 1.
var profiles = make(map[string]int64)

// VIOLATION: state outside State. Owner: t1-cx-runtime-profiling; moves to ProfileMng in phase 1.
// pprof runs one CPU profile at a time, this stops it.
var stopCPU func() error

func ProfileStart(name string) {
	if DebugProfile {
		profiles[name] = time.Now().UnixNano()
	}
}

func ProfileStop(name string) {
	if DebugProfile {
		t := time.Now().UnixNano()
		deltaTime := t - profiles[name]
		fmt.Printf("%s : %dms\n", name, deltaTime/(int64(time.Millisecond)))
	}
}

// CPUProfileStart writes a CPU profile to <argv0>_<name>_cpu.pprof at
// profRate samples per second. A rate of 0 does nothing. The returned
// function stops the profile and closes the file.
func CPUProfileStart(name string, profRate int) (stop func() error, err error) {
	return CPUProfileStartFile(fmt.Sprintf("%s_%s_cpu.pprof", os.Args[0], name), profRate)
}

// CPUProfileStartFile is CPUProfileStart writing to path.
func CPUProfileStartFile(path string, profRate int) (stop func() error, err error) {
	if profRate == 0 {
		return func() error { return nil }, nil
	}

	f, err := filelog.Create(path)
	if err != nil {
		return nil, fmt.Errorf("create CPU profile: %w", err)
	}
	return CPUProfileStartOnFile(f, profRate)
}

// CPUProfileStartOnFile is CPUProfileStart writing to f at a nonzero profRate.
// It closes f on stop or failure.
// VIOLATION STRHOT-016: B, hands a path or text to Go or the OS, #412.
// VIOLATION VMALLOC-018: a system library call allocates in Go and the OS, #407.
func CPUProfileStartOnFile(f *os.File, profRate int) (stop func() error, err error) {
	if err := profilingCPUStartAtRate(f, profRate); err != nil {
		f.Close()
		return nil, fmt.Errorf("start CPU profile: %w", err)
	}

	stopped := false
	stop = func() error {
		if stopped {
			return nil
		}
		stopped = true
		stopCPU = nil
		pprof.StopCPUProfile()
		return f.Close()
	}
	stopCPU = stop
	return stop, nil
}

// DumpMemProfile writes a heap profile to <argv0>_<name>_mem.pprof.
func DumpMemProfile(name string) (err error) {
	f, err := filelog.Create(fmt.Sprintf("%s_%s_mem.pprof", os.Args[0], name))
	if err != nil {
		return fmt.Errorf("create MEM profile: %w", err)
	}

	defer func() {
		if closeErr := f.Close(); err == nil {
			err = closeErr
		}
	}()

	runtime.GC()
	if err = pprof.WriteHeapProfile(f); err != nil {
		return fmt.Errorf("write MEM profile: %w", err)
	}

	return nil
}

// CleanupAndExit stops a running CPU profile and exits with exitCode.
func CleanupAndExit(exitCode int) {
	if stopCPU != nil {
		if err := stopCPU(); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
	os.Exit(exitCode)
}
