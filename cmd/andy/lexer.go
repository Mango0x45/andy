package main

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

var stringEscapes = map[rune]rune{
	'\\': '\\',
	'0':  '\000',
	'a':  '\a',
	'b':  '\b',
	'f':  '\f',
	'n':  '\n',
	'r':  '\r',
	't':  '\t',
	'v':  '\v',
}

const eof rune = -1

type lexer struct {
	input        string
	out          chan token
	pos          int
	start        int
	width        int
	bracketDepth int
	inQuotes     bool
}

type lexFn func(*lexer) lexFn

func newLexer(s string) lexer {
	return lexer{
		input: s,
		out:   make(chan token),
	}
}

func (l *lexer) run() {
	for state := lexDefault; state != nil; {
		state = state(l)
	}
	close(l.out)
}

func (l *lexer) emit(t tokenKind) {
	l.out <- token{t, l.input[l.start:l.pos]}
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
	l.out <- token{
		kind: tokError,
		val:  fmt.Sprintf(format, args...),
	}
	return nil
}

func lexDefault(l *lexer) lexFn {
	for {
		switch r := l.next(); {
		case isEol(r):
			l.emit(tokEndStmt)
		case r == eof:
			l.emit(tokEof)
			return nil

		case strings.HasPrefix(l.input[l.pos-l.width:], "`{"):
			l.pos += 1
			l.emit(tokProcSub)
		case strings.HasPrefix(l.input[l.pos-l.width:], "<{"):
			l.pos += 1
			l.emit(tokProcRead)
		case strings.HasPrefix(l.input[l.pos-l.width:], ">{"):
			l.pos += 1
			l.emit(tokProcWrite)
		case strings.HasPrefix(l.input[l.pos-l.width:], "<>{"):
			l.pos += 2
			l.emit(tokProcRdWr)

		case r == '&':
			return lexAmp
		case r == '|':
			return lexPipe
		case r == '\'':
			l.backup()
			return lexStringSingle
		case r == '"':
			l.backup()
			return lexStringDouble
		case r == '<':
			l.emit(tokRead)
		case r == '>':
			return lexWrite
		case r == '{':
			l.emit(tokBraceOpen)
		case r == '}':
			l.emit(tokBraceClose)
		case r == '(':
			l.emit(tokParenOpen)
		case r == ')':
			l.emit(tokParenClose)
			return lexMaybeConcat
		case r == ']' && l.bracketDepth > 0:
			l.emit(tokBracketClose)
			l.bracketDepth--
			if l.inQuotes {
				return lexStringDouble
			}
			return lexMaybeConcat
		case r == '#':
			return skipComment
		case r == '$':
			l.backup()
			return lexVarRef
		case unicode.IsSpace(r):
		default:
			l.backup()
			return lexArg
		}
	}
}

func skipComment(l *lexer) lexFn {
	if i := strings.IndexByte(l.input[l.pos:], '\n'); i != -1 {
		l.pos += i
	}
	return lexDefault
}

func lexAmp(l *lexer) lexFn {
	switch l.peek() {
	case '&':
		l.next()
		l.emit(tokLAnd)
	default:
		panic("Implement & operator")
	}
	return lexDefault
}

func lexPipe(l *lexer) lexFn {
	switch l.peek() {
	case '|':
		l.next()
		l.emit(tokLOr)
	default:
		l.emit(tokPipe)
	}
	return lexDefault
}

func lexArg(l *lexer) lexFn {
	sb := strings.Builder{}
	for {
		switch r := l.next(); {
		case r == '\\':
			r, err := escapeRune(l.next())
			if err != nil {
				l.errorf("%s", err)
			}
			sb.WriteRune(r)
		case r == ']' && l.bracketDepth > 0:
			l.backup()
			l.out <- token{tokArg, sb.String()}
			return lexDefault
		case unicode.IsSpace(r) || isMetachar(r) || isEol(r) || r == eof:
			l.backup()
			l.out <- token{tokArg, sb.String()}
			return lexMaybeConcat
		default:
			sb.WriteRune(r)
		}
	}
}

func lexVarRef(l *lexer) lexFn {
	l.next() // Consume ‘$’

	// Flat or not?
	kind := tokVarRef
	if l.inQuotes {
		kind = tokVarFlat
	}
	switch l.peek() {
	case '^':
		if l.inQuotes {
			return l.errorf("The ‘^’ variable prefix is redundant in double-quoted strings")
		}
		kind = tokVarFlat
		l.next()
	case '#':
		kind = tokVarLen
		l.next()
	}

	// Optional surrounding braces
	braces := false
	if l.peek() == '{' {
		braces = true
		l.next()
	}
	l.start = l.pos

	l.pos += strings.IndexFunc(l.input[l.pos:], func(r rune) bool {
		return !isRefRune(r)
	})
	if l.pos < l.start {
		l.pos = len(l.input)
	}

	if braces && l.peek() != '}' {
		return l.errorf("unterminated braced variable ‘${%s’",
			l.input[l.start:l.pos])
	}
	l.emit(kind)
	if braces {
		l.next() // Consume closing brace
	}

	if l.peek() == '[' {
		l.emit(tokBracketOpen)
		l.next()
		l.bracketDepth++
		return lexDefault
	}
	if l.inQuotes {
		return lexStringDouble
	}
	return lexMaybeConcat
}

func lexStringSingle(l *lexer) lexFn {
	start := l.pos
	n := l.acceptRun('\'')
	l.start = l.pos

	l.pos += strings.Index(l.input[l.pos:], l.input[start:start+n])
	if l.pos < l.start {
		return l.errorf("unterminated string")
	}

	l.emit(tokString)
	l.pos += n
	return lexMaybeConcat
}

func lexStringDouble(l *lexer) lexFn {
	// Consume quote
	if l.inQuotes {
		l.emit(tokConcat)
		l.inQuotes = false
	} else {
		l.next()
	}

	sb := strings.Builder{}
	for {
		switch r := l.next(); r {
		case eof:
			return l.errorf("unterminated string")
		case '\\':
			r, err := escapeRune(l.next())
			if err != nil {
				l.errorf("%s", err)
			}
			sb.WriteRune(r)
		case '$':
			l.backup()
			l.inQuotes = true
			fallthrough
		case '"':
			l.out <- token{tokString, sb.String()}
			return lexMaybeConcat
		default:
			sb.WriteRune(r)
		}
	}
}

func lexMaybeConcat(l *lexer) lexFn {
	switch r := l.peek(); {
	case r == '\'':
		l.emit(tokConcat)
		return lexStringSingle
	case r == '"':
		l.emit(tokConcat)
		return lexStringDouble
	case r == '(':
		l.emit(tokConcat)
		return lexDefault
	case r == '$':
		l.emit(tokConcat)
		return lexVarRef
	case unicode.IsSpace(r) || isMetachar(r) || isEol(r) || r == eof:
		return lexDefault
	}
	l.emit(tokConcat)
	return lexArg
}

func lexWrite(l *lexer) lexFn {
	switch l.peek() {
	case '!':
		l.next()
		l.emit(tokClobber)
	case '>':
		l.next()
		l.emit(tokAppend)
	default:
		l.emit(tokWrite)
	}
	return lexDefault
}

func escapeRune(r rune) (rune, error) {
	if unicode.IsSpace(r) || isMetachar(r) {
		return r, nil
	} else if r, ok := stringEscapes[r]; ok {
		return r, nil
	}
	return -1, errors.New(fmt.Sprintf("invalid escape sequence ‘\\%c’", r))
}
