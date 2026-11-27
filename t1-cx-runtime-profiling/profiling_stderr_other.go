//go:build (!unix || solaris) && !tinygo

package profiling

// profilingStderrMute does nothing where file descriptor 2 cannot be swapped.
func profilingStderrMute() (restore func()) {
	return func() {}
}
