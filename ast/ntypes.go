package ast

import "git.sr.ht/~mango/andy/lexer"

func IsValue(k lexer.TokenType) bool {
	return k == lexer.TokArg || k == lexer.TokString
}

func IsRedir(k lexer.TokenType) bool {
	return k == lexer.TokAppend ||
		k == lexer.TokRead ||
		k == lexer.TokWrite ||
		k == lexer.TokClobber
}
