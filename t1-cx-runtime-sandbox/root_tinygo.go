//go:build tinygo

package sandbox

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// Root is the directory every program path resolves in.
// TinyGo has no os.Root, so this one checks the path in Go before the call:
// a path must be local and no part of it may be a symlink. A symlink made
// between the check and the call is not caught, which os.Root catches.
type Root struct {
	rootDirectory string
}

// rootEscape is the error os.Root gives for a path that
// leaves the root, so fileErrorFromFileSystemError maps it the same way.
var rootEscape = errors.New(fileSystemPathEscapeMessage)

func rootDirectoryOpen(dir string) (*Root, error) {
	stat, err := os.Stat(dir)
	if err != nil {
		return nil, err
	}
	if !stat.IsDir() {
		return nil, &fs.PathError{Op: "openat", Path: dir, Err: syscall.ENOTDIR}
	}
	return &Root{rootDirectory: dir}, nil
}

// Name returns the directory as it was opened, as os.Root.Name does.
func (root *Root) Name() string {
	return root.rootDirectory
}

// Close does nothing, the root holds no descriptor.
func (root *Root) Close() error {
	return nil
}

// pathResolve returns name joined to the root, refusing a name that is not
// local or that passes through a symlink. Remove passes lastSymlinkAllowed,
// as it removes a symlink itself and never what it points to.
func (root *Root) pathResolve(operation string, name string, lastSymlinkAllowed bool) (string, error) {
	if !filepath.IsLocal(name) {
		return "", &fs.PathError{Op: operation, Path: name, Err: rootEscape}
	}
	resolved := root.rootDirectory
	parts := strings.Split(filepath.Clean(name), string(filepath.Separator))
	for i, part := range parts {
		resolved = filepath.Join(resolved, part)
		stat, err := os.Lstat(resolved)
		if err != nil {
			break
		}
		if stat.Mode()&fs.ModeSymlink != 0 && !(lastSymlinkAllowed && i == len(parts)-1) {
			return "", &fs.PathError{Op: operation, Path: name, Err: rootEscape}
		}
	}
	return filepath.Join(root.rootDirectory, name), nil
}

// Open opens name in the root for reading.
func (root *Root) Open(name string) (*os.File, error) {
	return rootFileOpen(root, name, os.O_RDONLY, 0)
}

// Create creates or truncates name in the root.
func (root *Root) Create(name string) (*os.File, error) {
	return rootFileOpen(root, name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o666)
}

// rootFileOpen opens name in root with flag and perm, as os.OpenFile does.
func rootFileOpen(root *Root, name string, flag int, perm os.FileMode) (*os.File, error) {
	path, err := root.pathResolve("openat", name, false)
	if err != nil {
		return nil, err
	}
	return os.OpenFile(path, flag, perm)
}

// Remove removes name in the root, a file or an empty directory.
func (root *Root) Remove(name string) error {
	path, err := root.pathResolve("removeat", name, true)
	if err != nil {
		return err
	}
	return os.Remove(path)
}
