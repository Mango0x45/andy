package vm

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
		kind == lexer.TokPOpen ||
		kind == lexer.TokProcRdWr ||
		kind == lexer.TokProcRead ||
		kind == lexer.TokProcWrite ||
		kind == lexer.TokString ||
		kind == lexer.TokVarFlat ||
		kind == lexer.TokVarLen ||
		kind == lexer.TokVarRef
}
