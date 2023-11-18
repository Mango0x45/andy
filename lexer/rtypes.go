package lexer

import "unicode"

func isMetaChar(r rune) bool {
	return r == '\'' ||
		r == '"' ||
		r == '|' ||
		r == '>' ||
		r == '<' ||
		r == '&' ||
		r == '{' ||
		r == '}' ||
		r == '(' ||
		r == ')' ||
		r == '$'
}

func isEol(r rune) bool {
	return r == ';' || r == '\n'
}

func IsRefChar(r rune) bool {
	return unicode.IsLetter(r) ||
		r == '_' ||
		r == '′' ||
		r == '″' ||
		r == '‴' ||
		r == '⁗'
}
