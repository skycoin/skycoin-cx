//go:build !tinygo

package profiling

import "syscall"

// profilingDescriptorReplace makes descriptor target a copy of source. Linux
// on arm64 and riscv64 has no dup2, so it is dup3.
func profilingDescriptorReplace(source int, target int) error {
	return syscall.Dup3(source, target, 0)
}
