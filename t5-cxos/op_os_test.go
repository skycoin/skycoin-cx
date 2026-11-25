//go:build !nocxos

package cxos

import (
	"reflect"
	"testing"
)

// os.Run split the command on single spaces. See #118.
func TestSplitCommand(t *testing.T) {
	cases := []struct {
		in   string
		want []string
		ok   bool
	}{
		{"", nil, true},
		{"   ", nil, true},
		{"echo a", []string{"echo", "a"}, true},
		{"echo  a\tb", []string{"echo", "a", "b"}, true},
		{"sh -c 'echo hi'", []string{"sh", "-c", "echo hi"}, true},
		{`printf "%s|" "a b" c`, []string{"printf", "%s|", "a b", "c"}, true},
		{`echo ""`, []string{"echo", ""}, true},
		{`echo a'b c'd`, []string{"echo", "ab cd"}, true},
		{`C:\bin\tool.exe x`, []string{`C:\bin\tool.exe`, "x"}, true},
		{"echo 'a", nil, false},
	}
	for _, c := range cases {
		got, err := splitCommand(c.in)
		if (err == nil) != c.ok {
			t.Errorf("splitCommand(%q) error %v, want ok=%v", c.in, err, c.ok)
			continue
		}
		if c.ok && !reflect.DeepEqual(got, c.want) {
			t.Errorf("splitCommand(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
