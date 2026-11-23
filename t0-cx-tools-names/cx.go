package main

import (
	"path"
	"strings"
)

// CXTokenKind is the lexical class of a CX token.
type CXTokenKind int

// The CX token kinds. CXTokenSemicolon covers both written and newline
// inserted semicolons, as in Go.
const (
	CXTokenEnd CXTokenKind = iota
	CXTokenIdent
	CXTokenNumber
	CXTokenString
	CXTokenPunctuation
	CXTokenSemicolon
)

// CXToken is one token with its 1-based line and byte column.
type CXToken struct {
	Kind   CXTokenKind
	Text   string
	Line   int
	Column int
}

// CXOperators lists the multi-byte operators, longest first.
var CXOperators = []string{"...", "<<=", ">>=", "&^=", ":=", "==", "!=", "<=", ">=", "&&", "||", "++", "--", "+=", "-=", "*=", "/=", "%=", "&=", "|=", "^=", "<<", ">>", "&^"}

// CXTokenize splits CX source into tokens. It never fails, so test inputs
// with deliberate syntax errors still yield their names.
func CXTokenize(source []byte) []CXToken {
	var tokens []CXToken
	line, column := 1, 1
	position := 0
	semicolonNeeded := func() bool {
		if len(tokens) == 0 {
			return false
		}
		last := tokens[len(tokens)-1]
		switch last.Kind {
		case CXTokenIdent, CXTokenNumber, CXTokenString:
			return true
		case CXTokenPunctuation:
			return last.Text == ")" || last.Text == "]" || last.Text == "}" || last.Text == "++" || last.Text == "--"
		}
		return false
	}
	newline := func() {
		if semicolonNeeded() {
			tokens = append(tokens, CXToken{Kind: CXTokenSemicolon, Text: "\n", Line: line, Column: column})
		}
		line++
		column = 1
	}
	advance := func(count int) {
		position += count
		column += count
	}
	for position < len(source) {
		current := source[position]
		switch {
		case current == '\n':
			newline()
			position++
		case current == ' ' || current == '\t' || current == '\r':
			advance(1)
		case strings.HasPrefix(string(source[position:min(position+2, len(source))]), "//"):
			for position < len(source) && source[position] != '\n' {
				advance(1)
			}
		case strings.HasPrefix(string(source[position:min(position+2, len(source))]), "/*"):
			advance(2)
			for position < len(source) && !strings.HasPrefix(string(source[position:min(position+2, len(source))]), "*/") {
				if source[position] == '\n' {
					newline()
					position++
				} else {
					advance(1)
				}
			}
			advance(min(2, len(source)-position))
		case current == '"' || current == '\'' || current == '`':
			startLine, startColumn, start := line, column, position
			advance(1)
			for position < len(source) && source[position] != current {
				if source[position] == '\\' && current != '`' && position+1 < len(source) {
					advance(1)
				}
				if source[position] == '\n' {
					if current != '`' {
						break
					}
					line++
					column = 1
					position++
					continue
				}
				advance(1)
			}
			if position < len(source) && source[position] == current {
				advance(1)
			}
			tokens = append(tokens, CXToken{Kind: CXTokenString, Text: string(source[start:position]), Line: startLine, Column: startColumn})
		case CXIdentByte(current) || current >= 0x80:
			start, startColumn := position, column
			kind := CXTokenIdent
			if current >= '0' && current <= '9' {
				kind = CXTokenNumber
			}
			for position < len(source) && (CXIdentByte(source[position]) || source[position] >= 0x80 || (kind == CXTokenNumber && source[position] == '.')) {
				advance(1)
			}
			tokens = append(tokens, CXToken{Kind: kind, Text: string(source[start:position]), Line: line, Column: startColumn})
		default:
			text := string(current)
			for _, operator := range CXOperators {
				if strings.HasPrefix(string(source[position:min(position+len(operator), len(source))]), operator) {
					text = operator
					break
				}
			}
			kind := CXTokenPunctuation
			if text == ";" {
				kind = CXTokenSemicolon
			}
			tokens = append(tokens, CXToken{Kind: kind, Text: text, Line: line, Column: column})
			advance(len(text))
		}
	}
	if semicolonNeeded() {
		tokens = append(tokens, CXToken{Kind: CXTokenSemicolon, Text: "\n", Line: line, Column: column})
	}
	return tokens
}

// CXIdentByte reports whether an ASCII byte can be part of an identifier.
func CXIdentByte(current byte) bool {
	return current == '_' || (current >= 'a' && current <= 'z') || (current >= 'A' && current <= 'Z') || (current >= '0' && current <= '9')
}

// CXParser walks the tokens of one CX file and records declared names. CX
// has no visibility rule, every package level name can be used from an
// importing package, so all of them are recorded as exported.
type CXParser struct {
	Tokens     []CXToken
	Position   int
	Repository string
	File       string
	Package    string
	TestFile   bool
	Rows       []NameRow
}

// CXFileNames returns the names declared in one CX file.
func CXFileNames(repository, file string, source []byte) []NameRow {
	parser := &CXParser{Tokens: CXTokenize(source), Repository: repository, File: file, TestFile: CXTestFile(file)}
	parser.FileParse()
	return parser.Rows
}

// CXTestFile reports whether a CX file is a test program or a test fixture.
func CXTestFile(file string) bool {
	for _, segment := range strings.Split(path.Dir(file), "/") {
		switch segment {
		case "tests", "test", "test_files", "test_folder", "test_tree", "test-workspace":
			return true
		}
	}
	base := path.Base(file)
	return strings.HasPrefix(base, "test-") || strings.HasPrefix(base, "test_") || strings.HasSuffix(base, "_test.cx")
}

// Peek returns the token offset places ahead, or an end token.
func (p *CXParser) Peek(offset int) CXToken {
	if p.Position+offset >= len(p.Tokens) {
		return CXToken{Kind: CXTokenEnd}
	}
	return p.Tokens[p.Position+offset]
}

// Next consumes and returns the current token.
func (p *CXParser) Next() CXToken {
	token := p.Peek(0)
	if token.Kind != CXTokenEnd {
		p.Position++
	}
	return token
}

// Add records a name at a token.
func (p *CXParser) Add(token CXToken, kind, parent string, exported bool) {
	if token.Text == "_" || token.Text == "" {
		return
	}
	p.Rows = append(p.Rows, NameRow{Repository: p.Repository, Language: "cx", Package: p.Package, File: p.File, Line: token.Line, Column: token.Column, Name: token.Text, Kind: kind, Parent: parent, Exported: exported, TestFile: p.TestFile})
}

// SkipStatement consumes tokens up to and including the next semicolon at
// nesting depth zero. It stops before an unmatched closing bracket.
func (p *CXParser) SkipStatement() {
	depth := 0
	for {
		token := p.Peek(0)
		switch {
		case token.Kind == CXTokenEnd:
			return
		case token.Kind == CXTokenSemicolon && depth == 0:
			p.Next()
			return
		case token.Text == "(" || token.Text == "[" || token.Text == "{":
			depth++
		case token.Text == ")" || token.Text == "]" || token.Text == "}":
			if depth == 0 {
				return
			}
			depth--
		}
		p.Next()
	}
}

// FileParse records the package level declarations of the file.
func (p *CXParser) FileParse() {
	for p.Peek(0).Kind != CXTokenEnd {
		token := p.Peek(0)
		switch {
		case token.Kind == CXTokenIdent && token.Text == "package":
			p.Next()
			if p.Peek(0).Kind == CXTokenIdent {
				p.Package = p.Peek(0).Text
				p.Add(p.Next(), "package", "", false)
			}
			p.SkipStatement()
		case token.Kind == CXTokenIdent && token.Text == "import":
			p.Next()
			p.ImportParse()
		case token.Kind == CXTokenIdent && token.Text == "type":
			p.TypeParse("")
		case token.Kind == CXTokenIdent && token.Text == "func":
			p.FuncParse()
		case token.Kind == CXTokenIdent && token.Text == "var":
			p.VarParse("global_variable", "")
		case token.Kind == CXTokenIdent && token.Text == "const":
			p.VarParse("constant", "")
		case token.Text == ")" || token.Text == "]" || token.Text == "}":
			p.Next()
		default:
			p.SkipStatement()
		}
	}
}

// ImportParse records one import path or a parenthesized group of them.
func (p *CXParser) ImportParse() {
	if p.Peek(0).Text != "(" {
		if p.Peek(0).Kind == CXTokenString {
			p.ImportAdd(p.Next())
		}
		p.SkipStatement()
		return
	}
	p.Next()
	for p.Peek(0).Kind != CXTokenEnd && p.Peek(0).Text != ")" {
		if p.Peek(0).Kind == CXTokenString {
			p.ImportAdd(p.Peek(0))
		}
		p.Next()
	}
	p.Next()
	p.SkipStatement()
}

// ImportAdd records an import path without its quotes.
func (p *CXParser) ImportAdd(token CXToken) {
	token.Text = strings.Trim(token.Text, "\"`")
	p.Add(token, "import", "", false)
}

// TypeParse records a type declaration and its fields. parent is the
// enclosing function, empty at package level.
func (p *CXParser) TypeParse(parent string) {
	p.Next()
	if p.Peek(0).Kind != CXTokenIdent {
		p.SkipStatement()
		return
	}
	name := p.Next()
	owner := name.Text
	if parent != "" {
		owner = parent + "." + owner
	}
	switch p.Peek(0).Text {
	case "struct":
		p.Add(name, "struct", parent, parent == "")
		p.Next()
		if p.Peek(0).Text == "{" {
			p.StructFieldsParse(owner, parent == "")
		}
	case "interface":
		p.Add(name, "interface", parent, parent == "")
	default:
		p.Add(name, "type", parent, parent == "")
	}
	p.SkipStatement()
}

// StructFieldsParse records the fields of a struct body, consuming its
// braces. A line with a lone identifier is an embedded type.
func (p *CXParser) StructFieldsParse(owner string, exported bool) {
	p.Next()
	for p.Peek(0).Kind != CXTokenEnd && p.Peek(0).Text != "}" {
		if p.Peek(0).Kind == CXTokenSemicolon {
			p.Next()
			continue
		}
		names := p.IdentListParse()
		next := p.Peek(0)
		if len(names) == 1 && (next.Kind == CXTokenSemicolon || next.Text == "}") {
			p.Add(names[0], "embedded_field", owner, exported)
		} else {
			for _, name := range names {
				p.Add(name, "field", owner, exported)
			}
		}
		if len(names) == 0 && next.Kind != CXTokenSemicolon {
			p.Next()
		}
		p.SkipStatement()
	}
	p.Next()
}

// IdentListParse consumes identifiers separated by commas.
func (p *CXParser) IdentListParse() []CXToken {
	var names []CXToken
	for p.Peek(0).Kind == CXTokenIdent {
		names = append(names, p.Next())
		if p.Peek(0).Text != "," || p.Peek(1).Kind != CXTokenIdent {
			break
		}
		p.Next()
	}
	return names
}

// VarParse records a var or const declaration, single or grouped.
func (p *CXParser) VarParse(kind, parent string) {
	p.Next()
	if p.Peek(0).Text != "(" {
		p.VarSpecParse(kind, parent)
		return
	}
	p.Next()
	for p.Peek(0).Kind != CXTokenEnd && p.Peek(0).Text != ")" {
		if p.Peek(0).Kind == CXTokenSemicolon {
			p.Next()
			continue
		}
		start := p.Position
		p.VarSpecParse(kind, parent)
		if p.Position == start {
			p.Next()
		}
	}
	p.Next()
	p.SkipStatement()
}

// VarSpecParse records the names of one var or const spec.
func (p *CXParser) VarSpecParse(kind, parent string) {
	names := p.IdentListParse()
	if len(names) == 1 && (p.Peek(0).Text == "func" || (p.Peek(0).Text == "=" && p.Peek(1).Text == "func")) {
		kind = "function_variable"
	}
	for _, name := range names {
		p.Add(name, kind, parent, parent == "")
	}
	p.SkipStatement()
}

// FuncParse records a package level function or method, its receiver,
// parameters, named results and body.
func (p *CXParser) FuncParse() {
	p.Next()
	owner := ""
	receiverType := ""
	var receiver []CXToken
	if p.Peek(0).Text == "(" {
		receiver = p.ParenthesesParse()
		for i := len(receiver) - 1; i >= 0; i-- {
			if receiver[i].Kind == CXTokenIdent {
				receiverType = receiver[i].Text
				break
			}
		}
	}
	if p.Peek(0).Kind != CXTokenIdent {
		p.SkipStatement()
		return
	}
	name := p.Next()
	owner = name.Text
	if receiverType != "" {
		p.Add(name, "method", receiverType, true)
		owner = receiverType + "." + name.Text
		if len(receiver) >= 2 && receiver[0].Kind == CXTokenIdent && receiver[1].Text != "." {
			p.Add(receiver[0], "receiver", owner, false)
		}
	} else {
		p.Add(name, "function", "", true)
	}
	if p.SignatureParse(owner) {
		p.BodyParse(owner)
	}
	p.SkipStatement()
}

// ParenthesesParse consumes a parenthesized list and returns its tokens
// without the outer parentheses.
func (p *CXParser) ParenthesesParse() []CXToken {
	var inner []CXToken
	p.Next()
	depth := 0
	for p.Peek(0).Kind != CXTokenEnd {
		token := p.Next()
		switch token.Text {
		case "(", "[", "{":
			depth++
		case ")", "]", "}":
			if depth == 0 {
				return inner
			}
			depth--
		}
		inner = append(inner, token)
	}
	return inner
}

// SignatureParse records parameters and named results, then consumes the
// result types. It reports whether a body brace follows.
func (p *CXParser) SignatureParse(owner string) bool {
	if p.Peek(0).Text == "(" {
		p.ParameterListAdd(p.ParenthesesParse(), "parameter", owner)
	}
	if p.Peek(0).Text == "(" {
		p.ParameterListAdd(p.ParenthesesParse(), "result", owner)
	}
	depth := 0
	for {
		token := p.Peek(0)
		switch {
		case token.Kind == CXTokenEnd:
			return false
		case token.Text == "{" && depth == 0:
			return true
		case token.Kind == CXTokenSemicolon && depth == 0:
			return false
		case token.Text == "(" || token.Text == "[":
			depth++
		case token.Text == ")" || token.Text == "]" || token.Text == "," || token.Text == "}":
			if depth == 0 {
				return false
			}
			if token.Text != "," {
				depth--
			}
		}
		p.Next()
	}
}

// ParameterListAdd records the names of a parameter list. As in Go, the
// list is named when any comma separated group is a name and a type.
func (p *CXParser) ParameterListAdd(inner []CXToken, kind, owner string) {
	var groups [][]CXToken
	var group []CXToken
	depth := 0
	for _, token := range inner {
		switch token.Text {
		case "(", "[", "{":
			depth++
		case ")", "]", "}":
			depth--
		}
		if token.Kind == CXTokenSemicolon {
			continue
		}
		if token.Text == "," && depth == 0 {
			groups = append(groups, group)
			group = nil
			continue
		}
		group = append(group, token)
	}
	if len(group) > 0 {
		groups = append(groups, group)
	}
	named := false
	for _, group := range groups {
		if len(group) >= 2 && group[0].Kind == CXTokenIdent && group[1].Text != "." {
			named = true
		}
	}
	if !named {
		return
	}
	for _, group := range groups {
		if len(group) > 0 && group[0].Kind == CXTokenIdent {
			p.Add(group[0], kind, owner, false)
		}
	}
}

// CXControlKeywords open a statement whose header ends at a block brace.
var CXControlKeywords = map[string]bool{"if": true, "else": true, "for": true, "switch": true, "select": true}

// BodyParse records the locals, labels, closures and local declarations of
// a function body, consuming it from its opening brace to its closing one.
func (p *CXParser) BodyParse(owner string) {
	p.Next()
	blocks := []bool{true}
	statementStart := true
	header := false
	loopHeader := false
	previous := CXToken{Kind: CXTokenPunctuation, Text: "{"}
	for len(blocks) > 0 && p.Peek(0).Kind != CXTokenEnd {
		token := p.Peek(0)
		if statementStart && blocks[len(blocks)-1] && token.Kind == CXTokenIdent {
			statementStart = false
			if p.StatementParse(owner, loopHeader) {
				statementStart = true
				previous = CXToken{Kind: CXTokenSemicolon}
				continue
			}
			token = p.Peek(0)
			if CXControlKeywords[token.Text] {
				header = true
				loopHeader = loopHeader || token.Text == "for"
				p.Next()
				statementStart = true
				previous = token
				continue
			}
		}
		switch {
		case token.Text == "{":
			block := header || previous.Kind == CXTokenSemicolon || previous.Text == ")" || previous.Text == "{" || previous.Text == "}" || previous.Text == ":"
			if previous.Kind == CXTokenIdent && p.Position >= 2 && p.Tokens[p.Position-2].Text == "]" {
				block = false
			}
			if previous.Kind == CXTokenIdent && previous.Text == "else" {
				block = true
			}
			blocks = append(blocks, block)
			if block {
				header = false
				loopHeader = false
				statementStart = true
			}
		case token.Text == "}":
			blocks = blocks[:len(blocks)-1]
			statementStart = len(blocks) > 0 && blocks[len(blocks)-1]
		case token.Kind == CXTokenSemicolon:
			statementStart = blocks[len(blocks)-1]
		case token.Kind == CXTokenIdent && token.Text == "func":
			p.Next()
			if p.SignatureParse(owner) {
				previous = CXToken{Kind: CXTokenPunctuation, Text: ")"}
				continue
			}
			previous = token
			continue
		}
		previous = p.Next()
	}
}

// StatementParse records the names a statement declares at its start and
// reports whether it consumed the statement. In a for header a short
// variable declaration declares loop variables.
func (p *CXParser) StatementParse(owner string, loopHeader bool) bool {
	token := p.Peek(0)
	switch token.Text {
	case "var":
		p.VarParse("local_variable", owner)
		return true
	case "const":
		p.VarParse("local_constant", owner)
		return true
	case "type":
		p.TypeParse(owner)
		return true
	case "case", "default":
		depth := 0
		for done := false; !done && p.Peek(0).Kind != CXTokenEnd; {
			current := p.Next()
			switch current.Text {
			case "(", "[", "{":
				depth++
			case ")", "]", "}":
				depth--
			}
			done = current.Text == ":" && depth == 0
		}
		return true
	}
	if p.Peek(1).Text == ":" {
		p.Add(p.Next(), "label", owner, false)
		p.Next()
		return true
	}
	start := p.Position
	names := p.IdentListParse()
	if p.Peek(0).Text != ":=" {
		p.Position = start
		return false
	}
	p.Next()
	kind := "local_variable"
	if len(names) == 1 && p.Peek(0).Text == "func" {
		kind = "function_variable"
	} else if loopHeader {
		kind = "loop_variable"
	}
	for _, name := range names {
		p.Add(name, kind, owner, false)
	}
	return false
}
