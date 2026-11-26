//go:build tinygo && cortexm

package main

import (
	"runtime/volatile"
	"unsafe"
)

// The helpers of the ARM run time ABI for unaligned 4 and 8 byte accesses
// (IHI 0043, section 4.4.3). LLVM calls them on a core without unaligned
// access, such as the Cortex-M0+ of the RP2040, and the compiler-rt of
// TinyGo leaves them out. Volatile byte accesses keep LLVM from merging
// them back into one word access, which would call the helper again.

//export __aeabi_uread4
func aeabiUnalignedRead4(address unsafe.Pointer) int32 {
	var value uint32
	for i := uintptr(0); i < 4; i++ {
		value |= uint32(volatile.LoadUint8((*uint8)(unsafe.Add(address, i)))) << (8 * i)
	}
	return int32(value)
}

//export __aeabi_uwrite4
func aeabiUnalignedWrite4(value int32, address unsafe.Pointer) int32 {
	for i := uintptr(0); i < 4; i++ {
		volatile.StoreUint8((*uint8)(unsafe.Add(address, i)), uint8(uint32(value)>>(8*i)))
	}
	return value
}

//export __aeabi_uread8
func aeabiUnalignedRead8(address unsafe.Pointer) int64 {
	return int64(uint64(uint32(aeabiUnalignedRead4(address))) | uint64(uint32(aeabiUnalignedRead4(unsafe.Add(address, 4))))<<32)
}

//export __aeabi_uwrite8
func aeabiUnalignedWrite8(value int64, address unsafe.Pointer) int64 {
	aeabiUnalignedWrite4(int32(uint32(value)), address)
	aeabiUnalignedWrite4(int32(uint32(uint64(value)>>32)), unsafe.Add(address, 4))
	return value
}
