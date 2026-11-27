//go:build tinygo

package profiling

import (
	"os"
	"runtime/pprof"
)

// profilingCPUStartAtRate ignores rate, as TinyGo has no CPU profiler.
func profilingCPUStartAtRate(f *os.File, rate int) error {
	return pprof.StartCPUProfile(f)
}
