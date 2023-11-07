package ast

import "git.sr.ht/~mango/andy/lexer"

type Program []Command

type Simple struct {
	Args   []Value
	Redirs []Redir
}

type Redir struct {
	File Value
	Mode RedirMode
}

type RedirMode int

const (
	RedirAppend RedirMode = iota
	RedirRead
	RedirWrite
	RedirWriteClob
)

func NewRedir(k lexer.TokenType) Redir {
	switch k {
	case lexer.TokAppend:
		return Redir{Mode: RedirAppend}
	case lexer.TokRead:
		return Redir{Mode: RedirRead}
	case lexer.TokWrite:
		return Redir{Mode: RedirWrite}
	case lexer.TokClobber:
		return Redir{Mode: RedirWriteClob}
	default:
		panic("unreachable")
	}
}

type Compound struct {
	Lhs, Rhs Command
	Op       CompoundOp
}

type CompoundOp int

const (
	CompoundPipe CompoundOp = iota
)

type Command interface {
	isCommand()
}

func (_ Simple) isCommand()   {}
func (_ Compound) isCommand() {}
