package lexer

func isMetachar(r rune) bool {
	return r == '|' || r == '>' || r == '<'
}

func isEol(r rune) bool {
	return r == ';' || r == '\n'
}
