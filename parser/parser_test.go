package parser

import (
	"testing"

	"git.sr.ht/~mango/andy/lexer"
)

func TestNext(t *testing.T) {
	xs := []lexer.Token{
		{Kind: lexer.TokString},
		{Kind: lexer.TokEndStmt},
		{Kind: lexer.TokEof},
		{Kind: lexer.TokError},
	}
	c := make(chan lexer.Token, len(xs))
	p := parser{toks: c}

	for _, x := range xs {
		c <- x
	}

	for i := range xs {
		x := p.next()
		if x != xs[i] {
			t.Errorf("Expected %v but got %v", xs[i], x)
		}
	}
}

func TestPeek(t *testing.T) {
	xs := []lexer.Token{
		{Kind: lexer.TokString},
		{Kind: lexer.TokEndStmt},
		{Kind: lexer.TokEof},
		{Kind: lexer.TokError},
	}
	c := make(chan lexer.Token, len(xs))
	p := parser{toks: c}

	for _, x := range xs {
		c <- x
	}

	f := func(x lexer.Token, i int) {
		if x != xs[i] {
			t.Errorf("Expected %v but got %v", xs[i], x)
		}
	}

	f(p.peek(), 0)
	f(p.peek(), 0)
	f(p.next(), 0)
	f(p.peek(), 1)
	f(p.peek(), 1)
	f(p.next(), 1)
}
