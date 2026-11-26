//go:build !tinygo

package main

import "os"

// buildPathSame reports whether first and second name the same file.
func buildPathSame(first, second string) bool {
	firstStat, firstErr := os.Stat(first)
	secondStat, secondErr := os.Stat(second)
	return firstErr == nil && secondErr == nil && os.SameFile(firstStat, secondStat)
}
