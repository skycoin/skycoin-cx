//go:build windows

package netbatch

import (
	"errors"
	"syscall"
)

// The Winsock error numbers, see
// https://learn.microsoft.com/en-us/windows/win32/winsock/windows-sockets-error-codes-2
const (
	netWinsockAddressInUse = syscall.Errno(10048)
	netWinsockReset        = syscall.Errno(10054)
	netWinsockRefused      = syscall.Errno(10061)
)

// netErrorOfSystem returns the NetError of a Winsock error, NetErrorNone for
// others.
func netErrorOfSystem(err error) NetError {
	switch {
	case errors.Is(err, netWinsockRefused):
		return NetErrorRefused
	case errors.Is(err, netWinsockReset):
		return NetErrorReset
	case errors.Is(err, netWinsockAddressInUse):
		return NetErrorAddressInUse
	}
	return NetErrorNone
}
