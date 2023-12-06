package main

import "fmt"

type tokenKind int

const (
	tokError tokenKind = iota

	tokEndStmt
	tokEof

	tokArg
	tokColon
	tokConcat
	tokString
	tokVarFlat
	tokVarLen
	tokVarRef

	tokAppend
	tokClobber
	tokRead
	tokWrite

	tokPipe

	tokLAnd
	tokLOr

	tokBraceOpen
	tokBraceClose
	tokBracketOpen
	tokBracketClose
	tokParenOpen
	tokParenClose

	tokProcSub
	tokProcRead
	tokProcWrite
	tokProcRdWr
)

type token struct {
	kind tokenKind
	val  string
}

const maxStrLen = 20

func (t token) String() string {
	switch t.kind {
	case tokError:
		return "lexing error: " + t.val

	case tokEndStmt:
		return "end of statement"
	case tokEof:
		return "end of file"

	case tokArg, tokString:
		if len(t.val) > maxStrLen {
			return fmt.Sprintf("‘%.*s…’", maxStrLen, t.val)
		}
		return "‘" + t.val + "’"
	case tokColon:
		return ":"
	case tokConcat:
		return "value concatination"
	case tokVarFlat:
		return fmt.Sprintf("‘$^%s’", t.val)
	case tokVarLen:
		return fmt.Sprintf("‘$#%s’", t.val)
	case tokVarRef:
		return fmt.Sprintf("‘$%s’", t.val)

	case tokAppend:
		return "‘>>’"
	case tokClobber:
		return "‘>!’"
	case tokRead:
		return "‘<’"
	case tokWrite:
		return "‘>’"

	case tokPipe:
		return "‘|’"

	case tokLAnd:
		return "‘&&’"
	case tokLOr:
		return "‘||’"

	case tokBraceOpen:
		return "‘{’"
	case tokBraceClose:
		return "‘}’"
	case tokBracketOpen:
		return "‘[’"
	case tokBracketClose:
		return "‘]’"
	case tokParenOpen:
		return "‘(’"
	case tokParenClose:
		return "‘)’"

	case tokProcSub:
		return "‘`{’"
	case tokProcRead:
		return "‘<{’"
	case tokProcWrite:
		return "‘>{’"
	case tokProcRdWr:
		return "‘<>{’"
	}

	panic("unreachable")
}
