package cwquery

import (
	"errors"
	"testing"
)

func TestLex(t *testing.T) {
	toks, err := Lex(`summary contains 'it''s' and company/id in (1, -2.5) or closedDate > [2026-01-02T00:00:00Z]`)
	if err != nil {
		t.Fatal(err)
	}

	want := []Token{
		{Ident, "summary", 0},
		{Ident, "contains", 8},
		{String, "it's", 17},
		{Ident, "and", 25},
		{Ident, "company", 29},
		{Slash, "/", 36},
		{Ident, "id", 37},
		{Ident, "in", 40},
		{LParen, "(", 43},
		{Number, "1", 44},
		{Comma, ",", 45},
		{Number, "-2.5", 47},
		{RParen, ")", 51},
		{Ident, "or", 53},
		{Ident, "closedDate", 56},
		{Op, ">", 67},
		{Date, "2026-01-02T00:00:00Z", 69},
		{EOF, "", 91},
	}

	if len(toks) != len(want) {
		t.Fatalf("got %d tokens, want %d: %+v", len(toks), len(want), toks)
	}
	for i := range want {
		if toks[i] != want[i] {
			t.Errorf("token %d: got %+v, want %+v", i, toks[i], want[i])
		}
	}
}

func TestLexOps(t *testing.T) {
	toks, err := Lex(`a = 1 b != 2 c <= 3 d >= 4 e < 5 f > 6`)
	if err != nil {
		t.Fatal(err)
	}
	var ops []string
	for _, tk := range toks {
		if tk.Kind == Op {
			ops = append(ops, tk.Text)
		}
	}
	want := []string{"=", "!=", "<=", ">=", "<", ">"}
	if len(ops) != len(want) {
		t.Fatalf("ops = %v, want %v", ops, want)
	}
	for i := range want {
		if ops[i] != want[i] {
			t.Errorf("op %d = %q, want %q", i, ops[i], want[i])
		}
	}
}

func TestParseEmptyIsAlways(t *testing.T) {
	for _, src := range []string{"", "   ", "\n\t"} {
		e, err := Parse(src)
		if err != nil {
			t.Fatalf("%q: %v", src, err)
		}
		if _, ok := e.(Always); !ok {
			t.Errorf("%q: got %T, want Always", src, e)
		}
	}
}

func TestParsePrecedence(t *testing.T) {
	// a or b and c  ==>  a or (b and c)
	e, err := Parse("a = 1 or b = 2 and c = 3")
	if err != nil {
		t.Fatal(err)
	}
	top, ok := e.(Binary)
	if !ok || top.Op != Or {
		t.Fatalf("top = %#v, want Or", e)
	}
	right, ok := top.R.(Binary)
	if !ok || right.Op != And {
		t.Fatalf("right = %#v, want And", top.R)
	}

	// (a or b) and c
	e, err = Parse("(a = 1 or b = 2) and c = 3")
	if err != nil {
		t.Fatal(err)
	}
	top, ok = e.(Binary)
	if !ok || top.Op != And {
		t.Fatalf("top = %#v, want And", e)
	}
	if _, ok := top.L.(Binary); !ok {
		t.Fatalf("left = %#v, want Binary", top.L)
	}
}

func TestParseNotForms(t *testing.T) {
	cases := map[string]func(Expr) bool{
		"not a = 1":             func(e Expr) bool { _, ok := e.(Not); return ok },
		"a not in (1,2)":        func(e Expr) bool { n, ok := e.(In); return ok && n.Negate },
		"a not like 'x*'":       func(e Expr) bool { n, ok := e.(Not); return ok && n.X.(Compare).Op == Like },
		"a not contains 'x'":    func(e Expr) bool { n, ok := e.(Not); return ok && n.X.(Compare).Op == Contains },
		"NOT (a = 1 AND b = 2)": func(e Expr) bool { _, ok := e.(Not); return ok },
	}
	for src, check := range cases {
		e, err := Parse(src)
		if err != nil {
			t.Errorf("%q: %v", src, err)
			continue
		}
		if !check(e) {
			t.Errorf("%q: unexpected tree %#v", src, e)
		}
	}
}

func TestParseErrors(t *testing.T) {
	cases := []struct {
		src string
		pos int
	}{
		{"summary =", 9},         // missing value
		{"(a = 1", 6},            // unbalanced paren
		{"a = 1 b = 2", 6},       // trailing garbage
		{"a == 1", 3},            // '==' lexes as '=' then '=' -> value expected at 3
		{"a = 'unterminated", 4}, // unterminated string
		{"a like 5", 7},          // like needs string
		{"a not = 1", 6},         // not followed by op
		{"a in ()", 6},           // empty list
		{"a > [not-a-date]", 4},  // bad date
		{"and = 1", 0},           // keyword as path
		{"a = 1 $", 6},           // bad character
		{"company/ = 1", 9},      // dangling slash
	}

	for _, c := range cases {
		_, err := Parse(c.src)
		if err == nil {
			t.Errorf("%q: expected error", c.src)
			continue
		}
		var se *SyntaxError
		if !errors.As(err, &se) {
			t.Errorf("%q: error %T is not *SyntaxError", c.src, err)
			continue
		}
		if se.Pos != c.pos {
			t.Errorf("%q: pos = %d (%s), want %d", c.src, se.Pos, se.Msg, c.pos)
		}
	}
}
