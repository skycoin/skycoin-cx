package main

import (
	"encoding/csv"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// objmodelgenSiteSteps are the steps of the Phases 4 and 5 table of
// docs/ObjectModel.md, contained objects first.
var objmodelgenSiteSteps = [][]string{
	{"CXArgumentDebug", "CXLine", "CXTypeSignature_Array", "CXTypeSignature_Struct"},
	{"CXArgument", "CXTypeSignature"},
	{"CXStruct"},
	{"CXAtomicOperator", "CXExpression"},
	{"CXFunction", "CXNativeFunction"},
	{"CXPackage", "CXCall", "CXProgram"},
}

// objmodelgenSiteCategories are the columns of the counts table, in order.
var objmodelgenSiteCategories = []string{
	"ctor-def", "ctor-call", "arena-add", "arena-append", "alloc-literal-heap",
	"alloc-literal-value", "alloc-copy", "alloc-var", "make-slice", "release-call",
	"drop", "get-call", "get-addr-arena", "ptr-field", "ptr-global", "value-field",
}

// objmodelgenSiteCounts counts the rows of the sites CSV by object and
// category, and by object and phase. A row with no phase needs no change.
type objmodelgenSiteCounts struct {
	category map[string]map[string]int
	phase4   map[string]int
	phase5   map[string]int
	total    int
}

func objmodelgenSitesRead(path string) (*objmodelgenSiteCounts, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	counts := &objmodelgenSiteCounts{category: map[string]map[string]int{}, phase4: map[string]int{}, phase5: map[string]int{}}
	for _, row := range rows[1:] {
		object, category, phase := row[1], row[2], row[8]
		switch phase {
		case "4-alloc":
			counts.phase4[object]++
		case "5-gid":
			counts.phase5[object]++
		default:
			continue
		}
		if counts.category[object] == nil {
			counts.category[object] = map[string]int{}
		}
		counts.category[object][category]++
		counts.total++
	}
	return counts, nil
}

// objmodelgenSitesDoc is the object model document with its site counts,
// the total, the Phases 4 and 5 table and the counts table, taken from
// the sites CSV.
func objmodelgenSitesDoc(docPath string, csvPath string) ([]byte, error) {
	data, err := os.ReadFile(docPath)
	if err != nil {
		return nil, err
	}
	counts, err := objmodelgenSitesRead(csvPath)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	var steps, table []string
	steps = append(steps, "| Step | Objects (phase 4, phase 5 sites each) | Phase 4 sites | Phase 5 sites |", "| --- | --- | --- | --- |")
	for i, step := range objmodelgenSiteSteps {
		var objects []string
		p4, p5 := 0, 0
		for _, object := range step {
			objects = append(objects, fmt.Sprintf("`%s` (%d, %d)", object, counts.phase4[object], counts.phase5[object]))
			p4 += counts.phase4[object]
			p5 += counts.phase5[object]
		}
		steps = append(steps, fmt.Sprintf("| %d | %s | %d | %d |", i+1, strings.Join(objects, ", "), p4, p5))
	}
	header := "| Object | " + strings.Join(objmodelgenSiteCategories, " | ") + " | phase 4 | phase 5 | total |"
	table = append(table, header, "|"+strings.Repeat(" --- |", len(objmodelgenSiteCategories)+4))
	sums := make([]int, len(objmodelgenSiteCategories)+3)
	for _, step := range objmodelgenSiteSteps {
		for _, object := range step {
			row := "| `" + object + "` |"
			for i, category := range objmodelgenSiteCategories {
				n := counts.category[object][category]
				sums[i] += n
				row += fmt.Sprintf(" %d |", n)
			}
			p4, p5 := counts.phase4[object], counts.phase5[object]
			sums[len(sums)-3] += p4
			sums[len(sums)-2] += p5
			sums[len(sums)-1] += p4 + p5
			table = append(table, row+fmt.Sprintf(" %d | %d | %d |", p4, p5, p4+p5))
		}
	}
	row := "| total |"
	for _, n := range sums {
		row += fmt.Sprintf(" %d |", n)
	}
	table = append(table, row)
	if sums[len(sums)-1] != counts.total {
		return nil, fmt.Errorf("%s: %d sites belong to no object of the steps", csvPath, counts.total-sums[len(sums)-1])
	}

	var out []string
	replaced := map[string]bool{}
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		switch {
		case strings.HasPrefix(line, "| Step | Objects"):
			out = append(out, steps...)
			i = objmodelgenTableEnd(lines, i)
			replaced["steps"] = true
		case strings.HasPrefix(line, "| Object | ctor-def"):
			out = append(out, table...)
			i = objmodelgenTableEnd(lines, i)
			replaced["counts"] = true
		case strings.HasPrefix(line, "that breaks the standard today, ") && strings.HasSuffix(line, " in all, plus one row for"):
			out = append(out, fmt.Sprintf("that breaks the standard today, %d in all, plus one row for", counts.total))
			replaced["total"] = true
		default:
			out = append(out, line)
		}
	}
	for _, part := range []string{"steps", "counts", "total"} {
		if !replaced[part] {
			return nil, fmt.Errorf("%s: no %s to replace", docPath, part)
		}
	}
	return []byte(strings.Join(out, "\n")), nil
}

// objmodelgenTableEnd is the index of the last row of the table at start.
func objmodelgenTableEnd(lines []string, start int) int {
	i := start
	for i+1 < len(lines) && strings.HasPrefix(lines[i+1], "|") {
		i++
	}
	return i
}

// objmodelgenSiteComment matches the site ids of an OBJMODEL comment, as in
// "site S" and four digits, or "sites S" and several ids.
var objmodelgenSiteComment = regexp.MustCompile(`[Ss]ites?((?: S[0-9]{4})+)`)

// objmodelgenSitesCodeCheck reports the sites CSV and the site comments of
// the Go and grammar files under root that disagree: a row whose id no
// comment names, a comment id with no row, or a row whose file does not name
// its id.
func objmodelgenSitesCodeCheck(root string, csvPath string) ([]string, error) {
	f, err := os.Open(csvPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", csvPath, err)
	}
	commented := map[string]map[string]bool{}
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			switch entry.Name() {
			case "vendor", ".git", ".claude", "bin":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, ".y") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		file := filepath.ToSlash(path)
		for _, line := range strings.Split(string(data), "\n") {
			at := strings.Index(line, "//")
			if at < 0 {
				continue
			}
			for _, match := range objmodelgenSiteComment.FindAllStringSubmatch(line[at:], -1) {
				for _, id := range strings.Fields(match[1]) {
					if commented[id] == nil {
						commented[id] = map[string]bool{}
					}
					commented[id][file] = true
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	var problems []string
	listed := map[string]bool{}
	for _, row := range rows[1:] {
		id, file := row[0], row[3]
		if listed[id] {
			problems = append(problems, fmt.Sprintf("%s: site %s has two rows", csvPath, id))
		}
		listed[id] = true
		switch files := commented[id]; {
		case files == nil:
			problems = append(problems, fmt.Sprintf("%s: site %s is named by no comment, remove the row", csvPath, id))
		case !files[file]:
			problems = append(problems, fmt.Sprintf("%s: site %s is not in %s", csvPath, id, file))
		}
	}
	for id, files := range commented {
		if !listed[id] {
			for file := range files {
				problems = append(problems, fmt.Sprintf("%s: site %s has no row in %s", file, id, csvPath))
			}
		}
	}
	sort.Strings(problems)
	return problems, nil
}
