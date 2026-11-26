//go:build !tinygo

package sandbox

import "os"

// Root is the directory every program path resolves in. With
// Go it is an os.Root, which keeps every path and symlink inside it.
type Root = os.Root

func rootDirectoryOpen(dir string) (*Root, error) {
	return os.OpenRoot(dir)
}

// rootFileOpen opens name in root with flag and perm, as os.OpenFile does.
// VIOLATION STRHOT-354: B, hands a path or text to Go or the OS, #412.
func rootFileOpen(root *Root, name string, flag int, perm os.FileMode) (*os.File, error) {
	return root.OpenFile(name, flag, perm)
}
