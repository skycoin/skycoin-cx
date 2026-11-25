//go:build cxcore && !cxcorelibrarynet

package core

// executorNetRunStart does nothing in a core built without
// cxcorelibrarynet, which refuses a program that uses the net library.
func executorNetRunStart() error {
	return nil
}
