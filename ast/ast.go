package ast

import (
	"os"

	"git.sr.ht/~mango/andy/lexer"
)

// See grammar.ebnf in the project root for details

// Program is a complete script
type Program = []CommandList

// XCommandList is a list of pipelines connected by binary operators, but in
// the form of (op e1 (op e2 (op e3 (cons e4 nil)))) instead of the form we
// actually need of (op (op (op (cons nil e1) e2) e3) e4).
type XCommandList struct {
	Lhs Pipeline
	Op  BinaryOp
	Rhs *XCommandList
}

// CommandList is a list of pipelines connected by binary operators
type CommandList struct {
	Lhs *CommandList
	Op  BinaryOp
	Rhs Pipeline
}

// Pipeline is a list of commands connected by pipes
type Pipeline []Simple

// Simple is the simplest form of a command, just arguments and redirects
type Simple struct {
	Args         []Value
	Redirs       []Redirect
	In, Out, Err *os.File
}

// Redirect is a redirection between files and file descriptors
type Redirect struct {
	Type RedirType
	File Value
}

type RedirType int

const (
	RedirAppend RedirType = iota
	RedirClob
	RedirRead
	RedirWrite
)

func NewRedir(k lexer.TokenType) Redirect {
	switch k {
	case lexer.TokAppend:
		return Redirect{Type: RedirAppend}
	case lexer.TokClobber:
		return Redirect{Type: RedirClob}
	case lexer.TokRead:
		return Redirect{Type: RedirRead}
	case lexer.TokWrite:
		return Redirect{Type: RedirWrite}
	}
	panic("unreachable")
}

// Value is anything that can be turned into a (list of) string(s)
type Value interface {
	ToStrings() []string
}

func NewValue(t lexer.Token) Value {
	switch t.Kind {
	case lexer.TokArg:
		return Argument(t.Val)
	case lexer.TokString:
		return String(t.Val)
	}
	panic("unreachable")
}

type Argument string

func (a Argument) ToStrings() []string {
	return []string{string(a)}
}

type String string

func (s String) ToStrings() []string {
	return []string{string(s)}
}

type Concat struct {
	Lhs, Rhs Value
}

func (c Concat) ToStrings() []string {
	xs := c.Lhs.ToStrings()
	ys := c.Rhs.ToStrings()
	zs := make([]string, 0, len(xs)*len(ys))

	for _, x := range xs {
		for _, y := range ys {
			zs = append(zs, x+y)
		}
	}

	return zs
}

type BinaryOp int

const (
	LAnd BinaryOp = iota
	LOr
)
