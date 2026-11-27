//go:build !tinygo

package profiling

import (
	"os"
	"runtime"
	"runtime/pprof"
)

// profilingCPUStartAtRate starts a CPU profile on f at rate samples per second.
// pprof.StartCPUProfile sets 100 Hz again and the runtime warns, so stderr is muted for that call.
func profilingCPUStartAtRate(f *os.File, rate int) error {
	if rate != 100 {
		runtime.SetCPUProfileRate(rate)
		defer profilingStderrMute()()
	}
	return pprof.StartCPUProfile(f)
}
