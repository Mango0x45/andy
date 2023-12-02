package main

import "unicode"

func isMetachar(r rune) bool {
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

func isRefRune(r rune) bool {
	return unicode.IsLetter(r) ||
		unicode.IsNumber(r) ||
		r == '_' ||
		r == '′' ||
		r == '″' ||
		r == '‴' ||
		r == '⁗'
}

func isRefName(s string) (bool, rune) {
	for _, r := range s {
		if !isRefRune(r) {
			return false, r
		}
	}

	return true, 0
}

func isRedirTok(kind tokenKind) bool {
	return kind == tokAppend ||
		kind == tokClobber ||
		kind == tokRead ||
		kind == tokWrite
}

func isValueTok(kind tokenKind) bool {
	return kind == tokArg ||
		kind == tokConcat ||
		kind == tokParenOpen ||
		kind == tokProcRdWr ||
		kind == tokProcRead ||
		kind == tokProcSub ||
		kind == tokProcWrite ||
		kind == tokString ||
		kind == tokVarFlat ||
		kind == tokVarLen ||
		kind == tokVarRef
}
