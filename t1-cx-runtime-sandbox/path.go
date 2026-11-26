package sandbox

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path"

	filelog "github.com/skycoin/skycoin-cx/t0-cx-runtime-filelog"
)

// PathLengthMaximum is the longest path a program may pass, in bytes.
const PathLengthMaximum = 4096

// Path is a path a running CX program passes to a file
// opcode, validated to stay inside the sandbox root, see #414 and #433.
type Path struct {
	pathText string
}

// Text returns the cleaned, slash separated path.
func (programPath Path) Text() string {
	return programPath.pathText
}

// PathFromBytes validates a path read from program memory.
// It must be relative, use only '/' and not start with ".." once cleaned.
// VIOLATION STRHOT-269: B, hands a path or text to Go or the OS, #412.
// VIOLATION VMALLOC-018: a system library call allocates in Go and the OS, #407.
func PathFromBytes(pathBytes []byte) (Path, *FileError) {
	invalid := func(code FileErrorCode, reason string) (Path, *FileError) {
		fileError := &FileError{
			FileErrorCode:      code,
			FileErrorOperation: FileOperationPathValidate,
			FileErrorMessage:   fmt.Sprintf("program path %q: %s", pathBytes, reason),
			FileErrorPath:      string(pathBytes),
		}
		if filelog.Enabled() {
			fmt.Printf("program path rejected: %s\n", fileError.FileErrorMessage)
		}
		return Path{}, fileError
	}
	switch {
	case len(pathBytes) == 0:
		return invalid(FileErrorCodePathInvalid, "empty")
	case len(pathBytes) > PathLengthMaximum:
		return invalid(FileErrorCodePathInvalid, "longer than 4096 bytes")
	case bytes.IndexByte(pathBytes, 0) >= 0:
		return invalid(FileErrorCodePathInvalid, "contains a NUL byte")
	case bytes.IndexByte(pathBytes, '\\') >= 0:
		return invalid(FileErrorCodePathInvalid, "contains '\\', use '/'")
	case pathBytes[0] == '/':
		return invalid(FileErrorCodePathOutsideSandbox, "absolute")
	case len(pathBytes) >= 2 && pathBytes[1] == ':' && pathDriveLetter(pathBytes[0]):
		return invalid(FileErrorCodePathOutsideSandbox, "has a volume name")
	}
	cleaned := path.Clean(string(pathBytes))
	if cleaned == ".." || len(cleaned) >= 3 && cleaned[:3] == "../" {
		return invalid(FileErrorCodePathOutsideSandbox, "leaves the sandbox")
	}
	return Path{pathText: cleaned}, nil
}

func pathDriveLetter(letter byte) bool {
	return letter >= 'A' && letter <= 'Z' || letter >= 'a' && letter <= 'z'
}

// RootOpen opens the directory every program path resolves in.
// An empty dir means ".".
func RootOpen(dir string) (*Root, *FileError) {
	if dir == "" {
		dir = "."
	}
	root, err := rootDirectoryOpen(dir)
	if err != nil {
		return nil, fileErrorFromFileSystemError(FileOperationSandboxRootOpen, dir, err)
	}
	return root, nil
}

// VIOLATION VMALLOC-018: a system library call allocates in Go and the OS, #407.
func rootCheck(root *Root, operation FileOperationCode, programPath Path) *FileError {
	if root != nil {
		return nil
	}
	return &FileError{
		FileErrorCode:      FileErrorCodeSandboxRootMissing,
		FileErrorOperation: operation,
		FileErrorMessage:   "no sandbox root is open for this program",
		FileErrorPath:      programPath.pathText,
	}
}

// FileOpen opens a program path for reading inside root.
// VIOLATION STRHOT-272: B, hands a path or text to Go or the OS, #412.
// VIOLATION VMALLOC-018: a system library call allocates in Go and the OS, #407.
func FileOpen(root *Root, programPath Path) (*os.File, *FileError) {
	if fileError := rootCheck(root, FileOperationOpen, programPath); fileError != nil {
		return nil, fileError
	}
	if filelog.Enabled() {
		fmt.Printf("FileOpen: Opening '%s' in '%s'\n", programPath.pathText, root.Name())
	}
	file, err := root.Open(programPath.pathText)
	if err != nil {
		return nil, pathFileError(FileOperationOpen, programPath, err)
	}
	return file, nil
}

// FileCreate creates or truncates a program path inside root.
// VIOLATION STRHOT-270: B, hands a path or text to Go or the OS, #412.
// VIOLATION VMALLOC-018: a system library call allocates in Go and the OS, #407.
func FileCreate(root *Root, programPath Path) (*os.File, *FileError) {
	if fileError := rootCheck(root, FileOperationCreate, programPath); fileError != nil {
		return nil, fileError
	}
	if filelog.Enabled() {
		fmt.Printf("FileCreate: Creating '%s' in '%s'\n", programPath.pathText, root.Name())
	}
	file, err := root.Create(programPath.pathText)
	if err != nil {
		return nil, pathFileError(FileOperationCreate, programPath, err)
	}
	return file, nil
}

// FileOpenReadWrite opens a program path inside root for
// reading and writing at any offset, creating it when it does not exist and
// keeping its content when it does.
// VIOLATION STRHOT-349: B, hands a path or text to Go or the OS, #412.
// VIOLATION VMALLOC-018: a system library call allocates in Go and the OS, #407.
func FileOpenReadWrite(root *Root, programPath Path) (*os.File, *FileError) {
	if fileError := rootCheck(root, FileOperationOpenReadWrite, programPath); fileError != nil {
		return nil, fileError
	}
	if filelog.Enabled() {
		fmt.Printf("FileOpenReadWrite: Opening '%s' in '%s'\n", programPath.pathText, root.Name())
	}
	file, err := rootFileOpen(root, programPath.pathText, os.O_RDWR|os.O_CREATE, 0o666)
	if err != nil {
		return nil, pathFileError(FileOperationOpenReadWrite, programPath, err)
	}
	return file, nil
}

// FileRemove removes a program path inside root, a file or an
// empty directory. A symlink is removed itself, never what it points to.
// VIOLATION STRHOT-350: B, hands a path or text to Go or the OS, #412.
// VIOLATION VMALLOC-018: a system library call allocates in Go and the OS, #407.
func FileRemove(root *Root, programPath Path) *FileError {
	if fileError := rootCheck(root, FileOperationRemove, programPath); fileError != nil {
		return fileError
	}
	if filelog.Enabled() {
		fmt.Printf("FileRemove: Removing '%s' in '%s'\n", programPath.pathText, root.Name())
	}
	if err := root.Remove(programPath.pathText); err != nil {
		return pathFileError(FileOperationRemove, programPath, err)
	}
	return nil
}

// FileRead reads a whole program path inside root.
func FileRead(root *Root, programPath Path) ([]byte, *FileError) {
	file, fileError := FileOpen(root, programPath)
	if fileError != nil {
		fileError.FileErrorOperation = FileOperationRead
		return nil, fileError
	}
	defer file.Close()
	content, err := io.ReadAll(file)
	if err != nil {
		return nil, pathFileError(FileOperationRead, programPath, err)
	}
	return content, nil
}

// VIOLATION STRHOT-271: B, hands a path or text to Go or the OS, #412.
// VIOLATION VMALLOC-018: a system library call allocates in Go and the OS, #407.
func pathFileError(operation FileOperationCode, programPath Path, err error) *FileError {
	fileError := fileErrorFromFileSystemError(operation, programPath.pathText, err)
	if filelog.Enabled() {
		fmt.Printf("program file operation %d failed: %s\n", operation, fileError.FileErrorMessage)
	}
	return fileError
}
