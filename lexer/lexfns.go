package lexer

import "unicode"

type lexFn func(*lexer) lexFn

func lexDefault(l *lexer) lexFn {
	for {
		switch r := l.next(); {
		case isEol(r):
			l.emit(TokEndStmt)
		case r == eof:
			l.emit(TokEof)
			return nil
		case r == '|':
			l.emit(TokPipe)
		case r == '"':
			l.backup()
			return lexStringDouble
		case r == '<':
			l.emit(TokRead)
		case r == '>':
			return lexWrite
		case unicode.IsSpace(r):
		default:
			l.backup()
			return lexArg
		}
	}
}

func lexArg(l *lexer) lexFn {
	l.align()
	for {
		if r := l.next(); isMetachar(r) || unicode.IsSpace(r) || r == eof {
			l.backup()
			l.emit(TokArg)
			return lexDefault
		}
	}
}

func lexStringDouble(l *lexer) lexFn {
	l.next() // Consume quote
	l.align()

	for r := l.next(); r != '"'; {
		if r == eof {
			return l.errorf("unterminated string")
		}
		r = l.next()
	}

	l.backup()
	l.emit(TokString)
	l.next()
	return lexDefault
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
