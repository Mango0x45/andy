package lexer

import "unicode"

type lexFn func(*lexer) lexFn

var firstState = lexDefault

func lexDefault(l *lexer) lexFn {
	for {
		l.start = l.pos
		switch r := l.next(); {
		case r == eof:
			l.emit(TokEof)
			return nil
		case r == '\'':
			l.backup()
			return lexStringSingle
		case r == '"':
			l.backup()
			return lexStringDouble
		case r == ';', r == '\n':
			l.emit(TokEndStmt)
		case r == '#':
			return lexComment
		case r == '>':
			return lexRedirWrite
		case r == '<':
			return lexRedirRead
		case unicode.IsSpace(r):
		default:
			l.backup()
			return lexStringRaw
		}
	}
}

func lexComment(l *lexer) lexFn {
	for !isStmtEnd(l.next()) {
	}

	l.emit(TokEndStmt)
	return lexDefault
}

func lexStringRaw(l *lexer) lexFn {
	l.start = l.pos
	for {
		if r := l.next(); isStmtEndOrSpace(r) || isMetaChar(r) {
			l.backup()
			l.emit(TokString)
			return lexMaybeConcat
		}
	}
}

func lexStringSingle(l *lexer) lexFn {
	sqs := l.acceptRun("'")
	l.start = l.pos

	for {
		switch r := l.next(); r {
		case eof:
			return l.errorf("unterminated single-quoted string: ‘%s’", Token{
				Kind: TokString,
				Val:  l.input[l.start:l.pos],
			})
		case '\'':
			if qs := l.acceptNRun("'", sqs-1); qs+1 == sqs {
				// We can add/sub to l.pos by qs only because a quote is 1 byte.
				// The rune we’re currently on though might be a multibyte rune,
				// so we need to backup properly.
				l.backup()
				l.pos -= qs
				l.emit(TokString)
				l.pos += qs
				l.next()
				return lexMaybeConcat
			}
		}
	}
}

func lexStringDouble(l *lexer) lexFn {
	l.next() // Consume opening quote
	l.start = l.pos

	for {
		switch r := l.next(); r {
		case eof:
			return l.errorf("unterminated double-quoted string: ‘%s’", Token{
				Kind: TokString,
				Val:  l.input[l.start:l.pos],
			})
		case '"':
			l.backup()
			l.emit(TokString)
			l.next()
			return lexMaybeConcat
		}
	}
}

func lexMaybeConcat(l *lexer) lexFn {
	r := l.next()
	if isStmtEndOrSpace(r) || isMetaNoQuotes(r) {
		l.backup()
		return lexDefault
	}

	l.emit(TokConcat)
	l.backup()
	switch r {
	case '\'':
		return lexStringSingle
	case '"':
		return lexStringDouble
	}
	return lexStringRaw
}

func lexRedirWrite(l *lexer) lexFn {
	switch r := l.next(); r {
	case '|':
		l.emit(TokWriteClob)
	case '!':
		l.emit(TokWriteErr)
	case '_':
		l.emit(TokWriteNull)
	default:
		l.backup()
		l.emit(TokWrite)
	}
	return lexDefault
}

func lexRedirRead(l *lexer) lexFn {
	switch r := l.next(); r {
	case '_':
		l.emit(TokReadNull)
	default:
		l.backup()
		l.emit(TokRead)
	}
	return lexDefault
}
