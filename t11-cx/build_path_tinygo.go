//go:build tinygo

package main

import "path/filepath"

// buildPathSame reports whether first and second name the same file. TinyGo
// has no os.SameFile, so it compares the paths with symlinks resolved, which
// misses two hard links to one file.
func buildPathSame(first, second string) bool {
	firstPath, firstErr := filepath.EvalSymlinks(first)
	secondPath, secondErr := filepath.EvalSymlinks(second)
	if firstErr != nil || secondErr != nil {
		return false
	}
	firstPath, firstErr = filepath.Abs(firstPath)
	secondPath, secondErr = filepath.Abs(secondPath)
	return firstErr == nil && secondErr == nil && firstPath == secondPath
}
