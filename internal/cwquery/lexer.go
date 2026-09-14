package cwquery

import (
	"strings"
	"unicode"
)

// Lex splits a condition string into tokens. The returned slice always ends with an EOF token.
func Lex(src string) ([]Token, error) {
	l := &lexer{src: src}
	var toks []Token
	for {
		t, err := l.next()
		if err != nil {
			return nil, err
		}
		toks = append(toks, t)
		if t.Kind == EOF {
			return toks, nil
		}
	}
}

type lexer struct {
	src string
	pos int
}

func (l *lexer) next() (Token, error) {
	l.skipSpace()
	if l.pos >= len(l.src) {
		return Token{Kind: EOF, Pos: l.pos}, nil
	}

	start := l.pos
	c := l.src[l.pos]

	switch {
	case c == '(':
		l.pos++
		return Token{Kind: LParen, Text: "(", Pos: start}, nil
	case c == ')':
		l.pos++
		return Token{Kind: RParen, Text: ")", Pos: start}, nil
	case c == ',':
		l.pos++
		return Token{Kind: Comma, Text: ",", Pos: start}, nil
	case c == '/':
		l.pos++
		return Token{Kind: Slash, Text: "/", Pos: start}, nil
	case c == '\'':
		return l.lexString()
	case c == '[':
		return l.lexDate()
	case c == '=' || c == '!' || c == '<' || c == '>':
		return l.lexOp()
	case isDigit(c) || (c == '-' && l.pos+1 < len(l.src) && isDigit(l.src[l.pos+1])):
		return l.lexNumber()
	case isIdentStart(c):
		return l.lexIdent()
	}

	return Token{}, syntaxErrorf(start, "unexpected character %q", string(c))
}

func (l *lexer) skipSpace() {
	for l.pos < len(l.src) && unicode.IsSpace(rune(l.src[l.pos])) {
		l.pos++
	}
}

func (l *lexer) lexString() (Token, error) {
	start := l.pos
	l.pos++ // opening quote
	var sb strings.Builder
	for l.pos < len(l.src) {
		c := l.src[l.pos]
		if c == '\'' {
			// doubled quote is an escaped quote
			if l.pos+1 < len(l.src) && l.src[l.pos+1] == '\'' {
				sb.WriteByte('\'')
				l.pos += 2
				continue
			}
			l.pos++
			return Token{Kind: String, Text: sb.String(), Pos: start}, nil
		}
		sb.WriteByte(c)
		l.pos++
	}

	return Token{}, syntaxErrorf(start, "unterminated string")
}

func (l *lexer) lexDate() (Token, error) {
	start := l.pos
	end := strings.IndexByte(l.src[l.pos:], ']')
	if end < 0 {
		return Token{}, syntaxErrorf(start, "unterminated date literal")
	}
	text := strings.TrimSpace(l.src[l.pos+1 : l.pos+end])
	l.pos += end + 1
	return Token{Kind: Date, Text: text, Pos: start}, nil
}

func (l *lexer) lexOp() (Token, error) {
	start := l.pos
	c := l.src[l.pos]
	two := ""
	if l.pos+1 < len(l.src) {
		two = l.src[l.pos : l.pos+2]
	}

	switch two {
	case "!=", "<=", ">=":
		l.pos += 2
		return Token{Kind: Op, Text: two, Pos: start}, nil
	}

	switch c {
	case '=', '<', '>':
		l.pos++
		return Token{Kind: Op, Text: string(c), Pos: start}, nil
	}

	return Token{}, syntaxErrorf(start, "unexpected character %q", string(c))
}

func (l *lexer) lexNumber() (Token, error) {
	start := l.pos
	if l.src[l.pos] == '-' {
		l.pos++
	}
	for l.pos < len(l.src) && isDigit(l.src[l.pos]) {
		l.pos++
	}
	if l.pos < len(l.src) && l.src[l.pos] == '.' {
		l.pos++
		if l.pos >= len(l.src) || !isDigit(l.src[l.pos]) {
			return Token{}, syntaxErrorf(start, "malformed number")
		}
		for l.pos < len(l.src) && isDigit(l.src[l.pos]) {
			l.pos++
		}
	}

	return Token{Kind: Number, Text: l.src[start:l.pos], Pos: start}, nil
}

func (l *lexer) lexIdent() (Token, error) {
	start := l.pos
	for l.pos < len(l.src) && isIdentPart(l.src[l.pos]) {
		l.pos++
	}

	return Token{Kind: Ident, Text: l.src[start:l.pos], Pos: start}, nil
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

func isIdentStart(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isIdentPart(c byte) bool { return isIdentStart(c) || isDigit(c) }
