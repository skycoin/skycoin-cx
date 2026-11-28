package syspackages

import (
	"fmt"
	"strings"
)

// DefaultPackages are the built-in packages, implemented in the compiler and
// runtime. Each is imported as SysPrefix + name, see docs/Conventions.md rule 14.
var DefaultPackages = []string{
	// temporary solution until we can implement these packages in pure CX I guess
	//"al", "gl", "glfw", "time", "http", "os", "explorer", "aff", "gltext", "cx", "json", "regexp", "cipher", "tcp",
	"al", "gl", "glfw", "time", "os", "gltext", "cx", "cipher", "net", "reflect", "regexp", "string",
}

// SysPrefix is the import path prefix of the built-in packages.
const SysPrefix = "sys/"

// PackageDefault reports whether ident is the name of a built-in package.
func PackageDefault(ident string) bool {
	for _, core := range DefaultPackages {
		if core == ident {
			return true
		}
	}
	return false
}

// ImportBuiltin reports whether path refers to a built-in package, correctly
// or not, so the loaders must not look for it in the workspace.
func ImportBuiltin(path string) bool {
	return strings.HasPrefix(path, SysPrefix) || PackageDefault(path)
}

// ImportCheck returns an error when path names a built-in package that does
// not exist or omits SysPrefix.
func ImportCheck(path string) error {
	if name, found := strings.CutPrefix(path, SysPrefix); found {
		if !PackageDefault(name) {
			return fmt.Errorf("unknown built-in package %q", path)
		}
		return nil
	}
	if PackageDefault(path) {
		return fmt.Errorf("built-in package %s is imported as \"%s%s\"", path, SysPrefix, path)
	}
	return nil
}
