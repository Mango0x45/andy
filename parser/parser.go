package parser

import (
	"git.sr.ht/~mango/andy/lexer"
	"git.sr.ht/~mango/andy/vm"
)

type Parser struct {
	stream <-chan lexer.Token // Incoming stream of tokens
	cache  *lexer.Token       // Cached token used for peeking
}

func New(stream <-chan lexer.Token) *Parser {
	return &Parser{stream: stream}
}

func (p *Parser) Run() vm.Program {
	return p.parseProgram()
}

func (p *Parser) next() lexer.Token {
	var t lexer.Token
	if p.cache != nil {
		t, p.cache = *p.cache, nil
	} else {
		t = <-p.stream
	}
	return t
}

func (p *Parser) peek() lexer.Token {
	if p.cache != nil {
		return *p.cache
	}

	t, ok := <-p.stream
	if !ok {
		t = lexer.Token{Kind: lexer.TokEof}
	}
	p.cache = &t
	return t
}
