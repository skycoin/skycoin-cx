package encoder

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/skycoin/skycoin-cx/t1-plloader" // thin: only ReadProgram and StoredPackage; no depth gain
)

func SavePackagesToDisk(programName string, outDir string) error {
	if programName == "" {
		return errors.New("save packages: empty program name")
	}
	if outDir == "" {
		return fmt.Errorf("save packages %s: empty output path", programName)
	}
	stored, err := loader.ReadProgram(programName)
	if err != nil {
		return err
	}
	err = os.Mkdir(outDir, 0755)
	if err != nil {
		return err
	}

	for _, pkg := range stored {
		dir := filepath.Join(outDir, "src")
		if pkg.Package.PackageName != "main" {
			if err := checkPathElement(pkg.Package.PackageName); err != nil {
				return fmt.Errorf("package %s: %w", pkg.Package.PackageName, err)
			}
			dir = filepath.Join(dir, pkg.Package.PackageName)
		}

		err = SaveFilesToDisk(pkg, dir)
		if err != nil {
			return err
		}
	}
	return nil
}

func SaveFilesToDisk(pkg loader.StoredPackage, dir string) error {
	if dir == "" {
		return fmt.Errorf("save files of package %s: empty output path", pkg.Package.PackageName)
	}
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return err
	}
	for _, file := range pkg.Files {
		if err := checkPathElement(file.FileName); err != nil {
			return fmt.Errorf("file %s of package %s: %w", file.FileName, pkg.Package.PackageName, err)
		}
		err = os.WriteFile(filepath.Join(dir, file.FileName), file.Content, 0644)
		if err != nil {
			return err
		}
	}
	return nil
}

// checkPathElement rejects a name read from the database that is not a single
// path element, so stored data cannot write outside the output directory.
func checkPathElement(name string) error {
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, `/\`) || filepath.Base(name) != name || filepath.IsAbs(name) || filepath.VolumeName(name) != "" {
		return fmt.Errorf("unsafe name %q", name)
	}
	return nil
}
