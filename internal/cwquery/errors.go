package cwquery

import "fmt"

// SyntaxError reports a lexing or parsing failure. Pos is the 0-based byte offset into the source.
type SyntaxError struct {
	Pos int
	Msg string
}

func (e *SyntaxError) Error() string {
	return fmt.Sprintf("%s at position %d", e.Msg, e.Pos)
}

func syntaxErrorf(pos int, format string, args ...any) *SyntaxError {
	return &SyntaxError{Pos: pos, Msg: fmt.Sprintf(format, args...)}
}
