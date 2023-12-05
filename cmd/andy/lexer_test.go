package main

import "testing"

// BEGIN TESTING LEXER OBJECT

func TestNext(t *testing.T) {
	s := "¢ȠʗǱɓǇϴ¤Ίϑ'щƎcɛǩΟȏɁƅ"
	l := newLexer(s)

	for _, x := range []rune(s) {
		if y := l.next(); x != y {
			t.Fatalf("Expected ‘%c’ but got ‘%c’", x, y)
		}
	}

	if r := l.next(); r != eof {
		t.Fatalf("Expected ‘eof’ but got ‘%c’", r)
	}
}

func TestPeek(t *testing.T) {
	s := "¢ȠʗǱɓǇϴ¤Ίϑ'щƎcɛǩΟȏɁƅ"
	l := newLexer(s)
	chk := func(x, y rune) {
		if x != y {
			t.Fatalf("Expected ‘%c’ but got ‘%c’", x, y)
		}
	}

	rs := []rune(s)
	chk(l.peek(), rs[0])
	chk(l.peek(), rs[0])

	l.next()
	l.next()

	chk(l.peek(), rs[2])
	chk(l.peek(), rs[2])

	l.next()
	l.next()
	l.next()
	l.next()

	chk(l.peek(), rs[6])
	chk(l.peek(), rs[6])
}

// BEGIN TESTING LEXER STATE FUNCTIONS

func getTokens(s string) []tokenKind {
	l := newLexer(s)
	go l.run()

	xs := []tokenKind{}
	for t := range l.out {
		xs = append(xs, t.kind)
	}
	return xs
}

func assertTokens(t *testing.T, xs, ys []tokenKind) {
	for i := range xs {
		if xs[i] != ys[i] {
			t.Fatalf("Expected token type %d at position %d but got %d",
				xs[i], i, ys[i])
		}
	}

	if len(xs) != len(ys) {
		t.Fatalf("Expected %d tokens but got %d", len(xs), len(ys))
	}
}

func TestEmitTokenTypes1(t *testing.T) {
	xs := []tokenKind{
		tokEndStmt, tokArg, tokString, tokEndStmt, tokArg, tokArg, tokPipe,
		tokArg, tokLAnd, tokArg, tokClobber, tokArg, tokArg, tokArg, tokArg,
		tokEndStmt, tokEndStmt, tokEndStmt, tokEndStmt, tokArg, tokRead,
		tokArg, tokWrite, tokArg, tokEndStmt, tokPipe, tokArg, tokArg,
		tokAppend, tokString, tokArg, tokEndStmt, tokEndStmt, tokPipe, tokArg,
		tokRead, tokArg, tokRead, tokArg, tokRead, tokArg, tokRead, tokString,
		tokRead, tokArg, tokRead, tokArg, tokEof,
	}
	s := `
	echo "hello world!"; cat my-file | tac && printf >! that was a no-op

	# IGNOREME

	cmd <file >file
	| another-cmd -f >> "foo.bar" -v

	| cmd  <there <is <a <r#.nob'reak.# < space < there`

	assertTokens(t, xs, getTokens(s))
}

func TestEmitTokenTypes2(t *testing.T) {
	xs := []tokenKind{
		tokEndStmt, tokArg, tokParenOpen, tokArg, tokArg, tokParenClose, tokConcat,
		tokArg, tokEndStmt, tokArg, tokArg, tokConcat, tokParenOpen, tokArg,
		tokArg, tokParenClose, tokEof,
	}
	s := `
	echo (foo bar).c
	echo foo(bar baz)`

	assertTokens(t, xs, getTokens(s))
}

func TestSkipComment(t *testing.T) {
	xs := []tokenKind{
		tokEndStmt, tokEndStmt, tokEndStmt,
		tokEndStmt, tokEndStmt, tokArg,
		tokEndStmt, tokEof,
	}
	s := `
	#!/usr/bin/andy

	# This prints a newline

	echo # Hello world
	`

	assertTokens(t, xs, getTokens(s))
}

func TestLexProcSub(t *testing.T) {
	xs := []tokenKind{
		tokEndStmt, tokArg, tokProcSub, tokArg, tokArg, tokBraceClose,
		tokEndStmt, tokArg, tokProcSub, tokParenOpen, tokArg, tokArg,
		tokArg, tokParenClose, tokArg, tokArg, tokBraceClose,
		tokEndStmt, tokArg, tokVarRef, tokBracketOpen, tokProcSub,
		tokParenOpen, tokArg, tokArg, tokArg, tokParenClose, tokArg,
		tokArg, tokBraceClose, tokBracketClose, tokEndStmt, tokEof,
	}
	s := `
	echo ` + "`" + `{echo foo}
	echo ` + "`" + `(a b c){echo foo}
	echo $xs[` + "`" + `(a b c)echo foo]
	`

	assertTokens(t, xs, getTokens(s))
}
