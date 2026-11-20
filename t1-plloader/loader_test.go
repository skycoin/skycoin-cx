package loader_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/skycoin/skycoin-cx/t0-plbolt"
	"github.com/skycoin/skycoin-cx/t0-util"
	"github.com/skycoin/skycoin-cx/t1-plloader"
	"golang.org/x/crypto/blake2b"
)

const TEST_SRC_PATH_INVALID = "test_folder/test_invalid_program/src/"
const TEST_SRC_PATH_VALID = "test_folder/test_valid_program/src/"
const TEST_SRC_PATH_LOOP = "test_folder/test_loop_program/src/"
const TEST_PACKAGE_FILE = "test_folder/test_various_files/package.cx"

func TestContains(t *testing.T) {
	tests := []struct {
		Scenario string
		List     []string
		Element  string
		WantBool bool
	}{
		{
			Scenario: "Testing if list does contain the element",
			List:     []string{"a", "b", "c"},
			Element:  "a",
			WantBool: true,
		},
		{
			Scenario: "Testing if list doesn't contain the element",
			List:     []string{"a", "b", "c"},
			Element:  "d",
			WantBool: false,
		},
	}
	for _, testcase := range tests {
		t.Run(testcase.Scenario, func(t *testing.T) {
			gotBool := loader.Contains(testcase.List, testcase.Element)
			if gotBool != testcase.WantBool {
				t.Errorf("want bool %v, got %v", gotBool, testcase.WantBool)
			}
		})
	}
}

func TestRemoveDuplicates(t *testing.T) {
	tests := []struct {
		Scenario string
		List     []string
		WantList []string
	}{
		{
			Scenario: "Testing list without duplicates",
			List:     []string{"a", "b", "c"},
			WantList: []string{"a", "b", "c"},
		},
		{
			Scenario: "Testing list with duplicates",
			List:     []string{"a", "b", "b", "c", "c"},
			WantList: []string{"a", "b", "c"},
		},
	}
	for _, testcase := range tests {
		t.Run(testcase.Scenario, func(t *testing.T) {
			gotList := loader.RemoveDuplicates(testcase.List)
			if !reflect.DeepEqual(gotList, testcase.WantList) {
				t.Errorf("want list %v, got %v", testcase.WantList, gotList)
			}
		})
	}
}

func TestGetPackageName(t *testing.T) {
	tests := []struct {
		Scenario        string
		FilesPath       string
		WantPackageName string
		WantErr         error
	}{
		{
			Scenario:        "Test with file that has package name",
			FilesPath:       TEST_PACKAGE_FILE,
			WantPackageName: "main",
			WantErr:         nil,
		},
		{
			Scenario:        "Test with file that has no package name",
			FilesPath:       "test_folder/test_various_files/nopackage.cx",
			WantPackageName: "",
			WantErr:         errors.New("file doesn't contain a package name"),
		},
	}
	for _, testcase := range tests {
		t.Run(testcase.Scenario, func(t *testing.T) {
			file, err := os.Open(testcase.FilesPath)
			if err != nil {
				t.Error(err)
			}

			gotPackageName, gotErr := loader.GetPackageName(file)

			if gotPackageName != testcase.WantPackageName {
				t.Errorf("want package name %s, got %s", testcase.WantPackageName, gotPackageName)
			}

			if (gotErr != nil && testcase.WantErr == nil) ||
				(gotErr == nil && testcase.WantErr != nil) {
				t.Errorf("want error %v, got %v", testcase.WantErr, gotErr)
			}
			if gotErr != nil && testcase.WantErr != nil {
				if gotErr.Error() != testcase.WantErr.Error() {
					t.Errorf("want error %v, got %v", testcase.WantErr, gotErr)
				}
			}
		})
	}

}

func TestCreateFileMap(t *testing.T) {
	tests := []struct {
		Scenario    string
		FilesPath   string
		WantFileMap map[string][]string
		WantErr     error
	}{
		{
			Scenario:  "valid file map",
			FilesPath: TEST_SRC_PATH_VALID,
			WantFileMap: map[string][]string{
				"main":       {"testfile.cx", "testfile2.cx"},
				"testimport": {"testimport.cx"},
			},
		},
		{
			Scenario:    "no package name",
			FilesPath:   "test_folder/test_various_files/nopackage.cx",
			WantFileMap: map[string][]string{},
			WantErr:     errors.New("file doesn't contain a package name"),
		},
		{
			Scenario:  "error in main package",
			FilesPath: TEST_SRC_PATH_INVALID,
			WantFileMap: map[string][]string{
				"main": {"testfile.cx"},
			},
			WantErr: errors.New("testfile2.cx: package error: package main2 found in main"),
		},
		{
			Scenario:    "multiple packages in directory",
			FilesPath:   "test_folder/test_package_error_program",
			WantFileMap: map[string][]string{},
			WantErr:     errors.New("testimportFile2.cx: package error: package testimport2 found in testimport"),
		},
	}

	for _, testcase := range tests {
		t.Run(testcase.Scenario, func(t *testing.T) {
			_, sourceCodes, _, perr := util.ParseArgsForCX([]string{testcase.FilesPath}, true)
			if perr != nil {
				t.Fatal(perr)
			}
			gotFileMap, gotErr := loader.CreateFileMap(sourceCodes)
			for wantKey, wantValue := range testcase.WantFileMap {
				gotValue, ok := gotFileMap[wantKey]
				if !ok {
					t.Errorf("package %s not found in file map", wantKey)
				}
				for _, wantFile := range wantValue {
					var match bool
					var gotFileName string
					for _, gotFile := range gotValue {
						gotFileName = filepath.Base(gotFile.Name())
						if wantFile == gotFileName {
							match = true
							break
						}
					}
					if !match {
						t.Errorf("want file %s, got %s", wantFile, gotFileName)
					}
				}
			}
			if (gotErr != nil && testcase.WantErr == nil) ||
				(gotErr == nil && testcase.WantErr != nil) {
				t.Errorf("want error %v, got %v", testcase.WantErr, gotErr)
			}

			if gotErr != nil && testcase.WantErr != nil {
				if gotErr.Error() != testcase.WantErr.Error() {
					t.Errorf("want error %v, got %v", testcase.WantErr, gotErr)
				}
			}
		})
	}
}

func TestFileStructFromFile(t *testing.T) {
	tests := []struct {
		Scenario       string
		FilesPath      string
		WantFileStruct loader.File
		WantErr        error
	}{
		{
			Scenario:  "Testing package file with no errors",
			FilesPath: TEST_PACKAGE_FILE,
			WantFileStruct: loader.File{
				FileName: "package.cx",
			},
			WantErr: nil,
		},
	}
	for _, testcase := range tests {
		t.Run(testcase.Scenario, func(t *testing.T) {
			file, err := os.Open(testcase.FilesPath)
			if err != nil {
				t.Error(err)
			}
			gotFileStruct, gotErr := loader.FileStructFromFile(file)

			if gotFileStruct.FileName != testcase.WantFileStruct.FileName {
				t.Errorf("want file %s, got %s", testcase.WantFileStruct.FileName, gotFileStruct.FileName)
			}

			if (gotErr != nil && testcase.WantErr == nil) ||
				(gotErr == nil && testcase.WantErr != nil) {
				t.Errorf("want error %v, got %v", testcase.WantErr, gotErr)
			}

			if gotErr != nil && testcase.WantErr != nil {
				if gotErr.Error() != testcase.WantErr.Error() {
					t.Errorf("want error %v, got %v", testcase.WantErr, gotErr)
				}
			}
		})
	}

}

func TestAddNewPackage(t *testing.T) {
	tests := []struct {
		Scenario             string
		FilesPath            string
		WantNumberOfPackages int
		WantImportMap        map[string][]string
		WantErr              error
	}{
		{
			Scenario:             "Test adding package to Bolt database",
			FilesPath:            TEST_SRC_PATH_VALID,
			WantNumberOfPackages: 1,
			WantImportMap: map[string][]string{
				"main": {"os", "testimport"},
			},
			WantErr: nil,
		},
		{
			Scenario:             "Test adding package to Bolt database with missing package",
			FilesPath:            "test_folder/test_valid_program/",
			WantNumberOfPackages: 0,
			WantImportMap:        map[string][]string{},
			WantErr:              errors.New("package main not found"),
		},
	}
	for _, testcase := range tests {
		t.Run(testcase.Scenario, func(t *testing.T) {
			_, sourceCodes, _, perr := util.ParseArgsForCX([]string{testcase.FilesPath}, false)
			if perr != nil {
				t.Fatal(perr)
			}
			fileMap, err := loader.CreateFileMap(sourceCodes)
			if err != nil {
				t.Error(err)
			}
			gotImportMap := make(map[string][]string)
			testPackageStruct := loader.PackageList{}
			gotErr := loader.AddNewPackage(&testPackageStruct, "main", fileMap, gotImportMap)

			gotNumberOfPackages := len(testPackageStruct.Packages)
			if gotNumberOfPackages != testcase.WantNumberOfPackages {
				t.Errorf("want %d packages, got %d packages", testcase.WantNumberOfPackages, gotNumberOfPackages)
			}

			if !reflect.DeepEqual(gotImportMap, testcase.WantImportMap) {
				t.Errorf("want import map %v, got %v", testcase.WantImportMap, gotImportMap)
			}

			if (gotErr != nil && testcase.WantErr == nil) ||
				(gotErr == nil && testcase.WantErr != nil) {
				t.Errorf("want error %v, got %v", testcase.WantErr, gotErr)
			}
			if gotErr != nil && testcase.WantErr != nil {
				if gotErr.Error() != testcase.WantErr.Error() {
					t.Errorf("want error %v, got %v", testcase.WantErr, gotErr)
				}
			}
		})
	}
}

func TestLoadImportPackages(t *testing.T) {
	tests := []struct {
		Scenario             string
		FilesPath            string
		WantNumberOfPackages int
		WantImportMap        map[string][]string
		WantErr              error
	}{
		{
			Scenario:             "Test adding package to Bolt database",
			FilesPath:            TEST_SRC_PATH_VALID,
			WantNumberOfPackages: 2,
			WantImportMap: map[string][]string{
				"main":       {"os", "testimport"},
				"testimport": {},
			},
			WantErr: nil,
		},
		{
			Scenario:             "Test adding package to Bolt database with missing package",
			FilesPath:            "test_folder/test_import_package_error/",
			WantNumberOfPackages: 2,
			WantImportMap: map[string][]string{
				"main":       {"os", "testimport"},
				"testimport": {"testimport2"},
			},
			WantErr: errors.New("package testimport2 not found"),
		},
	}

	for _, testcase := range tests {
		t.Run(testcase.Scenario, func(t *testing.T) {
			_, sourceCodes, _, perr := util.ParseArgsForCX([]string{testcase.FilesPath}, true)
			if perr != nil {
				t.Fatal(perr)
			}
			fileMap, err := loader.CreateFileMap(sourceCodes)
			if err != nil {
				t.Error(err)
			}
			gotImportMap := make(map[string][]string)
			testPackageStruct := loader.PackageList{}
			err = loader.AddNewPackage(&testPackageStruct, "main", fileMap, gotImportMap)
			if err != nil {
				t.Error(err)
			}

			gotErr := loader.LoadImportPackages(&testPackageStruct, "main", fileMap, gotImportMap)

			gotNumberOfPackages := len(testPackageStruct.Packages)
			if gotNumberOfPackages != testcase.WantNumberOfPackages {
				t.Errorf("want %d packages, got %d packages", testcase.WantNumberOfPackages, gotNumberOfPackages)
			}

			if fmt.Sprint(gotImportMap) != fmt.Sprint(testcase.WantImportMap) {
				t.Errorf("want import map %v, got %v", testcase.WantImportMap, gotImportMap)
			}

			if (gotErr != nil && testcase.WantErr == nil) ||
				(gotErr == nil && testcase.WantErr != nil) {
				t.Errorf("want error %v, got %v", testcase.WantErr, gotErr)
			}
			if gotErr != nil && testcase.WantErr != nil {
				if gotErr.Error() != testcase.WantErr.Error() {
					t.Errorf("want error %v, got %v", testcase.WantErr, gotErr)
				}
			}
		})
	}
}

func TestCheckForDependencyLoop(t *testing.T) {
	tests := []struct {
		Scenario  string
		ImportMap map[string][]string
		WantErr   error
	}{
		{
			Scenario: "Testing with a program with no dependency loop",
			ImportMap: map[string][]string{
				"main": {"testimport"},
			},
			WantErr: nil,
		},
		{
			Scenario: "Test with a program that contains a self dependency loop",
			ImportMap: map[string][]string{
				"main": {"main"},
			},
			WantErr: errors.New("Module main imports itself"),
		},
		{
			Scenario: "Test with a program that contains a dependency loop between modules",
			ImportMap: map[string][]string{
				"main":       {"testimport"},
				"testimport": {"main"},
			},
			WantErr: errors.New("Dependency loop between modules main -> testimport -> main"),
		},
		{
			Scenario: "Test with a program that contains a loop of three modules",
			ImportMap: map[string][]string{
				"main": {"a"},
				"a":    {"b"},
				"b":    {"c"},
				"c":    {"a"},
			},
			WantErr: errors.New("Dependency loop between modules a -> b -> c -> a"),
		},
		{
			Scenario: "Testing a diamond, which is not a loop",
			ImportMap: map[string][]string{
				"main": {"a", "b"},
				"a":    {"c"},
				"b":    {"c"},
				"c":    {"os"},
			},
			WantErr: nil,
		},
	}

	for _, testcase := range tests {
		t.Run(testcase.Scenario, func(t *testing.T) {
			gotErr := loader.CheckForDependencyLoop(testcase.ImportMap)

			if (gotErr != nil && testcase.WantErr == nil) ||
				(gotErr == nil && testcase.WantErr != nil) {
				t.Errorf("want error %v, got %v", testcase.WantErr, gotErr)
			}

			if gotErr != nil && testcase.WantErr != nil {
				if gotErr.Error() != testcase.WantErr.Error() {
					t.Errorf("want error %v, got %v", testcase.WantErr, gotErr)
				}
			}
		})
	}
}

func TestLoadCXProgram(t *testing.T) {
	tests := []struct {
		Scenario         string
		FilesPath        string
		ExpectedPackages int
		WantErr          bool
	}{
		{
			Scenario:         "valid program stores main and testimport",
			FilesPath:        TEST_SRC_PATH_VALID,
			ExpectedPackages: 2,
		},
		{
			Scenario:  "file with the wrong package is an error",
			FilesPath: TEST_SRC_PATH_INVALID,
			WantErr:   true,
		},
	}
	for _, testcase := range tests {
		t.Run(testcase.Scenario, func(t *testing.T) {
			_, sourceCodes, _, perr := util.ParseArgsForCX([]string{testcase.FilesPath}, true)
			if perr != nil {
				t.Fatal(perr)
			}
			err := loader.LoadCXProgram("test", sourceCodes)
			if testcase.WantErr {
				if err == nil {
					t.Error("want an error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			value, err := bolt.Get("test")
			if err != nil {
				t.Fatal(err)
			}
			var packageList loader.PackageList
			if err := packageList.UnmarshalBinary(value); err != nil {
				t.Fatal(err)
			}
			if len(packageList.Packages) != testcase.ExpectedPackages {
				t.Errorf("got %d packages, want %d", len(packageList.Packages), testcase.ExpectedPackages)
			}
		})
	}
}

func TestParseArgsForCXMissingPath(t *testing.T) {
	if _, _, _, err := util.ParseArgsForCX([]string{"test_folder/does_not_exist"}, true); err == nil {
		t.Error("want an error for a missing path, got nil")
	}
}

func TestLoadCXProgramImportLoop(t *testing.T) {
	_, sourceCodes, _, err := util.ParseArgsForCX([]string{TEST_SRC_PATH_LOOP}, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := loader.LoadCXProgram("loop", sourceCodes); err == nil {
		t.Error("want a dependency loop error, got nil")
	}
}

func TestBlake2HashSurvivesRoundTrip(t *testing.T) {
	file, err := os.Open(TEST_PACKAGE_FILE)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	fileStruct, err := loader.FileStructFromFile(file)
	if err != nil {
		t.Fatal(err)
	}
	data, err := fileStruct.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var got loader.File
	if err := got.UnmarshalBinary(data); err != nil {
		t.Fatal(err)
	}
	if got.Blake2Hash != fileStruct.Blake2Hash {
		t.Errorf("Blake2Hash changed in the round trip: stored %d bytes, read back %d bytes", len(fileStruct.Blake2Hash), len(got.Blake2Hash))
	}
	content, err := os.ReadFile(TEST_PACKAGE_FILE)
	if err != nil {
		t.Fatal(err)
	}
	if want := fmt.Sprintf("%x", blake2b.Sum512(content)); got.Blake2Hash != want {
		t.Errorf("Blake2Hash = %q, want the hex hash of the content %q", got.Blake2Hash, want)
	}
}

func TestLoadCXProgramCommentedImport(t *testing.T) {
	_, sourceCodes, _, err := util.ParseArgsForCX([]string{"test_folder/test_comment_import/src"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := loader.LoadCXProgram("commentimport", sourceCodes); err != nil {
		t.Errorf("an import inside a comment was read as an import: %v", err)
	}
}
