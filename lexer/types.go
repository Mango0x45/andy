package lexer

import "unicode"

func IsMetaChar(r rune) bool {
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

func IsEol(r rune) bool {
	return r == ';' ||
		r == '\n'
}

func IsRefChar(r rune) bool {
	return unicode.IsLetter(r) ||
		r == '_' ||
		r == '′' ||
		r == '″' ||
		r == '‴' ||
		r == '⁗'
}

func IsRedir(kind TokenType) bool {
	return kind == TokAppend ||
		kind == TokClobber ||
		kind == TokRead ||
		kind == TokWrite
}

func IsValue(kind TokenType) bool {
	return kind == TokArg ||
		kind == TokConcat ||
		kind == TokPOpen ||
		kind == TokProcRdWr ||
		kind == TokProcRead ||
		kind == TokProcWrite ||
		kind == TokString ||
		kind == TokVarFlat ||
		kind == TokVarLen ||
		kind == TokVarRef
}
