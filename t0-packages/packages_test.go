package packages

import "testing"

func TestCheckImport(t *testing.T) {
	for path, ok := range map[string]bool{
		"sys/os":     true,
		"sys/al":     true,
		"gl":         false,
		"al":         false,
		"sys/nosuch": false,
		"sys/":       false,
		"math":       true,
		"math/sys":   true,
	} {
		if err := CheckImport(path); (err == nil) != ok {
			t.Errorf("CheckImport(%q) = %v, want ok %v", path, err, ok)
		}
	}
}

func TestIsBuiltinImport(t *testing.T) {
	for path, want := range map[string]bool{"sys/os": true, "sys/nosuch": true, "os": true, "math": false, "math/os": false} {
		if got := IsBuiltinImport(path); got != want {
			t.Errorf("IsBuiltinImport(%q) = %v, want %v", path, got, want)
		}
	}
}
