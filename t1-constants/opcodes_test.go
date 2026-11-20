package constants

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
)

const opcodesCSVPath = "../docs/opcodes.csv"

func TestOpcodeLifecycleTableConsistent(t *testing.T) {
	live := map[string]int{}
	byNumber := map[int][]OpcodeLifecycle{}
	previous := 0
	for _, row := range OpcodeLifecycleTable {
		if row.Number <= 0 || row.Number < previous || row.Name == "" {
			t.Errorf("row %+v: numbers must be positive and in order", row)
		}
		previous = row.Number
		byNumber[row.Number] = append(byNumber[row.Number], row)
		if row.VersionAdded != 0 && (!LanguageVersionValid(row.VersionAdded) || row.VersionAdded > LanguageVersionCurrent) {
			t.Errorf("row %+v: bad version added", row)
		}
		if row.VersionDeprecated == 0 {
			if row.VersionReassignable != 0 {
				t.Errorf("row %+v: live row with a reassignable version", row)
			}
			if other, found := live[row.Name]; found {
				t.Errorf("row %+v: name is also live on %d", row, other)
			}
			live[row.Name] = row.Number
			continue
		}
		if !LanguageVersionValid(row.VersionDeprecated) || row.VersionDeprecated > LanguageVersionCurrent || row.VersionDeprecated < row.VersionAdded {
			t.Errorf("row %+v: bad version deprecated", row)
		}
		if !LanguageVersionValid(row.VersionReassignable) || row.VersionReassignable < row.VersionDeprecated {
			t.Errorf("row %+v: bad version reassignable", row)
		}
	}
	for number, rows := range byNumber {
		for i := 0; i < len(rows)-1; i++ {
			if rows[i].VersionDeprecated == 0 {
				t.Errorf("number %d: live row %q is followed by another row", number, rows[i].Name)
			} else if rows[i+1].VersionAdded < rows[i].VersionReassignable {
				t.Errorf("number %d: %q is assigned in %d, inside the window of %q until %d", number, rows[i+1].Name, rows[i+1].VersionAdded, rows[i].Name, rows[i].VersionReassignable)
			}
		}
	}
}

func opcodeLifecycleTableCSV() string {
	var b strings.Builder
	b.WriteString("number,name,version_assigned,version_deprecated,version_reassignment_allowed\n")
	for _, row := range OpcodeLifecycleTable {
		fmt.Fprintf(&b, "%d,%s,%d,%d,%d\n", row.Number, row.Name, row.VersionAdded, row.VersionDeprecated, row.VersionReassignable)
	}
	return b.String()
}

func TestOpcodeLifecycleTableMatchesCSV(t *testing.T) {
	want := opcodeLifecycleTableCSV()
	if os.Getenv("CX_UPDATE_OPCODES_CSV") == "1" {
		if err := os.WriteFile(opcodesCSVPath, []byte(want), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := os.ReadFile(opcodesCSVPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.ReplaceAll(string(got), "\r\n", "\n") != want {
		t.Errorf("%s differs from OpcodeLifecycleTable, run CX_UPDATE_OPCODES_CSV=1 go test ./t1-constants", opcodesCSVPath)
	}
}

func TestOpcodeLifecycleHistoryMatchesTable(t *testing.T) {
	var want []string
	for _, row := range OpcodeLifecycleTable {
		if row.VersionAdded != 0 {
			want = append(want, fmt.Sprintf("%010d 0 assigned %d %s", row.VersionAdded, row.Number, row.Name))
		}
		if row.VersionDeprecated != 0 {
			want = append(want, fmt.Sprintf("%010d 1 deprecated %d %s", row.VersionDeprecated, row.Number, row.Name))
			want = append(want, fmt.Sprintf("%010d 2 reassignable %d %s", row.VersionReassignable, row.Number, row.Name))
		}
	}
	sort.Strings(want)
	order := map[string]string{"assigned": "0", "deprecated": "1", "reassignable": "2"}
	f, err := os.Open("opcodes_history.go")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var got []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(strings.TrimPrefix(scanner.Text(), "//"))
		if len(fields) == 4 && len(fields[0]) == 10 && order[fields[1]] != "" && fields[0] != "0000000000" {
			got = append(got, fmt.Sprintf("%s %s %s %s %s", fields[0], order[fields[1]], fields[1], fields[2], fields[3]))
		}
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("opcodes_history.go events differ from OpcodeLifecycleTable, want\n%s", strings.Join(want, "\n"))
	}
}

func TestLanguageVersionReassignable(t *testing.T) {
	for deprecated, want := range map[int64]int64{
		2611200001: 2711200001,
		2611209999: 2711200001,
		2802290003: 2903010001,
		2702280001: 2802280001,
		2712310002: 2812310001,
	} {
		if got := LanguageVersionReassignable(deprecated); got != want {
			t.Errorf("LanguageVersionReassignable(%d) = %d, want %d", deprecated, got, want)
		}
	}
}

func TestLanguageVersionValid(t *testing.T) {
	for version, want := range map[int64]bool{
		LanguageVersionCurrent: true,
		2802290001:             true,
		2702290001:             false,
		2611200000:             false,
		2613010001:             false,
		2611310001:             false,
		0:                      false,
	} {
		if got := LanguageVersionValid(version); got != want {
			t.Errorf("LanguageVersionValid(%d) = %v, want %v", version, got, want)
		}
	}
}
