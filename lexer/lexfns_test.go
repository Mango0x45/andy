package lexer

import "testing"

func getTokens(s string) []TokenType {
	l := New(s)
	go l.Run()

	xs := []TokenType{}
	for t := range l.Out {
		xs = append(xs, t.Kind)
	}
	return xs
}

func assertTokens(t *testing.T, xs, ys []TokenType) {
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

func TestEmitTokenTypes(t *testing.T) {
	xs := []TokenType{
		TokEndStmt, TokArg, TokString, TokEndStmt, TokArg, TokArg, TokPipe,
		TokArg, TokLAnd, TokArg, TokClobber, TokArg, TokArg, TokArg, TokArg,
		TokEndStmt, TokEndStmt, TokEndStmt, TokEndStmt, TokArg, TokRead,
		TokArg, TokWrite, TokArg, TokEndStmt, TokPipe, TokArg, TokArg,
		TokAppend, TokString, TokArg, TokEndStmt, TokEndStmt, TokPipe, TokArg,
		TokRead, TokArg, TokRead, TokArg, TokRead, TokArg, TokRead, TokArg,
		TokRead, TokArg, TokRead, TokArg, TokEof,
	}
	s := `
	echo "hello world!"; cat my-file | tac && printf >| that was a no-op

	# IGNOREME

	cmd <file >file
	| another-cmd -f >> "foo.bar" -v

	| cmd  <there <is <a <nobreak < space < there`

	assertTokens(t, xs, getTokens(s))
}

func TestSkipComment(t *testing.T) {
	xs := []TokenType{
		TokEndStmt, TokEndStmt, TokEndStmt,
		TokEndStmt, TokEndStmt, TokArg,
		TokEndStmt, TokEof,
	}
	s := `
	#!/usr/bin/andy

	# This prints a newline

	echo # Hello world
	`

	assertTokens(t, xs, getTokens(s))
}
