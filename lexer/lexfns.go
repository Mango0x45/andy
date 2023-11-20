package lexer

import (
	"strings"
	"unicode"
)

var backslashEsc = map[rune]rune{
	'\\': '\\',
	'$':  '$',
	'0':  '\000',
	'a':  '\a',
	'b':  '\b',
	'f':  '\f',
	'n':  '\n',
	'r':  '\r',
	't':  '\t',
	'v':  '\v',
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
		case r == '\'':
			l.backup()
			return lexStringSingle
		case r == '"':
			l.backup()
			return lexStringDouble
		case r == '<':
			l.emit(TokRead)
		case r == '>':
			return lexWrite
		case r == '{':
			l.emit(TokBOpen)
		case r == '}':
			l.emit(TokBClose)
		case r == '(':
			l.emit(TokPOpen)
		case r == ')':
			l.emit(TokPClose)
			return lexMaybeConcat
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
	if i := strings.IndexByte(l.input[l.pos:], '\n'); i != -1 {
		l.pos += i
	}
	return lexDefault
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
	sb := strings.Builder{}
	for {
		switch r := l.next(); {
		case r == '\\':
			if r := l.next(); unicode.IsSpace(r) || isMetaChar(r) {
				sb.WriteRune(r)
			} else if r, ok := backslashEsc[r]; ok {
				sb.WriteRune(r)
			} else {
				return l.errorf("invalid escape sequence ‘\\%c’", r)
			}
		case unicode.IsSpace(r) || isMetaChar(r) || isEol(r) || r == eof:
			l.backup()
			l.Out <- Token{TokArg, sb.String()}
			return lexMaybeConcat
		default:
			sb.WriteRune(r)
		}
	}
}

func lexVarRef(l *lexer) lexFn {
	l.next() // Consume ‘$’

	// Flat or not?
	kind := TokVarRef
	if l.inQuotes {
		kind = TokFlatRef
	}
	switch l.peek() {
	case '^':
		if l.inQuotes {
			return l.errorf("The ‘^’ variable prefix is redundant in double-quoted strings")
		}
		kind = TokFlatRef
		l.next()
	case '#':
		kind = TokRefLen
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
		return !IsRefChar(r)
	})
	if l.pos < l.start {
		l.pos = len(l.input)
	}

	if braces {
		if l.peek() != '}' {
			return l.errorf("unterminated braced variable ‘${%s’",
				l.input[l.start:l.pos])
		}
		// Defer so that l.emit() doesn’t emit the brace in .Val
		defer l.next()
	}

	l.emit(kind)
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

	l.emit(TokString)
	l.pos += n
	return lexMaybeConcat
}

func lexStringDouble(l *lexer) lexFn {
	// Consume quote
	if l.inQuotes {
		l.emit(TokConcat)
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
			r, ok := backslashEsc[l.next()]
			if !ok {
				return l.errorf("invalid escape sequence ‘\\%c’", r)
			}
			sb.WriteRune(r)
		case '$':
			l.backup()
			l.inQuotes = true
			fallthrough
		case '"':
			l.Out <- Token{TokString, sb.String()}
			return lexMaybeConcat
		default:
			sb.WriteRune(r)
		}
	}
}

func lexMaybeConcat(l *lexer) lexFn {
	switch r := l.peek(); {
	case r == '\'':
		l.emit(TokConcat)
		return lexStringSingle
	case r == '"':
		l.emit(TokConcat)
		return lexStringDouble
	case r == '(':
		l.emit(TokConcat)
		return lexDefault
	case r == '$':
		l.emit(TokConcat)
		return lexVarRef
	case unicode.IsSpace(r) || isMetaChar(r) || isEol(r) || r == eof:
		return lexDefault
	}
	l.emit(TokConcat)
	return lexArg
}

func lexWrite(l *lexer) lexFn {
	switch l.peek() {
	case '!':
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
