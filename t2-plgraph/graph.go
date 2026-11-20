package graph

import (
	"errors"
	"strconv"

	"github.com/skycoin/skycoin-cx/t1-plloader"
)

func GetImportGraph(programName string) (output string, err error) {
	if programName == "" {
		return output, errors.New("GetImportGraph: empty program name")
	}
	stored, err := loader.ReadProgram(programName)
	if err != nil {
		return output, err
	}

	var importMap = make(map[string][]string)
	var indexList = make([]string, 0)
	var importIndexMap = make(map[string]int)
	for _, pkg := range stored {
		imports, err := pkg.Imports()
		if err != nil {
			return output, err
		}
		indexList = append(indexList, pkg.Package.PackageName)
		importIndexMap[pkg.Package.PackageName] = len(indexList) - 1
		importMap[pkg.Package.PackageName] = imports
	}
	var otherImports []string
	for i, module := range indexList {
		output += "id=" + strconv.Itoa(i) + ", "
		output += "module=" + module + ", "
		var importIDs = "imports="
		imports := importMap[module]
		for _, importString := range imports {
			importIndex, ok := importIndexMap[importString]
			if !ok {
				importIndex = len(indexList) + len(otherImports)
				importIndexMap[importString] = importIndex
				otherImports = append(otherImports, "id="+strconv.Itoa(importIndex)+", module="+importString+", imports=,\n")
			}
			importIDs += strconv.Itoa(importIndex) + ","
		}
		output += importIDs + "\n"
	}
	for _, otherImport := range otherImports {
		output += otherImport
	}
	return output, nil
}
