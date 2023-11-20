package ast

import "git.sr.ht/~mango/andy/lexer"

func IsRedir(kind lexer.TokenType) bool {
	return kind == lexer.TokAppend ||
		kind == lexer.TokClobber ||
		kind == lexer.TokRead ||
		kind == lexer.TokWrite
}

func IsValue(kind lexer.TokenType) bool {
	return kind == lexer.TokArg ||
		kind == lexer.TokConcat ||
		kind == lexer.TokFlatRef ||
		kind == lexer.TokPOpen ||
		kind == lexer.TokString ||
		kind == lexer.TokVarRef ||
		kind == lexer.TokRefLen
}
