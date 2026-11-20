package main

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"

	"github.com/skycoin/skycoin-cx/t0-plbolt" // thin: only Init and Close; no depth gain
	"github.com/skycoin/skycoin-cx/t0-util"
	"github.com/skycoin/skycoin-cx/t1-plloader"  // thin: only LoadCXProgram; no depth gain
	"github.com/skycoin/skycoin-cx/t2-plencoder" // thin: only SavePackagesToDisk; no depth gain
	"github.com/skycoin/skycoin-cx/t2-plgraph"   // thin: only GetImportGraph; no depth gain
	"github.com/skycoin/skycoin-cx/t2-pltree"    // thin: only GetImportTree; no depth gain
)

func main() {
	loadFlag := flag.Bool("load", false, "OPTION: Load a program to the database, with a given name to load it as and path to the program")
	saveFlag := flag.Bool("save", false, "OPTION: Save a package to disk, with a given name to search on the database and a new directory path to save to")
	treeFlag := flag.Bool("tree", false, "OPTION: Print the import dependency tree for a given program on the database")
	graphFlag := flag.Bool("graph", false, "OPTION: Print the import dependencies for each module in a program on the database")
	nameFlag := flag.String("name", "", "The name of the program to load or save")
	pathFlag := flag.String("path", "", "The path to the program to load or save")
	dbFlag := flag.String("db", "", "The path to the database file, created if missing")
	flag.Parse()

	if *dbFlag == "" {
		log.Fatal("-db is required")
	}
	if err := bolt.Init(*dbFlag); err != nil {
		log.Fatal(err)
	}
	defer bolt.Close()

	// NOTE: counts set flags rather than what an action needs. See #102.
	if flag.NFlag()+flag.NArg() > 4 || flag.NFlag()+flag.NArg() < 2 {
		log.Fatal("Wrong number of arguments. Type -help for more information")
	}
	actions := 0
	for _, set := range []bool{*loadFlag, *saveFlag, *treeFlag, *graphFlag} {
		if set {
			actions++
		}
	}
	if actions != 1 {
		log.Fatal("Give exactly one of -load, -save, -tree or -graph. Type -help for more information")
	}
	if *nameFlag == "" {
		log.Fatal("-name is required")
	}

	programName := *nameFlag

	if *treeFlag {
		output, err := tree.GetImportTree(programName)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Print(output)
		return
	}

	if *graphFlag {
		output, err := graph.GetImportGraph(programName)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Print(output)
		return
	}

	if *pathFlag == "" {
		log.Fatal("-path is required with -load and -save")
	}
	path := filepath.Clean(*pathFlag)

	if *loadFlag {
		_, sourceCodes, _, err := util.ParseArgsForCX([]string{path}, true)
		if err != nil {
			log.Fatal(err)
		}
		err = loader.LoadCXProgram(programName, sourceCodes)
		if err != nil {
			log.Fatal(err)
		}
		return
	}
	if *saveFlag {
		err := encoder.SavePackagesToDisk(programName, path)
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	log.Fatal("Wrong arguments provided. Type -help for more information")
}
