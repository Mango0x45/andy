package parser

import (
	"fmt"
	"os"

	"git.sr.ht/~mango/andy/lexer"
)

type errExpected struct {
	e, g any
}

func (e errExpected) Error() string {
	return fmt.Sprintf("Expected %v but got %v", e.e, e.g)
}

type errRedirect lexer.TokenType

func (e errRedirect) Error() string {
	switch t := lexer.TokenType(e); {
	case t.IsRead():
		return "You can’t read from multiple files at once"
	case t.IsWrite():
		return "You can’t write to multiple files at once"
	}
	panic("unreachable")
}

func eprintln(e error) {
	fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], e)
	os.Exit(1)
}

type parser struct {
	toks  <-chan lexer.Token
	cache *lexer.Token
}

func (p *parser) next() lexer.Token {
	var t lexer.Token
	if p.cache != nil {
		t, p.cache = *p.cache, nil
	} else {
		t = <-p.toks
	}
	return t
}

func (p *parser) peek() lexer.Token {
	if p.cache == nil {
		t := <-p.toks
		p.cache = &t
	}
	return *p.cache
}
