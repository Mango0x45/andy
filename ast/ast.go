package ast

import (
	"os"
	"os/user"
	"strings"

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

	In() *os.File
	Out() *os.File
	Err() *os.File

	SetIn(*os.File)
	SetOut(*os.File)
	SetErr(*os.File)
}

// Simple is the simplest form of a command, just arguments and redirects
type Simple struct {
	Args         []Value
	Redirs       []Redirect
	in, out, err *os.File
}

// Compound is a code wrapped within braces
type Compound struct {
	Cmds         []CommandList
	in, out, err *os.File
}

// If is a conditional branch; it executes Body if Cond was successful
type If struct {
	Cond, Body   CommandList
	Else         *CommandList
	in, out, err *os.File
}

type While struct {
	Cond, Body   CommandList
	in, out, err *os.File
}

func (_ Simple) isCommand()   {}
func (_ Compound) isCommand() {}
func (_ If) isCommand()       {}
func (_ While) isCommand()    {}

func (c Simple) In() *os.File    { return c.in }
func (c Simple) Out() *os.File   { return c.out }
func (c Simple) Err() *os.File   { return c.err }
func (c Compound) In() *os.File  { return c.in }
func (c Compound) Out() *os.File { return c.out }
func (c Compound) Err() *os.File { return c.err }
func (c If) In() *os.File        { return c.in }
func (c If) Out() *os.File       { return c.out }
func (c If) Err() *os.File       { return c.err }
func (c While) In() *os.File     { return c.in }
func (c While) Out() *os.File    { return c.out }
func (c While) Err() *os.File    { return c.err }

func (c *Simple) SetIn(f *os.File)    { c.in = f }
func (c *Simple) SetOut(f *os.File)   { c.out = f }
func (c *Simple) SetErr(f *os.File)   { c.err = f }
func (c *Compound) SetIn(f *os.File)  { c.in = f }
func (c *Compound) SetOut(f *os.File) { c.out = f }
func (c *Compound) SetErr(f *os.File) { c.err = f }
func (c *If) SetIn(f *os.File)        { c.in = f }
func (c *If) SetOut(f *os.File)       { c.out = f }
func (c *If) SetErr(f *os.File)       { c.err = f }
func (c *While) SetIn(f *os.File)     { c.in = f }
func (c *While) SetOut(f *os.File)    { c.out = f }
func (c *While) SetErr(f *os.File)    { c.err = f }

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
	return []string{a.TildeExpand()}
}

func (a Argument) TildeExpand() string {
	s := string(a)
	if len(s) == 0 || s[0] != '~' {
		return s
	}
	i := strings.IndexByte(s, '/')
	if i == -1 {
		i = len(s)
	}

	var u *user.User
	var err error
	if i == 1 {
		u, err = user.Current()
	} else {
		u, err = user.Lookup(s[1:i])
	}

	if err != nil {
		return s
	}
	return u.HomeDir + s[i:]
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
