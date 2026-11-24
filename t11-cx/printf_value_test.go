package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// printf %v of arrays and slices goes through ast.AppendPrintableValue. See #238.
func TestPrintfArraysAndSlices(t *testing.T) {
	file := filepath.Join(t.TempDir(), "p.cx")
	src := `package main

func main() {
	var a [3]i32
	a[0] = 1
	a[1] = 2
	a[2] = 3
	printf("%v\n", a)
	var m [2][2]i32
	m[1][1] = 4
	printf("%v\n", m)
	var s []i32
	s = append(s, 8)
	s = append(s, 9)
	printf("%v\n", s)
	var ss []str
	ss = append(ss, "hi")
	ss = append(ss, "there")
	printf("%v\n", ss)
}
`
	if err := os.WriteFile(file, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(os.Args[0], file)
	cmd.Env = append(os.Environ(), "CX_TEST_RUN_MAIN=1")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("cx: %v\n%s", err, out)
	}
	want := "[1, 2, 3]\n[[0 0] [0 4]]\n[8, 9]\n[hi, there]\n"
	if string(out) != want {
		t.Errorf("stdout:\n got %q\nwant %q", out, want)
	}
}
