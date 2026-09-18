package cwquery

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Query is a compiled condition ready for evaluation.
type Query struct {
	Source string
	Expr   Expr
}

// Compile lexes and parses src. An empty or whitespace-only source compiles to Always.
func Compile(src string) (*Query, error) {
	e, err := Parse(src)
	if err != nil {
		return nil, err
	}

	return &Query{Source: src, Expr: e}, nil
}

// Parse returns the expression tree for src.
func Parse(src string) (Expr, error) {
	toks, err := Lex(src)
	if err != nil {
		return nil, err
	}

	p := &parser{toks: toks}
	if p.peek().Kind == EOF {
		return Always{}, nil
	}

	e, err := p.parseOr()
	if err != nil {
		return nil, err
	}

	if t := p.peek(); t.Kind != EOF {
		return nil, syntaxErrorf(t.Pos, "unexpected %s %q", t.Kind, t.Text)
	}

	return e, nil
}

type parser struct {
	toks []Token
	i    int
}

func (p *parser) peek() Token { return p.toks[p.i] }

func (p *parser) advance() Token {
	t := p.toks[p.i]
	if t.Kind != EOF {
		p.i++
	}
	return t
}

func (p *parser) isKeyword(kw string) bool {
	t := p.peek()
	return t.Kind == Ident && strings.EqualFold(t.Text, kw)
}

func (p *parser) expect(k Kind) (Token, error) {
	t := p.peek()
	if t.Kind != k {
		return t, syntaxErrorf(t.Pos, "expected %s, got %s %q", k, t.Kind, t.Text)
	}
	return p.advance(), nil
}

func (p *parser) parseOr() (Expr, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}

	for p.isKeyword("or") {
		p.advance()
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = Binary{Op: Or, L: left, R: right}
	}

	return left, nil
}

func (p *parser) parseAnd() (Expr, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}

	for p.isKeyword("and") {
		p.advance()
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		left = Binary{Op: And, L: left, R: right}
	}

	return left, nil
}

func (p *parser) parseUnary() (Expr, error) {
	if p.isKeyword("not") {
		p.advance()
		x, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return Not{X: x}, nil
	}

	return p.parsePrimary()
}

func (p *parser) parsePrimary() (Expr, error) {
	if p.peek().Kind == LParen {
		p.advance()
		e, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(RParen); err != nil {
			return nil, err
		}
		return e, nil
	}

	return p.parseComparison()
}

func (p *parser) parseComparison() (Expr, error) {
	path, err := p.parsePath()
	if err != nil {
		return nil, err
	}

	negate := false
	if p.isKeyword("not") {
		p.advance()
		negate = true
	}

	t := p.peek()
	switch {
	case t.Kind == Op:
		if negate {
			return nil, syntaxErrorf(t.Pos, "'not' must be followed by in, like or contains")
		}
		p.advance()
		val, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		return Compare{Path: path, Op: cmpOpFor(t.Text), Val: val}, nil

	case p.isKeyword("in"):
		p.advance()
		if p.isKeyword("list") {
			p.advance()
			idTok := p.peek()
			id, ok := listIDToken(idTok)
			if !ok {
				return nil, syntaxErrorf(idTok.Pos, "expected a list id (positive integer), got %s %q", idTok.Kind, idTok.Text)
			}
			p.advance()
			return InList{Path: path, ListID: id, Negate: negate, Pos: idTok.Pos}, nil
		}
		vals, err := p.parseList()
		if err != nil {
			return nil, err
		}
		return In{Path: path, Vals: vals, Negate: negate}, nil

	case p.isKeyword("like"), p.isKeyword("contains"):
		p.advance()
		vt := p.peek()
		if vt.Kind != String {
			return nil, syntaxErrorf(vt.Pos, "%s requires a quoted string", strings.ToLower(t.Text))
		}
		p.advance()
		c := Compare{Path: path, Val: Value{Kind: KString, S: vt.Text}}
		if strings.EqualFold(t.Text, "like") {
			c.Op = Like
			c.re = likeRegexp(vt.Text)
		} else {
			c.Op = Contains
		}
		if negate {
			return Not{X: c}, nil
		}
		return c, nil
	}

	if negate {
		return nil, syntaxErrorf(t.Pos, "'not' must be followed by in, like or contains")
	}

	return nil, syntaxErrorf(t.Pos, "expected operator, got %s %q", t.Kind, t.Text)
}

func (p *parser) parsePath() ([]string, error) {
	t, err := p.expect(Ident)
	if err != nil {
		return nil, err
	}
	if isReserved(t.Text) {
		return nil, syntaxErrorf(t.Pos, "unexpected keyword %q", t.Text)
	}

	path := []string{t.Text}
	for p.peek().Kind == Slash {
		p.advance()
		seg, err := p.expect(Ident)
		if err != nil {
			return nil, err
		}
		path = append(path, seg.Text)
	}

	return path, nil
}

func (p *parser) parseList() ([]Value, error) {
	if _, err := p.expect(LParen); err != nil {
		return nil, err
	}

	var vals []Value
	for {
		v, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		vals = append(vals, v)

		if p.peek().Kind == Comma {
			p.advance()
			continue
		}
		break
	}

	if _, err := p.expect(RParen); err != nil {
		return nil, err
	}

	return vals, nil
}

func (p *parser) parseValue() (Value, error) {
	t := p.peek()
	switch t.Kind {
	case String:
		p.advance()
		return Value{Kind: KString, S: t.Text}, nil
	case Number:
		p.advance()
		n, err := strconv.ParseFloat(t.Text, 64)
		if err != nil {
			return Value{}, syntaxErrorf(t.Pos, "malformed number %q", t.Text)
		}
		return Value{Kind: KNumber, N: n, S: t.Text}, nil
	case Date:
		p.advance()
		tm, err := parseDate(t.Text)
		if err != nil {
			return Value{}, syntaxErrorf(t.Pos, "malformed date %q", t.Text)
		}
		return Value{Kind: KTime, T: tm, S: t.Text}, nil
	case Ident:
		switch strings.ToLower(t.Text) {
		case "true":
			p.advance()
			return Value{Kind: KBool, B: true}, nil
		case "false":
			p.advance()
			return Value{Kind: KBool, B: false}, nil
		case "null":
			p.advance()
			return Value{Kind: KNull}, nil
		}
	}

	return Value{}, syntaxErrorf(t.Pos, "expected value, got %s %q", t.Kind, t.Text)
}

// listIDToken accepts a plain positive integer token as a list id.
func listIDToken(t Token) (int, bool) {
	if t.Kind != Number {
		return 0, false
	}
	for i := 0; i < len(t.Text); i++ {
		if t.Text[i] < '0' || t.Text[i] > '9' {
			return 0, false
		}
	}
	id, err := strconv.Atoi(t.Text)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

func cmpOpFor(text string) CmpOp {
	switch text {
	case "=":
		return Eq
	case "!=":
		return Ne
	case "<":
		return Lt
	case ">":
		return Gt
	case "<=":
		return Le
	default:
		return Ge
	}
}

func isReserved(s string) bool {
	switch strings.ToLower(s) {
	case "and", "or", "not", "in", "like", "contains", "true", "false", "null":
		return true
	}
	return false
}

var dateLayouts = []string{
	time.RFC3339,
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
	"2006-01-02",
}

func parseDate(s string) (time.Time, error) {
	var err error
	for _, layout := range dateLayouts {
		var t time.Time
		t, err = time.Parse(layout, s)
		if err == nil {
			return t, nil
		}
	}
	return time.Time{}, err
}

// likeRegexp turns a CW/SQL-style pattern into an anchored, case-insensitive regexp.
// '*' and '%' match any run of characters; '_' matches a single character.
func likeRegexp(pattern string) *regexp.Regexp {
	var sb strings.Builder
	sb.WriteString("(?is)^")
	for _, r := range pattern {
		switch r {
		case '*', '%':
			sb.WriteString(".*")
		case '_':
			sb.WriteString(".")
		default:
			sb.WriteString(regexp.QuoteMeta(string(r)))
		}
	}
	sb.WriteString("$")

	return regexp.MustCompile(sb.String())
}
