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
	if rate == 100 {
		return pprof.StartCPUProfile(f)
	}
	runtime.SetCPUProfileRate(rate)
	restore, err := profilingStderrMute()
	if err != nil {
		return pprof.StartCPUProfile(f)
	}
	if err := pprof.StartCPUProfile(f); err != nil {
		if restoreErr := restore(); restoreErr != nil {
			return restoreErr
		}
		return err
	}
	if err := restore(); err != nil {
		pprof.StopCPUProfile()
		return err
	}
	return nil
}
