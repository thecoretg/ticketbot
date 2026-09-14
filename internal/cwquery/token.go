package cwquery

import "fmt"

// Kind identifies a lexical token.
type Kind int

const (
	EOF Kind = iota
	Ident
	String
	Number
	Date
	LParen
	RParen
	Comma
	Slash
	Op // = != < > <= >=
)

func (k Kind) String() string {
	switch k {
	case EOF:
		return "end of input"
	case Ident:
		return "identifier"
	case String:
		return "string"
	case Number:
		return "number"
	case Date:
		return "date"
	case LParen:
		return "'('"
	case RParen:
		return "')'"
	case Comma:
		return "','"
	case Slash:
		return "'/'"
	case Op:
		return "operator"
	default:
		return fmt.Sprintf("Kind(%d)", int(k))
	}
}

// Token is a lexed unit of a condition string. Pos is the 0-based byte offset of the token's first character.
type Token struct {
	Kind Kind
	Text string // literal text; for String the unescaped contents, for Date the bracket contents
	Pos  int
}
