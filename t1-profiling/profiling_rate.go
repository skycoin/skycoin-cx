//go:build !tinygo

package profiling

import "runtime"

func profilingCPURateSet(rate int) {
	runtime.SetCPUProfileRate(rate)
}
