package lexer

import "fmt"

type TokenType int

const (
	// TokError is the token emitted during a lexing error.  It signals the
	// end of lexical analysis.
	TokError TokenType = iota

	TokConcat    // Not a real syntax element, signals string concatination
	TokEndStmt   // End of statement, either a newline or semicolon
	TokEof       // End of file
	TokRead      // The ‘<’ operator
	TokReadNull  // The ‘<_’ operator
	TokString    // A string
	TokWrite     // The ‘>’ operator
	TokWriteClob // The ‘>|’ operator
	TokWriteErr  // The ‘>!’ operator
	TokWriteNull // The ‘>_’ operator
)

func (t TokenType) IsRead() bool {
	return t == TokRead || t == TokReadNull
}

func (t TokenType) IsWrite() bool {
	return t == TokWrite ||
		t == TokWriteClob ||
		t == TokWriteErr ||
		t == TokWriteNull
}

const maxStrLen = 20

type Token struct {
	Kind TokenType
	Val  string
}

func (t Token) String() string {
	switch t.Kind {
	case TokError:
		return t.Val
	case TokConcat:
		return "string concatination"
	case TokEndStmt:
		return "end of line"
	case TokEof:
		return "EOF"
	case TokRead:
		return "<"
	case TokReadNull:
		return "<_"
	case TokString:
		if len(t.Val) > maxStrLen {
			return fmt.Sprintf("%.*s…", maxStrLen, t.Val)
		}
		return t.Val
	case TokWrite:
		return ">"
	case TokWriteClob:
		return ">|"
	case TokWriteErr:
		return ">!"
	case TokWriteNull:
		return ">_"
	}

	panic("unreachable")
}
