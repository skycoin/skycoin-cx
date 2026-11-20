package file_output_test

import (
	"testing"

	"github.com/skycoin/skycoin-cx/t0-util"
	"github.com/skycoin/skycoin-cx/t1-plloader"
	"github.com/skycoin/skycoin-cx/t4-actions" // thin: tests use only AST; tests only set actions.AST = nil, could drop it, tier 5 to 2
	"github.com/skycoin/skycoin-cx/t5-plfileoutput"
)

func TestFile_Output_GetImportFiles(t *testing.T) {

	type pkg struct {
		pkgName string
		imports []string
	}
	tests := []struct {
		scenario    string
		programName string
		testDir     string
		files       []loader.File
	}{
		{
			scenario:    "Has Imports",
			programName: "tester",
			testDir:     "./test_files/test_valid_program",
			files: []loader.File{
				{
					FileName: "testimport.cx",
				},
				{
					FileName: "testfile.cx",
				},
				{
					FileName: "testfile.cx",
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.scenario, func(t *testing.T) {

			actions.AST = nil

			_, sourceCodes, _, perr := util.ParseArgsForCX([]string{tc.testDir}, true)
			if perr != nil {
				t.Fatal(perr)
			}
			err := loader.LoadCXProgram(tc.programName, sourceCodes)
			if err != nil {
				t.Fatal(err)
			}

			files, err := file_output.GetImportFiles(tc.programName)
			if err != nil {
				t.Fatal(err)
			}

			for _, wantFile := range tc.files {
				var match bool
				var wantFileName string = wantFile.FileName
				var gotFileName string
				for _, gotFile := range files {
					gotFileName = gotFile.FileName
					if gotFileName == wantFileName {
						match = true
						break
					}
				}

				if !match {
					t.Errorf("want file %s, got %s", wantFileName, gotFileName)
				}
			}

		})
	}
}
