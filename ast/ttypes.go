package ast

import "git.sr.ht/~mango/andy/lexer"

func IsRedir(kind lexer.TokenType) bool {
	return kind == lexer.TokAppend ||
		kind == lexer.TokClobber ||
		kind == lexer.TokRead ||
		kind == lexer.TokWrite
}
