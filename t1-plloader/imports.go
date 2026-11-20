package loader

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
)

// Imports returns the paths of the import statements in CX source, in order
// and each once. Comments and string literals are skipped. It is not a syntax
// check, an unterminated comment or string only ends the scan.
func Imports(src []byte) ([]string, error) {
	var imports []string
	afterImport := false
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
				if afterImport {
					return nil, errors.New("unterminated import path")
				}
				return imports, nil
			}
			i++
			if afterImport {
				path, err := strconv.Unquote(string(src[start:i]))
				if err != nil {
					return nil, fmt.Errorf("import path %s: %w", src[start:i], err)
				}
				if !Contains(imports, path) {
					imports = append(imports, path)
				}
				afterImport = false
			}
			continue
		case c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9':
			for i < len(src) && (src[i] == '_' || src[i] >= 'a' && src[i] <= 'z' || src[i] >= 'A' && src[i] <= 'Z' || src[i] >= '0' && src[i] <= '9') {
				i++
			}
		default:
			i++
		}
		if afterImport {
			return nil, fmt.Errorf("import not followed by a quoted path, found %q", src[start:i])
		}
		afterImport = string(src[start:i]) == "import"
	}
	if afterImport {
		return nil, errors.New("import at the end of the file")
	}
	return imports, nil
}
