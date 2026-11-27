//go:build unix && !solaris && !tinygo

package profiling

import (
	"os"
	"syscall"
)

// profilingStderrMute points file descriptor 2 at the null device and returns
// a function that restores it. It does nothing if either step fails.
// VIOLATION STRHOT-355: B, hands a path or text to Go or the OS, #412.
func profilingStderrMute() (restore func()) {
	saved, err := syscall.Dup(2)
	if err != nil {
		return func() {}
	}
	null, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		syscall.Close(saved)
		return func() {}
	}
	defer null.Close()
	if err := profilingDescriptorReplace(int(null.Fd()), 2); err != nil {
		syscall.Close(saved)
		return func() {}
	}
	return func() {
		profilingDescriptorReplace(saved, 2)
		syscall.Close(saved)
	}
}
