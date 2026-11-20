package loader_test

import (
	"reflect"
	"testing"

	"github.com/skycoin/skycoin-cx/t1-plloader"
)

func TestImports(t *testing.T) {
	tests := []struct {
		Scenario string
		Source   string
		Want     []string
		WantErr  bool
	}{
		{"one per line", "package main\nimport \"os\"\nimport \"b\"\n", []string{"os", "b"}, false},
		{"duplicates once", "import \"os\"\nimport \"os\"\n", []string{"os"}, false},
		{"block comment", "/*\nimport \"fromcomment\"\n*/\nimport \"os\"\n", []string{"os"}, false},
		{"line comments", "// import \"a\"\n//import \"b\"\nimport \"os\" // import \"c\"\n", []string{"os"}, false},
		{"string literal", "func main() { str.print(\"import \\\"s\\\"\") }\n", nil, false},
		{"string over two lines", "var s str = \"a\nimport \\\"s\\\"\"\nimport \"os\"\n", []string{"os"}, false},
		{"raw string", "var s str = `import\n\"r\"`\n", nil, false},
		{"several on a line", "import \"a\"; import \"b\";\n", []string{"a", "b"}, false},
		{"no path", "import os\n", nil, true},
		{"unterminated comment", "import \"os\"\n/* import \"a\"\n", []string{"os"}, false},
		{"unterminated string", "import \"os\"\nvar s str = \"import \\\"a\\\"\n", []string{"os"}, false},
		{"unterminated path", "import \"os\n", nil, true},
	}
	for _, tc := range tests {
		t.Run(tc.Scenario, func(t *testing.T) {
			got, err := loader.Imports([]byte(tc.Source))
			if (err != nil) != tc.WantErr {
				t.Fatalf("err = %v, want error %v", err, tc.WantErr)
			}
			if !tc.WantErr && !reflect.DeepEqual(got, tc.Want) {
				t.Errorf("got %q, want %q", got, tc.Want)
			}
		})
	}
}
