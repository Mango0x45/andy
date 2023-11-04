package lexer

import (
	"fmt"
	"math"
	"strings"
	"unicode/utf8"
)

const eof = -1

type lexer struct {
	input string
	start int
	pos   int
	width int
	Out   chan Token
}

func New(input string) *lexer {
	l := &lexer{
		input: input,
		Out:   make(chan Token),
	}
	return l
}

func (l *lexer) Run() {
	for state := firstState; state != nil; {
		state = state(l)
	}
	close(l.Out)
}

func (l *lexer) emit(t TokenType) {
	l.Out <- Token{
		Kind: t,
		Val: l.input[l.start:l.pos],
	}
	l.start = l.pos
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

func (l *lexer) accept(valid string) bool {
	if strings.IndexRune(valid, l.next()) != -1 {
		return true
	}
	l.backup()
	return false
}

func (l *lexer) acceptRun(valid string) int {
	return l.acceptNRun(valid, math.MaxInt)
}

func (l *lexer) acceptNRun(valid string, m int) int {
	n := 0
	for strings.IndexRune(valid, l.next()) != -1 && n < m {
		n++
	}
	l.backup()
	return n
}

func (l *lexer) errorf(format string, args ...any) lexFn {
	l.Out <- Token{
		Kind: TokError,
		Val: fmt.Sprintf(format, args...),
	}
	return nil
}
