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
	TokConcat // Concatination between two values
	TokString // A quoted string

	TokAppend  // The ‘>>’ operator
	TokClobber // The ‘>!’ operator
	TokRead    // The ‘<’ operator
	TokWrite   // The ‘>’ operator

	TokPipe // The ‘|’ operator

	TokLAnd // The ‘&&’ operator
	TokLOr  // The ‘||’ operator

	TokBOpen  // The opening brace of a compound command
	TokBClose // The opening brace of a compound command
	TokPOpen  // The opening parenthesis of a list
	TokPClose // The closing parenthesis of a list
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
		return "lexing error: " + t.Val

	case TokEndStmt:
		return "end of statement"
	case TokEof:
		return "end of file"

	case TokArg, TokString:
		if len(t.Val) > maxStrLen {
			return fmt.Sprintf("‘%.*s…’", maxStrLen, t.Val)
		}
		return "‘" + t.Val + "’"
	case TokConcat:
		return "value concatination"

	case TokAppend:
		return "‘>>’"
	case TokClobber:
		return "‘>!’"
	case TokRead:
		return "‘<’"
	case TokWrite:
		return "‘>’"

	case TokPipe:
		return "‘|’"

	case TokLAnd:
		return "‘&&’"
	case TokLOr:
		return "‘||’"

	case TokBOpen:
		return "‘{’"
	case TokBClose:
		return "‘}’"
	case TokPOpen:
		return "‘(’"
	case TokPClose:
		return "‘)’"
	}

	panic("unreachable")
}
