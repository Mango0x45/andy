package lexer

import "testing"

func TestEmitTokenTypes(t *testing.T) {
	xs := []TokenType{
		TokEndStmt, TokArg, TokString, TokEndStmt, TokArg, TokArg, TokPipe,
		TokArg, TokLAnd, TokArg, TokClobber, TokArg, TokArg, TokArg, TokArg,
		TokEndStmt, TokEndStmt, TokEndStmt, TokArg, TokRead, TokArg, TokWrite,
		TokArg, TokEndStmt, TokPipe, TokArg, TokArg, TokAppend, TokString,
		TokArg, TokEndStmt, TokEndStmt, TokPipe, TokArg, TokRead, TokArg,
		TokRead, TokArg, TokRead, TokArg, TokRead, TokArg, TokRead, TokArg,
		TokRead, TokArg, TokEof,
	}
	s := `
	echo "hello world!"; cat my-file | tac && printf >| that was a no-op

	# IGNOREME

	cmd <file >file
	| another-cmd -f >> "foo.bar" -v

	| cmd  <there <is <a <nobreak < space < there`

	l := New(s)
	go l.Run()

	ys := make([]TokenType, 0, len(xs))
	for t := range l.Out {
		ys = append(ys, t.Kind)
	}

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
