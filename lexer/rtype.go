package lexer

import "unicode"

func isStmtEnd(r rune) bool {
	return r == ';' || r == '\n' || r == eof
}

func isStmtEndOrSpace(r rune) bool {
	return isStmtEnd(r) || unicode.IsSpace(r)
}

func isMetaChar(r rune) bool {
	return r == '\'' ||
		r == '"' ||
		r == '>' ||
		r == '<'
}

func isMetaNoQuotes(r rune) bool {
	if r == '\'' || r == '"' {
		return false
	}
	return isMetaChar(r)
}
