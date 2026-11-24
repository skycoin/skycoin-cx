//go:build unix

package netbatch

import (
	"errors"
	"syscall"
)

// netErrorOfSystem returns the NetError of an errno, NetErrorNone for others.
// VIOLATION STRHOT-346: B, reads the error Go returned, outside the update, #508.
func netErrorOfSystem(err error) NetError {
	switch {
	case errors.Is(err, syscall.ECONNREFUSED):
		return NetErrorRefused
	case errors.Is(err, syscall.ECONNRESET), errors.Is(err, syscall.EPIPE):
		return NetErrorReset
	case errors.Is(err, syscall.EADDRINUSE):
		return NetErrorAddressInUse
	}
	return NetErrorNone
}
