package lexer

import (
	"strings"
	"unicode"
)

var backslashEsc = map[rune]rune{
	'\\': '\\',
	'n':  '\n',
	't':  '\t',
}

type lexFn func(*lexer) lexFn

func lexDefault(l *lexer) lexFn {
	for {
		switch r := l.next(); {
		case isEol(r):
			l.emit(TokEndStmt)
		case r == eof:
			l.emit(TokEof)
			return nil
		case r == '&':
			return lexAmp
		case r == '|':
			return lexPipe
		case r == '"':
			l.backup()
			return lexStringDouble
		case r == '<':
			l.emit(TokRead)
		case r == '>':
			return lexWrite
		case r == '#':
			return skipComment
		case unicode.IsSpace(r):
		default:
			l.backup()
			return lexArg
		}
	}
}

func skipComment(l *lexer) lexFn {
	for {
		if t := l.next(); t == '\n' || t == eof {
			l.backup()
			return lexDefault
		}
	}
}

func lexAmp(l *lexer) lexFn {
	switch l.peek() {
	case '&':
		l.next()
		l.emit(TokLAnd)
	default:
		panic("Implement & operator")
	}
	return lexDefault
}

func lexPipe(l *lexer) lexFn {
	switch l.peek() {
	case '|':
		l.next()
		l.emit(TokLOr)
	default:
		l.emit(TokPipe)
	}
	return lexDefault
}

func lexArg(l *lexer) lexFn {
	l.start = l.pos
	for {
		if r := l.next(); isMetachar(r) || isEol(r) || unicode.IsSpace(r) || r == eof {
			l.backup()
			l.emit(TokArg)
			return lexDefault
		}
	}
}

func lexStringDouble(l *lexer) lexFn {
	l.next() // Consume quote

	sb := strings.Builder{}
	for {
		switch r := l.next(); r {
		case eof:
			return l.errorf("unterminated string")
		case '\\':
			r, ok := backslashEsc[l.next()]
			if !ok {
				return l.errorf("invalid escape sequence ‘\\%c’", r)
			}
			sb.WriteRune(r)
		case '"':
			l.Out <- Token{TokString, sb.String()}
			return lexDefault
		default:
			sb.WriteRune(r)
		}
	}
}

func lexWrite(l *lexer) lexFn {
	switch l.peek() {
	case '|':
		l.next()
		l.emit(TokClobber)
	case '>':
		l.next()
		l.emit(TokAppend)
	default:
		l.emit(TokWrite)
	}
	return lexDefault
}
