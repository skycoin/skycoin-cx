package partialparsing

import (
	"strings"
	"testing"
)

func TestLexTokenLines(t *testing.T) {
	src := "package main\n\nfunc main() {\n\tvar x i32 = 5\n\tx = 6\n}\n"
	want := []struct {
		yys  int
		line int
	}{
		{PACKAGE, 1}, {IDENTIFIER, 1},
		{FUNC, 3}, {IDENTIFIER, 3},
		{VAR, 4}, {IDENTIFIER, 4}, {I32, 4}, {INT_LITERAL, 4},
		{IDENTIFIER, 5}, {INT_LITERAL, 5},
	}

	lx := NewLexer(strings.NewReader(src))
	var got []yySymType
	for {
		var lval yySymType
		if lx.Lex(&lval) <= 0 {
			break
		}
		got = append(got, lval)
	}

	i := 0
	for _, w := range want {
		for i < len(got) && got[i].yys != w.yys {
			i++
		}
		if i == len(got) {
			t.Fatalf("token %d (line %d) not found", w.yys, w.line)
		}
		if got[i].line != w.line {
			t.Errorf("token %d: line %d, want %d", w.yys, got[i].line, w.line)
		}
		i++
	}
}
