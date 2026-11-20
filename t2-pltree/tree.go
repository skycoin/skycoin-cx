package tree

import (
	"bufio"
	"errors"
	"fmt"
	"strings"

	"github.com/skycoin/skycoin-cx/t0-plbolt" // thin: only Get; no depth gain
	"github.com/skycoin/skycoin-cx/t1-plloader"
)

// NOTE: the read and decode walk is repeated in four packages. See #92.
func GetImportTree(programName string) (output string, err error) {
	if programName == "" {
		return output, errors.New("GetImportTree: empty program name")
	}
	var packageList loader.PackageList
	listBytes, err := bolt.Get(programName)
	if err != nil {
		return output, err
	}
	if err := packageList.UnmarshalBinary(listBytes); err != nil {
		return output, fmt.Errorf("decode package list %s: %w", programName, err)
	}

	var importMap = make(map[string][]string)
	for _, packageKey := range packageList.Packages {
		var imports []string
		var packageStruct loader.Package
		packageBytes, err := bolt.Get(packageKey)
		if err != nil {
			return output, err
		}
		if err := packageStruct.UnmarshalBinary(packageBytes); err != nil {
			return output, fmt.Errorf("decode package %s: %w", packageKey, err)
		}

		for _, fileKey := range packageStruct.Files {
			var fileStruct loader.File
			fileBytes, err := bolt.Get(fileKey)
			if err != nil {
				return output, err
			}
			if err := fileStruct.UnmarshalBinary(fileBytes); err != nil {
				return output, fmt.Errorf("decode file %s: %w", fileKey, err)
			}

			scanner := bufio.NewScanner(strings.NewReader(string(fileStruct.Content)))
			scanner.Split(bufio.ScanWords)

			wordBefore := ""
			for scanner.Scan() {
				if scanner.Text() != "import" {
					wordBefore = scanner.Text()
					continue
				}
				if wordBefore == "//" {
					wordBefore = scanner.Text()
					continue
				}
				// NOTE: unreachable, the word is always "import" here. The scan also takes
				// "import" in block comments and strings as an import. See #98.
				if scanner.Text() == "var" || scanner.Text() == "const" || scanner.Text() == "type" || scanner.Text() == "func" {
					break
				}
				if !scanner.Scan() {
					break
				}
				tok := scanner.Text()
				if len(tok) < 2 || tok[0] != '"' || tok[len(tok)-1] != '"' {
					// "import" not followed by a quoted path, so not an import statement.
					wordBefore = tok
					continue
				}
				importString := tok[1 : len(tok)-1]
				if !loader.Contains(imports, importString) {
					imports = append(imports, importString)
				}
				wordBefore = scanner.Text()
			}
			if err := scanner.Err(); err != nil {
				return output, fmt.Errorf("file %s: scan: %w", fileStruct.FileName, err)
			}
		}
		importMap[packageStruct.PackageName] = imports
	}
	output += "main\n"
	var layers = make(map[int]bool)
	for i, mainImport := range importMap["main"] {
		if i == len(importMap["main"])-1 {
			output += "`--" + mainImport + "\n"
			layers[0] = false
		} else {
			output += "|--" + mainImport + "\n"
			layers[0] = true
		}
		onPath := map[string]bool{"main": true, mainImport: true}
		output += addDependencies(importMap, importMap[mainImport], 1, layers, onPath)
	}
	return output, nil
}

func AddDependenciesRecur(importMap map[string][]string, dependencies []string, depth int, layers map[int]bool) (output string) {
	return addDependencies(importMap, dependencies, depth, layers, map[string]bool{})
}

// addDependencies is AddDependenciesRecur with the packages on the current
// branch in onPath. The import map comes from stored data, so a cycle is
// printed once with a marker instead of recursing forever.
func addDependencies(importMap map[string][]string, dependencies []string, depth int, layers map[int]bool, onPath map[string]bool) (output string) {
	for i, dependencyImport := range dependencies {
		for l := 0; l < depth; l++ {
			if layers[l] {
				output += "|  "
			} else {
				output += "   "
			}
		}
		branch := "|--"
		layers[depth] = true
		if i == len(dependencies)-1 {
			branch = "`--"
			layers[depth] = false
		}
		if onPath[dependencyImport] {
			output += branch + dependencyImport + " (import cycle)\n"
			continue
		}
		output += branch + dependencyImport + "\n"
		onPath[dependencyImport] = true
		output += addDependencies(importMap, importMap[dependencyImport], depth+1, layers, onPath)
		delete(onPath, dependencyImport)
	}
	return output
}
