package ast

import "testing"

// TestCompileErrorSyntaxText checks the rewrite of goyacc messages.
func TestCompileErrorSyntaxText(t *testing.T) {
	cases := []struct {
		message, text, want string
	}{
		{"syntax error: unexpected COMMA, expecting RBRACK", "", "unexpected comma, expected ]"},
		{"syntax error: unexpected SEMICOLON, expecting COMMA or ASSIGN or CASSIGN", "\n", "unexpected newline, expected comma or = or :="},
		{"syntax error: unexpected SEMICOLON", ";", "unexpected semicolon"},
		{"syntax error: unexpected IDENTIFIER, expecting COMMA or SEMICOLON", "y", "unexpected name y, expected comma or semicolon or newline"},
		{"syntax error: unexpected I32, expecting LPAREN or LBRACE", "", "unexpected i32, expected ( or {"},
		{"syntax error: unexpected INC_OP", "", "unexpected ++"},
		{"syntax error: unexpected $end", "", "unexpected end of file"},
		{"syntax error", "", "unexpected token"},
	}
	for _, tc := range cases {
		if got := CompileErrorSyntaxText(tc.message, tc.text); got != tc.want {
			t.Errorf("%q: got %q, want %q", tc.message, got, tc.want)
		}
	}
}
