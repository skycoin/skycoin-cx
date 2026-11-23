// Command t0-cx-tools-names writes one CSV row per declared name in the Go
// and CX sources of one or more repositories, for the naming review of
// AGENTS.md. Usage: go run ./t0-cx-tools-names [-out file.csv] repo...
package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// NameRow is one declared name and where it is declared.
type NameRow struct {
	Repository string
	Language   string
	Package    string
	File       string
	Line       int
	Column     int
	Name       string
	Kind       string
	Parent     string
	Exported   bool
	TestFile   bool
}

// NameRowHeader is the CSV header, in NameRow field order.
var NameRowHeader = []string{"repository", "language", "module_or_package", "file", "line", "column", "name", "kind", "parent", "exported", "test_file"}

func main() {
	outPath := flag.String("out", "", "CSV file to write, standard output when empty")
	flag.Parse()
	if flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "usage: t0-cx-tools-names [-out file.csv] repository...")
		os.Exit(2)
	}
	var rows []NameRow
	for _, repositoryPath := range flag.Args() {
		repositoryRows, err := RepositoryNames(repositoryPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		rows = append(rows, repositoryRows...)
	}
	NameRowsSort(rows)
	output := io.Writer(os.Stdout)
	if *outPath != "" {
		file, err := os.Create(*outPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		defer file.Close()
		output = file
	}
	if err := NameRowsWrite(output, rows); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// RepositoryNames walks one repository and returns the names of its Go and
// CX files. Hidden directories, vendor, testdata and node_modules are skipped.
func RepositoryNames(repositoryPath string) ([]NameRow, error) {
	absolutePath, err := filepath.Abs(repositoryPath)
	if err != nil {
		return nil, err
	}
	repository := filepath.Base(absolutePath)
	goFiles := map[string][]string{}
	var rows []NameRow
	err = filepath.WalkDir(absolutePath, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		base := entry.Name()
		if entry.IsDir() {
			if path != absolutePath && (strings.HasPrefix(base, ".") || base == "vendor" || base == "testdata" || base == "node_modules") {
				return filepath.SkipDir
			}
			return nil
		}
		relativePath, err := filepath.Rel(absolutePath, path)
		if err != nil {
			return err
		}
		relativePath = filepath.ToSlash(relativePath)
		switch filepath.Ext(base) {
		case ".go":
			directory := filepath.Dir(relativePath)
			goFiles[directory] = append(goFiles[directory], relativePath)
		case ".cx":
			source, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			rows = append(rows, CXFileNames(repository, relativePath, source)...)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	for _, files := range goFiles {
		directoryRows, err := GoDirectoryNames(repository, absolutePath, files)
		if err != nil {
			return nil, err
		}
		rows = append(rows, directoryRows...)
	}
	opcodeRows, err := OpcodeNames(repository, absolutePath)
	if err != nil {
		return nil, err
	}
	rows = append(rows, opcodeRows...)
	return rows, nil
}

// NameRowsSort orders rows by repository, file, line, column, then kind and
// name so the output is stable.
func NameRowsSort(rows []NameRow) {
	sort.Slice(rows, func(leftIndex, rightIndex int) bool {
		left, right := rows[leftIndex], rows[rightIndex]
		if left.Repository != right.Repository {
			return left.Repository < right.Repository
		}
		if left.File != right.File {
			return left.File < right.File
		}
		if left.Line != right.Line {
			return left.Line < right.Line
		}
		if left.Column != right.Column {
			return left.Column < right.Column
		}
		if left.Kind != right.Kind {
			return left.Kind < right.Kind
		}
		return left.Name < right.Name
	})
}

// NameRowsWrite writes the header and rows as RFC 4180 CSV.
func NameRowsWrite(output io.Writer, rows []NameRow) error {
	writer := csv.NewWriter(output)
	writer.UseCRLF = true
	if err := writer.Write(NameRowHeader); err != nil {
		return err
	}
	for _, row := range rows {
		record := []string{row.Repository, row.Language, row.Package, row.File, strconv.Itoa(row.Line), strconv.Itoa(row.Column), row.Name, row.Kind, row.Parent, YesNo(row.Exported), YesNo(row.TestFile)}
		if err := writer.Write(record); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}

// YesNo spells a bool as the CSV does.
func YesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}
