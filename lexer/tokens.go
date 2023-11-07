package lexer

import "fmt"

type TokenType int

const (
	// TokError is the token emitted during a lexing error.  It signals the end
	// of lexical analysis.
	TokError TokenType = iota

	TokEndStmt // End of statement, either a newline or semicolon
	TokEof     // End of file

	TokArg    // An unquoted string
	TokString // A quoted string

	TokAppend  // The ‘>>’ operator
	TokClobber // The ‘>|’ operator
	TokRead    // The ‘<’ operator
	TokWrite   // The ‘>’ operator

	TokPipe // The ‘|’ operator
)

type Token struct {
	Kind TokenType
	Val  string
}

// Maximum length of a string before truncation in diagnostics printing
// TokString
const maxStrLen = 20

func (t Token) String() string {
	switch t.Kind {
	case TokError:
		return "Error: " + t.Val

	case TokEndStmt:
		return "end of statement"
	case TokEof:
		return "EOF"

	case TokArg, TokString:
		if len(t.Val) > maxStrLen {
			return fmt.Sprintf("%.*s…", maxStrLen, t.Val)
		}
		return t.Val

	case TokAppend:
		return ">>"
	case TokClobber:
		return ">|"
	case TokRead:
		return "<"
	case TokWrite:
		return ">"

	case TokPipe:
		return "|"
	}

	panic("unreachable")
}
