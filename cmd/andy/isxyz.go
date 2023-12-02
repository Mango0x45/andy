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

func isRedirTok(k tokenKind) bool {
	return k == tokAppend ||
		k == tokClobber ||
		k == tokRead ||
		k == tokWrite
}

func isValueTok(k tokenKind) bool {
	return k == tokArg ||
		k == tokConcat ||
		k == tokParenOpen ||
		k == tokProcRdWr ||
		k == tokProcRead ||
		k == tokProcSub ||
		k == tokProcWrite ||
		k == tokString ||
		k == tokVarFlat ||
		k == tokVarLen ||
		k == tokVarRef
}
