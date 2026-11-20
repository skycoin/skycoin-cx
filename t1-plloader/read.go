package loader

import (
	"errors"
	"fmt"

	"github.com/skycoin/skycoin-cx/t0-plbolt"
)

// StoredPackage is a package read back from the database with its files.
type StoredPackage struct {
	Package Package
	Files   []File
}

// ReadProgram reads the program stored under programName, its packages and
// their files, in stored order.
func ReadProgram(programName string) ([]StoredPackage, error) {
	if programName == "" {
		return nil, errors.New("read program: empty program name")
	}
	var packageList PackageList
	listBytes, err := bolt.Get(programName)
	if err != nil {
		return nil, err
	}
	if err := packageList.UnmarshalBinary(listBytes); err != nil {
		return nil, fmt.Errorf("decode package list %s: %w", programName, err)
	}

	packages := make([]StoredPackage, 0, len(packageList.Packages))
	for _, packageKey := range packageList.Packages {
		var stored StoredPackage
		packageBytes, err := bolt.Get(packageKey)
		if err != nil {
			return nil, err
		}
		if err := stored.Package.UnmarshalBinary(packageBytes); err != nil {
			return nil, fmt.Errorf("decode package %s: %w", packageKey, err)
		}
		for _, fileKey := range stored.Package.Files {
			var file File
			fileBytes, err := bolt.Get(fileKey)
			if err != nil {
				return nil, err
			}
			if err := file.UnmarshalBinary(fileBytes); err != nil {
				return nil, fmt.Errorf("decode file %s of package %s: %w", fileKey, stored.Package.PackageName, err)
			}
			stored.Files = append(stored.Files, file)
		}
		packages = append(packages, stored)
	}
	return packages, nil
}

// Imports returns the imports of all files of the package, each once.
func (p StoredPackage) Imports() ([]string, error) {
	var imports []string
	for _, file := range p.Files {
		fileImports, err := Imports(file.Content)
		if err != nil {
			return nil, fmt.Errorf("file %s: %w", file.FileName, err)
		}
		for _, importString := range fileImports {
			if !Contains(imports, importString) {
				imports = append(imports, importString)
			}
		}
	}
	return imports, nil
}
