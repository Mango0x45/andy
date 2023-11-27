package lexer

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
)

var escapes = map[rune]rune{
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

type lexFn func(*lexer) lexFn

func lexDefault(l *lexer) lexFn {
	for {
		switch r := l.next(); {
		case IsEol(r):
			l.emit(TokEndStmt)
		case r == eof:
			l.emit(TokEof)
			return nil

		case strings.HasPrefix(l.input[l.pos-l.width:], "`{"):
			l.pos += 1
			l.emit(TokProcSub)
		case strings.HasPrefix(l.input[l.pos-l.width:], "<{"):
			l.pos += 1
			l.emit(TokProcRead)
		case strings.HasPrefix(l.input[l.pos-l.width:], ">{"):
			l.pos += 1
			l.emit(TokProcWrite)
		case strings.HasPrefix(l.input[l.pos-l.width:], "<>{"):
			l.pos += 2
			l.emit(TokProcRdWr)

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
			l.emit(TokBcOpen)
		case r == '}':
			l.emit(TokBcClose)
		case r == '(':
			l.emit(TokPOpen)
		case r == ')':
			l.emit(TokPClose)
			return lexMaybeConcat
		case r == ']' && l.brktDepth > 0:
			l.emit(TokBkClose)
			l.brktDepth--
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
			r, err := escape(l.next())
			if err != nil {
				l.errorf("%s", err)
			}
			sb.WriteRune(r)
		case r == ']' && l.brktDepth > 0:
			l.backup()
			l.Out <- Token{TokArg, sb.String()}
			return lexDefault
		case unicode.IsSpace(r) || IsMetaChar(r) || IsEol(r) || r == eof:
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
		kind = TokVarFlat
	}
	switch l.peek() {
	case '^':
		if l.inQuotes {
			return l.errorf("The ‘^’ variable prefix is redundant in double-quoted strings")
		}
		kind = TokVarFlat
		l.next()
	case '#':
		kind = TokVarLen
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

	if braces && l.peek() != '}' {
		return l.errorf("unterminated braced variable ‘${%s’",
			l.input[l.start:l.pos])
	}
	l.emit(kind)
	if braces {
		l.next() // Consume closing brace
	}

	if l.peek() == '[' {
		l.emit(TokBkOpen)
		l.next()
		l.brktDepth++
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
			r, err := escape(l.next())
			if err != nil {
				l.errorf("%s", err)
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
	case unicode.IsSpace(r) || IsMetaChar(r) || IsEol(r) || r == eof:
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

func escape(r rune) (rune, error) {
	if unicode.IsSpace(r) || IsMetaChar(r) {
		return r, nil
	} else if r, ok := escapes[r]; ok {
		return r, nil
	}
	return -1, errors.New(fmt.Sprintf("invalid escape sequence ‘\\%c’", r))
}
