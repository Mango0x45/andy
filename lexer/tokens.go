package lexer

import "fmt"

type TokenType int

const (
	// TokError is the token emitted during a lexing error.  It signals the end
	// of lexical analysis.
	TokError TokenType = iota

	TokEndStmt // End of statement, either a newline or semicolon
	TokEof     // End of file

	TokArg     // An unquoted string
	TokConcat  // Concatination between two values
	TokVarFlat // A flattened variable reference
	TokVarLen  // The length of a variable
	TokVarRef  // A variable reference
	TokString  // A quoted string

	TokAppend  // The ‘>>’ operator
	TokClobber // The ‘>!’ operator
	TokRead    // The ‘<’ operator
	TokWrite   // The ‘>’ operator

	TokPipe // The ‘|’ operator

	TokLAnd // The ‘&&’ operator
	TokLOr  // The ‘||’ operator

	TokBcOpen  // The opening brace of a compound command
	TokBcClose // The closing brace of a compound command
	TokBkOpen  // The opening bracket of a variable index
	TokBkClose // The closing bracket of a variable index
	TokPOpen   // The opening parenthesis of a list
	TokPClose  // The closing parenthesis of a list
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
	case TokVarFlat:
		return fmt.Sprintf("‘$^%s’", t.Val)
	case TokVarLen:
		return fmt.Sprintf("‘$#%s’", t.Val)
	case TokVarRef:
		return fmt.Sprintf("‘$%s’", t.Val)

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

	case TokBcOpen:
		return "‘{’"
	case TokBcClose:
		return "‘}’"
	case TokBkOpen:
		return "‘[’"
	case TokBkClose:
		return "‘]’"
	case TokPOpen:
		return "‘(’"
	case TokPClose:
		return "‘)’"
	}

	panic("unreachable")
}
