package loader

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/skycoin/skycoin-cx/t0-plbolt" // thin: only Add; no depth gain
	"golang.org/x/crypto/blake2b"
)

// NOTE: a hand kept copy of the default packages in t0-packages. See #95.
var builtinPackages = []string{"al", "gl", "glfw", "time", "os", "gltext", "cx", "json", "cipher", "tcp"}

// VIOLATION: state outside State. Owner: t1-plloader; moves to PackageFileMng in phase 1.
// NOTE: written but never read, and keyed by base file name, so files in different
// packages overwrite each other. See #94.
var FileHashMap = make(map[string]string)

// VIOLATION: state outside State. Owner: t1-plloader; moves to PackageFileMng in phase 1.
// NOTE: written but never read. See #94.
var PackageHashMap = make(map[string]string)

func LoadCXProgram(programName string, sourceCode []*os.File) (err error) {
	if programName == "" {
		return errors.New("load program: empty program name")
	}
	if len(sourceCode) == 0 {
		return fmt.Errorf("load program %s: no source files", programName)
	}

	// Gets the source files
	fileMap, err := createFileMap(sourceCode)
	if err != nil {
		return err
	}

	var packageListStruct PackageList
	importMap := make(map[string][]string)

	// Start with the main package
	err = addNewPackage(&packageListStruct, "main", fileMap, importMap)
	if err != nil {
		return err
	}

	// load the imported packages
	err = loadImportPackages(&packageListStruct, "main", fileMap, importMap)
	if err != nil {
		return err
	}

	err = checkForDependencyLoop(importMap)
	if err != nil {
		return err
	}

	value, err := packageListStruct.MarshalBinary()
	if err != nil {
		return err
	}
	return bolt.Add(programName, value)
}

func createFileMap(files []*os.File) (fileMap map[string][]*os.File, err error) {
	fileMap = make(map[string][]*os.File)
	for _, file := range files {
		if file == nil {
			return fileMap, errors.New("create file map: nil source file")
		}
		absPath, err := filepath.Abs(file.Name())
		if err != nil {
			return fileMap, fmt.Errorf("%s: resolve path: %w", file.Name(), err)
		}
		packageName := filepath.Base(filepath.Dir(absPath))
		filePackageName, err := getPackageName(file)
		if err != nil {
			return fileMap, err
		}
		if packageName == "src" {
			if filePackageName != "main" {
				return fileMap, fmt.Errorf("%s: package error: package %s found in main", filepath.Base(file.Name()), filePackageName)
			}
			fileMap["main"] = append(fileMap["main"], file)
			continue
		}
		if filePackageName != packageName {
			return fileMap, fmt.Errorf("%s: package error: package %s found in %v", filepath.Base(file.Name()), filePackageName, packageName)
		}
		fileMap[packageName] = append(fileMap[packageName], file)
	}
	return fileMap, nil
}

func getPackageName(file *os.File) (string, error) {
	if file == nil {
		return "", errors.New("get package name: nil file")
	}
	openFile, err := os.Open(file.Name())
	if err != nil {
		return "", err
	}
	defer openFile.Close()

	scanner := bufio.NewScanner(openFile)

	for scanner.Scan() {
		line := strings.Fields(scanner.Text())
		if len(line) > 0 && line[0] == "package" {
			if len(line) < 2 {
				return "", fmt.Errorf("%s: package clause without a name", file.Name())
			}
			return line[1], nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("%s: read: %w", file.Name(), err)
	}
	return "", errors.New("file doesn't contain a package name")
}

//	Adds a new package to the package list and add imports of the new package to the import map
//	1. packageListStruct - package list pointer
//	2. packageName -  name of package to be added
//	3. fileMap - file map that contains the files
//	4. importMap - import map that contains the imports
//
// This function contains steps 4 - 9 of package loader
func addNewPackage(packageListStruct *PackageList, packageName string, fileMap map[string][]*os.File, importMap map[string][]string) error {
	if packageListStruct == nil || fileMap == nil || importMap == nil {
		return errors.New("add package: nil package list, file map or import map")
	}
	if packageName == "" {
		return errors.New("add package: empty package name")
	}

	// Checks if package is found in the directory
	files, ok := fileMap[packageName]
	if !ok && !Contains(builtinPackages, packageName) {
		return fmt.Errorf("package %s not found", packageName)
	}

	// Skip if the import is a built-in package
	if Contains(builtinPackages, packageName) {
		return nil
	}

	// Creates the package struct
	packageStruct, err := createPackageStruct(packageName, files, Package{})
	if err != nil {
		return err
	}

	// Creates the import list
	importList, err := createImportList(files, []string{})
	if err != nil {
		return err
	}

	// Removes duplicates of imports and adds them to the import map
	newImportList := RemoveDuplicates(importList)
	importMap[packageName] = newImportList

	// Append the package to the package struct
	return packageListStruct.appendPackage(&packageStruct)
}

func getImports(file *os.File) (imports []string, err error) {
	if file == nil {
		return nil, errors.New("get imports: nil file")
	}
	openFile, err := os.Open(file.Name())
	if err != nil {
		return imports, err
	}
	defer openFile.Close()

	scanner := bufio.NewScanner(openFile)

	for scanner.Scan() {
		line := strings.Fields(scanner.Text())
		if len(line) == 0 || line[0] != "import" {
			continue
		}
		if len(line) < 2 || len(line[1]) < 2 || line[1][0] != '"' || line[1][len(line[1])-1] != '"' {
			return nil, fmt.Errorf("%s: malformed import line %q", file.Name(), scanner.Text())
		}
		imports = append(imports, line[1][1:len(line[1])-1])
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("%s: read: %w", file.Name(), err)
	}
	return imports, nil
}

// checkForDependencyLoop walks the imports depth first in name order and
// reports the first loop it finds.
func checkForDependencyLoop(importMap map[string][]string) error {
	names := make([]string, 0, len(importMap))
	for name := range importMap {
		names = append(names, name)
	}
	sort.Strings(names)

	done := make(map[string]bool)
	var path []string
	var visit func(name string) error
	visit = func(name string) error {
		for i, onPath := range path {
			if onPath != name {
				continue
			}
			if i == len(path)-1 {
				return errors.New("Module " + name + " imports itself")
			}
			return errors.New("Dependency loop between modules " + strings.Join(append(path[i:len(path):len(path)], name), " -> "))
		}
		if done[name] {
			return nil
		}
		path = append(path, name)
		for _, importName := range importMap[name] {
			if err := visit(importName); err != nil {
				return err
			}
		}
		path = path[:len(path)-1]
		done[name] = true
		return nil
	}
	for _, name := range names {
		if err := visit(name); err != nil {
			return err
		}
	}
	return nil
}

func fileStructFromFile(file *os.File) (File, error) {
	if file == nil {
		return File{}, errors.New("file struct: nil file")
	}
	fileName := filepath.Base(file.Name())
	fileBytes, err := ioutil.ReadFile(file.Name())
	if err != nil {
		return File{}, fmt.Errorf("%s: read: %w", file.Name(), err)
	}
	if uint64(len(fileBytes)) > math.MaxUint32 {
		return File{}, fmt.Errorf("%s: %d bytes does not fit the uint32 Length field", file.Name(), len(fileBytes))
	}
	fileHash := blake2b.Sum512(fileBytes)

	fileStruct := File{
		FileName:   fileName,
		Length:     uint32(len(fileBytes)),
		Content:    fileBytes,
		Blake2Hash: fmt.Sprintf("%x", fileHash[:]),
	}

	return fileStruct, nil
}

// Creates Blake2Hash from file UUIDs
func blake2HashFromFileUUID(fileUUID []string) ([64]byte, error) {
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)

	err := encoder.Encode(fileUUID)
	if err != nil {
		return [64]byte{}, err
	}
	return blake2b.Sum512(buffer.Bytes()), nil
}

//	Creates Package Struct
//	1. name - name of the struct
//	2. files - file list that are part of the package
//	3. packageStruct - pass an empty package struct "Package{}"
//
// This function works recursively
// NOTE: recursion with an accumulator the caller must pass, a loop would do. See #103.
func createPackageStruct(name string, files []*os.File, packageStruct Package) (Package, error) {

	// Base case
	if len(files) == 0 {
		return packageStruct, nil
	}

	packageStruct.PackageName = name

	// Create file struct
	currentIndex := len(files) - 1
	currentFile := files[currentIndex]
	fileStruct, err := fileStructFromFile(currentFile)
	if err != nil {
		return packageStruct, err
	}

	// Append file struct
	err = packageStruct.appendFile(&fileStruct)
	if err != nil {
		return packageStruct, err
	}

	// Remove file from slice
	newFiles := files[:currentIndex]

	// Generate Blake2Hash and add to package struct
	if len(newFiles) == 0 {
		hash, err := blake2HashFromFileUUID(packageStruct.Files)
		if err != nil {
			return packageStruct, err
		}

		packageStruct.Blake2Hash = fmt.Sprintf("%x", hash[:])
	}

	return createPackageStruct(name, newFiles, packageStruct)
}

//	 Create an import list from the list of files
//		1. files - files list
//		2. importList - empty string slice "[]string{}"
//
// This function works recursively
// NOTE: recursion with an accumulator the caller must pass, a loop would do. See #103.
func createImportList(files []*os.File, importList []string) ([]string, error) {
	// Base case
	if len(files) == 0 {
		return importList, nil
	}

	// Gets imports
	currentIndex := len(files) - 1
	currentFile := files[currentIndex]
	imports, err := getImports(currentFile)
	if err != nil {
		return importList, err
	}

	// Removes file from list
	importList = append(importList, imports...)
	newFiles := files[:currentIndex]

	return createImportList(newFiles, importList)
}

//	 Loads Import Packages
//		1. packageListStruct - package list pointer
//		2. parentName - package with imports
//		3. fileMap - file map that contains the files
//		4. importMap - import map that contains the imports
//
// This function works recursively, loading the import packages and then loading import packages of imports
func loadImportPackages(packageListStruct *PackageList, parentName string, fileMap map[string][]*os.File, importMap map[string][]string) error {
	for _, importName := range importMap[parentName] {
		// A package already in importMap is loaded. Skipping it stops an import
		// loop from recursing forever, checkForDependencyLoop reports it later.
		if _, loaded := importMap[importName]; loaded {
			continue
		}
		err := addNewPackage(packageListStruct, importName, fileMap, importMap)
		if err != nil {
			return err
		}

		// Load the imports of the import
		err = loadImportPackages(packageListStruct, importName, fileMap, importMap)
		if err != nil {
			return err
		}
	}

	return nil
}
