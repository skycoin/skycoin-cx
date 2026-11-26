//go:build tinygo

package profiling

// profilingCPURateSet does nothing, as TinyGo has no CPU profiler.
func profilingCPURateSet(rate int) {}
