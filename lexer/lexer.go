package lexer

import (
	"fmt"
	"unicode/utf8"
)

const eof rune = -1

type lexer struct {
	input    string     // The input string to lex
	inQuotes bool       // If we’re in double quotes
	Out      chan Token // Token output channel
	pos      int        // The pos of the cursor in input
	start    int        // The start of the current token in input
	width    int        // Width of the last rune lexed
}

func New(input string) *lexer {
	return &lexer{
		input: input,
		Out:   make(chan Token),
	}
}

func (l *lexer) Run() {
	for state := lexDefault; state != nil; {
		state = state(l)
	}
	close(l.Out)
}

func (l *lexer) emit(t TokenType) {
	l.Out <- Token{t, l.input[l.start:l.pos]}
}

func (l *lexer) next() rune {
	var r rune

	if l.pos >= len(l.input) {
		l.width = 0
		return eof
	}
	r, l.width = utf8.DecodeRuneInString(l.input[l.pos:])
	l.pos += l.width
	return r
}

func (l *lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

func (l *lexer) backup() {
	l.pos -= l.width
}

func (l *lexer) acceptRun(r rune) int {
	m := 0
	for l.next() == r {
		m++
	}
	l.backup()
	return m
}

func (l *lexer) errorf(format string, args ...any) lexFn {
	l.Out <- Token{
		Kind: TokError,
		Val:  fmt.Sprintf(format, args...),
	}
	return nil
}
