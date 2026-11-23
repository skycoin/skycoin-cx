package loader

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
)

// importsState is where Imports is in an import statement.
type importsState int

const (
	importsStateNone importsState = iota
	// importsStateKeyword follows import, which a scope word, a period,
	// a name list or the path may follow.
	importsStateKeyword
	// importsStateScope follows the scope word of import file . "p".
	importsStateScope
	// importsStateNames is inside the name list of import (a, b) "p".
	importsStateNames
	// importsStatePath expects the path.
	importsStatePath
)

// Imports returns the paths of the import statements in CX source, in order
// and each once, of the forms import "p", import . "p", import file . "p",
// import (a, b) "p" and import file (a, b) "p". Comments and string literals
// are skipped. It is not a syntax check, an unterminated comment or string
// only ends the scan.
func Imports(src []byte) ([]string, error) {
	var imports []string
	state := importsStateNone
	for i := 0; i < len(src); {
		c := src[i]
		switch {
		case c == '/' && i+1 < len(src) && src[i+1] == '/':
			for i < len(src) && src[i] != '\n' {
				i++
			}
			continue
		case c == '/' && i+1 < len(src) && src[i+1] == '*':
			end := bytes.Index(src[i+2:], []byte("*/"))
			if end < 0 {
				return imports, nil
			}
			i += 2 + end + 2
			continue
		case c == ' ' || c == '\t' || c == '\r' || c == '\n':
			i++
			continue
		}

		start := i
		switch {
		case c == '"' || c == '`':
			i++
			for i < len(src) && src[i] != c {
				if c == '"' && src[i] == '\\' {
					i++
				}
				i++
			}
			if i >= len(src) {
				if state != importsStateNone {
					return nil, errors.New("unterminated import path")
				}
				return imports, nil
			}
			i++
			if state == importsStateNames {
				return nil, fmt.Errorf("import name list holds %s", src[start:i])
			}
			if state != importsStateNone {
				path, err := strconv.Unquote(string(src[start:i]))
				if err != nil {
					return nil, fmt.Errorf("import path %s: %w", src[start:i], err)
				}
				if !Contains(imports, path) {
					imports = append(imports, path)
				}
				state = importsStateNone
			}
			continue
		case c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9':
			for i < len(src) && (src[i] == '_' || src[i] >= 'a' && src[i] <= 'z' || src[i] >= 'A' && src[i] <= 'Z' || src[i] >= '0' && src[i] <= '9') {
				i++
			}
			switch state {
			case importsStateNone:
				if string(src[start:i]) == "import" {
					state = importsStateKeyword
				}
				continue
			case importsStateKeyword:
				state = importsStateScope
				continue
			case importsStateNames:
				continue
			}
		default:
			i++
			switch {
			case c == '.' && (state == importsStateKeyword || state == importsStateScope):
				state = importsStatePath
				continue
			case c == '(' && (state == importsStateKeyword || state == importsStateScope):
				state = importsStateNames
				continue
			case c == ',' && state == importsStateNames:
				continue
			case c == ')' && state == importsStateNames:
				state = importsStatePath
				continue
			case state == importsStateNone:
				continue
			}
		}
		return nil, fmt.Errorf("import not followed by a quoted path, found %q", src[start:i])
	}
	if state != importsStateNone {
		return nil, errors.New("import at the end of the file")
	}
	return imports, nil
}
