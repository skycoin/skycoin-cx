//go:build unix && !linux && !solaris && !tinygo

package profiling

import "syscall"

// profilingDescriptorReplace makes descriptor target a copy of source.
func profilingDescriptorReplace(source int, target int) error {
	return syscall.Dup2(source, target)
}
