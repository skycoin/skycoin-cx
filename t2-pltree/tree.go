package tree

import (
	"errors"

	"github.com/skycoin/skycoin-cx/t1-plloader"
)

func GetImportTree(programName string) (output string, err error) {
	if programName == "" {
		return output, errors.New("GetImportTree: empty program name")
	}
	stored, err := loader.ReadProgram(programName)
	if err != nil {
		return output, err
	}

	var importMap = make(map[string][]string)
	for _, pkg := range stored {
		imports, err := pkg.Imports()
		if err != nil {
			return output, err
		}
		importMap[pkg.Package.PackageName] = imports
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
