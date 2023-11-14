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
type Pipeline []Command

// Command is a command the shell can execute
type Command interface {
	isCommand()
	GetIn() *os.File
	GetOut() *os.File
	GetErr() *os.File
	SetIn(*os.File)
	SetOut(*os.File)
	SetErr(*os.File)
}

// Simple is the simplest form of a command, just arguments and redirects
type Simple struct {
	Args         []Value
	Redirs       []Redirect
	In, Out, Err *os.File
}

type If struct {
	Cond, Body   CommandList
	Redirs       []Redirect
	In, Out, Err *os.File
}

func (c *Simple) GetIn() *os.File  { return c.In }
func (c *Simple) GetOut() *os.File { return c.Out }
func (c *Simple) GetErr() *os.File { return c.Err }
func (c *If) GetIn() *os.File      { return c.In }
func (c *If) GetOut() *os.File     { return c.Out }
func (c *If) GetErr() *os.File     { return c.Err }

func (c *Simple) SetIn(f *os.File)  { c.In = f }
func (c *Simple) SetOut(f *os.File) { c.Out = f }
func (c *Simple) SetErr(f *os.File) { c.Err = f }
func (c *If) SetIn(f *os.File)      { c.In = f }
func (c *If) SetOut(f *os.File)     { c.Out = f }
func (c *If) SetErr(f *os.File)     { c.Err = f }

func (_ Simple) isCommand() {}
func (_ If) isCommand()     {}

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

type List []Value

func (l List) ToStrings() []string {
	xs := make([]string, 0, len(l))
	for _, x := range l {
		xs = append(xs, x.ToStrings()...)
	}
	return xs
}

type BinaryOp int

const (
	LAnd BinaryOp = iota
	LOr
)
