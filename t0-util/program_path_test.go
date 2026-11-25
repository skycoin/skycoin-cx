package util

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProgramFileSystemPathFromBytesRejects(t *testing.T) {
	cases := []struct {
		path string
		code FileErrorCode
	}{
		{"", FileErrorCodePathInvalid},
		{"a\x00b", FileErrorCodePathInvalid},
		{strings.Repeat("a", ProgramFileSystemPathLengthMaximum+1), FileErrorCodePathInvalid},
		{`C:\x`, FileErrorCodePathInvalid},
		{`\\server\x`, FileErrorCodePathInvalid},
		{`a\b`, FileErrorCodePathInvalid},
		{"/etc/passwd", FileErrorCodePathOutsideSandbox},
		{"//server/x", FileErrorCodePathOutsideSandbox},
		{"C:/x", FileErrorCodePathOutsideSandbox},
		{"c:x", FileErrorCodePathOutsideSandbox},
		{"..", FileErrorCodePathOutsideSandbox},
		{"../x", FileErrorCodePathOutsideSandbox},
		{"a/../../x", FileErrorCodePathOutsideSandbox},
		{"./../x", FileErrorCodePathOutsideSandbox},
	}
	for _, c := range cases {
		_, fileError := ProgramFileSystemPathFromBytes([]byte(c.path))
		if fileError == nil {
			t.Errorf("%q accepted", c.path)
			continue
		}
		if fileError.FileErrorCode != c.code || fileError.FileErrorOperation != FileOperationPathValidate {
			t.Errorf("%q: code %d operation %d, want code %d", c.path, fileError.FileErrorCode, fileError.FileErrorOperation, c.code)
		}
	}
}

func TestProgramFileSystemPathFromBytesAccepts(t *testing.T) {
	cases := map[string]string{
		"a.txt":       "a.txt",
		"d/a.txt":     "d/a.txt",
		"./a.txt":     "a.txt",
		"a/../b.txt":  "b.txt",
		"d//e/./f":    "d/e/f",
		"..a":         "..a",
		"a..":         "a..",
		"d/..x/y.txt": "d/..x/y.txt",
	}
	for path, want := range cases {
		programPath, fileError := ProgramFileSystemPathFromBytes([]byte(path))
		if fileError != nil {
			t.Errorf("%q rejected: %s", path, fileError.FileErrorMessage)
			continue
		}
		if got := programPath.ProgramFileSystemPathText(); got != want {
			t.Errorf("%q cleaned to %q, want %q", path, got, want)
		}
	}
	_, fileError := ProgramFileSystemPathFromBytes([]byte(strings.Repeat("a", ProgramFileSystemPathLengthMaximum)))
	if fileError != nil {
		t.Errorf("a path of the maximum length rejected: %s", fileError.FileErrorMessage)
	}
}

// programSandboxFixture makes <tmp>/sandbox and <tmp>/outside/secret.txt and
// opens the sandbox root.
func programSandboxFixture(t *testing.T) (root *os.Root, sandbox string, outside string) {
	t.Helper()
	dir := t.TempDir()
	sandbox = filepath.Join(dir, "sandbox")
	outside = filepath.Join(dir, "outside")
	for _, d := range []string{sandbox, outside} {
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	root, fileError := ProgramSandboxRootOpen(sandbox)
	if fileError != nil {
		t.Fatal(fileError.FileErrorMessage)
	}
	t.Cleanup(func() { root.Close() })
	return root, sandbox, outside
}

// programSandboxOutsideCheck fails unless outside holds only secret.txt, unchanged.
func programSandboxOutsideCheck(t *testing.T, outside string) {
	t.Helper()
	entries, err := os.ReadDir(outside)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "secret.txt" {
		names := []string{}
		for _, entry := range entries {
			names = append(names, entry.Name())
		}
		t.Errorf("outside directory holds %v", names)
	}
	if content, err := os.ReadFile(filepath.Join(outside, "secret.txt")); err != nil || string(content) != "secret" {
		t.Errorf("outside secret.txt is %q, %v", content, err)
	}
}

func programPathMust(t *testing.T, path string) ProgramFileSystemPath {
	t.Helper()
	programPath, fileError := ProgramFileSystemPathFromBytes([]byte(path))
	if fileError != nil {
		t.Fatal(fileError.FileErrorMessage)
	}
	return programPath
}

// programSandboxSymlink makes a symlink or skips where that needs privileges.
func programSandboxSymlink(t *testing.T, target string, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("cannot create a symlink here: %v", err)
	}
}

func TestProgramSandboxFileOperationsInside(t *testing.T) {
	root, sandbox, outside := programSandboxFixture(t)
	if err := os.Mkdir(filepath.Join(sandbox, "d"), 0o755); err != nil {
		t.Fatal(err)
	}
	programPath := programPathMust(t, "d/a.txt")
	file, fileError := ProgramSandboxFileCreate(root, programPath)
	if fileError != nil {
		t.Fatal(fileError.FileErrorMessage)
	}
	file.WriteString("hi")
	file.Close()
	if content, err := os.ReadFile(filepath.Join(sandbox, "d", "a.txt")); err != nil || string(content) != "hi" {
		t.Fatalf("created file holds %q, %v", content, err)
	}
	file, fileError = ProgramSandboxFileOpen(root, programPath)
	if fileError != nil {
		t.Fatal(fileError.FileErrorMessage)
	}
	file.Close()
	if content, fileError := ProgramSandboxFileRead(root, programPathMust(t, "./d/../d/a.txt")); fileError != nil || string(content) != "hi" {
		t.Fatalf("read %q, %v", content, fileError)
	}
	_, fileError = ProgramSandboxFileOpen(root, programPathMust(t, "missing.txt"))
	if fileError == nil || fileError.FileErrorCode != FileErrorCodeNotFound || fileError.FileErrorOperation != FileOperationOpen {
		t.Errorf("missing file gave %+v", fileError)
	}
	_, fileError = ProgramSandboxFileRead(root, programPathMust(t, "missing.txt"))
	if fileError == nil || fileError.FileErrorCode != FileErrorCodeNotFound || fileError.FileErrorOperation != FileOperationRead {
		t.Errorf("missing file read gave %+v", fileError)
	}
	programSandboxOutsideCheck(t, outside)
}

func TestProgramSandboxRootMissing(t *testing.T) {
	_, fileError := ProgramSandboxFileOpen(nil, programPathMust(t, "a.txt"))
	if fileError == nil || fileError.FileErrorCode != FileErrorCodeSandboxRootMissing {
		t.Errorf("open without a root gave %+v", fileError)
	}
}

func TestProgramSandboxSymlinkOut(t *testing.T) {
	root, sandbox, outside := programSandboxFixture(t)
	programSandboxSymlink(t, filepath.Join(outside, "secret.txt"), filepath.Join(sandbox, "link.txt"))
	programSandboxSymlink(t, outside, filepath.Join(sandbox, "linkdir"))
	programSandboxSymlink(t, filepath.Join(outside, "new.txt"), filepath.Join(sandbox, "dangling.txt"))

	for _, path := range []string{"link.txt", "linkdir/secret.txt", "dangling.txt"} {
		programPath := programPathMust(t, path)
		if _, fileError := ProgramSandboxFileOpen(root, programPath); fileError == nil || fileError.FileErrorCode != FileErrorCodePathOutsideSandbox {
			t.Errorf("open %q gave %+v", path, fileError)
		}
		if _, fileError := ProgramSandboxFileRead(root, programPath); fileError == nil || fileError.FileErrorCode != FileErrorCodePathOutsideSandbox {
			t.Errorf("read %q gave %+v", path, fileError)
		}
		if _, fileError := ProgramSandboxFileCreate(root, programPath); fileError == nil || fileError.FileErrorCode != FileErrorCodePathOutsideSandbox {
			t.Errorf("create %q gave %+v", path, fileError)
		}
	}
	if _, fileError := ProgramSandboxFileCreate(root, programPathMust(t, "linkdir/new.txt")); fileError == nil || fileError.FileErrorCode != FileErrorCodePathOutsideSandbox {
		t.Errorf("create linkdir/new.txt gave %+v", fileError)
	}
	programSandboxOutsideCheck(t, outside)
}

// A directory that is swapped for a symlink to the outside between two calls
// on the same root is caught at the second call.
func TestProgramSandboxDirectorySymlinkSwap(t *testing.T) {
	root, sandbox, outside := programSandboxFixture(t)
	d := filepath.Join(sandbox, "d")
	if err := os.Mkdir(d, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(d, "secret.txt"), []byte("inside"), 0o644); err != nil {
		t.Fatal(err)
	}
	programPath := programPathMust(t, "d/secret.txt")
	content, fileError := ProgramSandboxFileRead(root, programPath)
	if fileError != nil || string(content) != "inside" {
		t.Fatalf("first read %q, %+v", content, fileError)
	}

	if err := os.RemoveAll(d); err != nil {
		t.Fatal(err)
	}
	programSandboxSymlink(t, outside, d)

	if content, fileError := ProgramSandboxFileRead(root, programPath); fileError == nil || fileError.FileErrorCode != FileErrorCodePathOutsideSandbox {
		t.Errorf("read after the swap gave %q, %+v", content, fileError)
	}
	if _, fileError := ProgramSandboxFileCreate(root, programPath); fileError == nil || fileError.FileErrorCode != FileErrorCodePathOutsideSandbox {
		t.Errorf("create after the swap gave %+v", fileError)
	}
	if _, fileError := ProgramSandboxFileCreate(root, programPathMust(t, "d/new.txt")); fileError == nil || fileError.FileErrorCode != FileErrorCodePathOutsideSandbox {
		t.Errorf("create of a new file after the swap gave %+v", fileError)
	}
	programSandboxOutsideCheck(t, outside)
}

// os.OpenReadWrite creates a missing file, keeps the content of an existing
// one and writes at any offset, and os.Remove removes inside the sandbox.
func TestProgramSandboxOpenReadWriteAndRemove(t *testing.T) {
	root, sandbox, outside := programSandboxFixture(t)
	programPath := programPathMust(t, "rw.bin")
	file, fileError := ProgramSandboxFileOpenReadWrite(root, programPath)
	if fileError != nil {
		t.Fatal(fileError.FileErrorMessage)
	}
	file.WriteString("abcd")
	file.Close()
	file, fileError = ProgramSandboxFileOpenReadWrite(root, programPath)
	if fileError != nil {
		t.Fatal(fileError.FileErrorMessage)
	}
	file.WriteAt([]byte("X"), 2)
	if err := file.Sync(); err != nil {
		t.Fatal(err)
	}
	file.Close()
	if content, err := os.ReadFile(filepath.Join(sandbox, "rw.bin")); err != nil || string(content) != "abXd" {
		t.Fatalf("read-write file holds %q, %v", content, err)
	}
	if fileError := ProgramSandboxFileRemove(root, programPath); fileError != nil {
		t.Fatal(fileError.FileErrorMessage)
	}
	if _, err := os.Stat(filepath.Join(sandbox, "rw.bin")); !os.IsNotExist(err) {
		t.Fatalf("removed file: %v", err)
	}
	fileError = ProgramSandboxFileRemove(root, programPath)
	if fileError == nil || fileError.FileErrorCode != FileErrorCodeNotFound || fileError.FileErrorOperation != FileOperationRemove {
		t.Errorf("removing a missing file gave %+v", fileError)
	}
	if err := os.Mkdir(filepath.Join(sandbox, "d"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sandbox, "d", "x"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if fileError := ProgramSandboxFileRemove(root, programPathMust(t, "d")); fileError == nil {
		t.Error("removed a directory that is not empty")
	}
	if fileError := ProgramSandboxFileRemove(root, programPathMust(t, ".")); fileError == nil {
		t.Error("removed the sandbox root")
	}
	if _, fileError := ProgramSandboxFileOpenReadWrite(nil, programPath); fileError == nil || fileError.FileErrorCode != FileErrorCodeSandboxRootMissing {
		t.Errorf("read-write open without a root gave %+v", fileError)
	}
	if fileError := ProgramSandboxFileRemove(nil, programPath); fileError == nil || fileError.FileErrorCode != FileErrorCodeSandboxRootMissing {
		t.Errorf("remove without a root gave %+v", fileError)
	}
	programSandboxOutsideCheck(t, outside)
}

// A symlink inside the sandbox that points out is removed itself, and
// nothing outside is opened or removed through it.
func TestProgramSandboxRemoveAndReadWriteSymlinkOut(t *testing.T) {
	root, sandbox, outside := programSandboxFixture(t)
	programSandboxSymlink(t, filepath.Join(outside, "secret.txt"), filepath.Join(sandbox, "link.txt"))
	programSandboxSymlink(t, outside, filepath.Join(sandbox, "linkdir"))

	for _, path := range []string{"link.txt", "linkdir/secret.txt", "linkdir/new.txt"} {
		if _, fileError := ProgramSandboxFileOpenReadWrite(root, programPathMust(t, path)); fileError == nil || fileError.FileErrorCode != FileErrorCodePathOutsideSandbox {
			t.Errorf("read-write open %q gave %+v", path, fileError)
		}
	}
	if fileError := ProgramSandboxFileRemove(root, programPathMust(t, "linkdir/secret.txt")); fileError == nil || fileError.FileErrorCode != FileErrorCodePathOutsideSandbox {
		t.Errorf("remove linkdir/secret.txt gave %+v", fileError)
	}
	programSandboxOutsideCheck(t, outside)
	for _, path := range []string{"link.txt", "linkdir"} {
		if fileError := ProgramSandboxFileRemove(root, programPathMust(t, path)); fileError != nil {
			t.Errorf("remove the symlink %q: %s", path, fileError.FileErrorMessage)
		}
		if _, err := os.Lstat(filepath.Join(sandbox, path)); !os.IsNotExist(err) {
			t.Errorf("symlink %q still there: %v", path, err)
		}
	}
	programSandboxOutsideCheck(t, outside)
}
