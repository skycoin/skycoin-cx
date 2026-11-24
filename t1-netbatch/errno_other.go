//go:build !unix && !windows

package netbatch

// netErrorOfSystem has no error numbers to read on js/wasm and wasip1.
func netErrorOfSystem(err error) NetError {
	return NetErrorNone
}
