package vm

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/user"
	"strconv"
	"strings"
	"unicode"

	"git.sr.ht/~mango/andy/builtin"
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

	In() io.Reader
	Out() io.Writer
	Err() io.Writer

	SetIn(io.Reader)
	SetOut(io.Writer)
	SetErr(io.Writer)

	Redirs() []Redirect
	SetRedirs([]Redirect)
}

// Simple is the simplest form of a command, just arguments and redirects
type Simple struct {
	Args     []Value
	redirs   []Redirect
	in       io.Reader
	out, err io.Writer
}

// Compound is a code wrapped within braces
type Compound struct {
	Cmds     []CommandList
	redirs   []Redirect
	in       io.Reader
	out, err io.Writer
}

// If is a conditional branch; it executes Body if Cond was successful
type If struct {
	Cond       CommandList
	Body, Else []CommandList
	redirs     []Redirect
	in         io.Reader
	out, err   io.Writer
}

// While is a loop; it executes Body for as long as Cond is successful
type While struct {
	Cond     CommandList
	Body     []CommandList
	redirs   []Redirect
	in       io.Reader
	out, err io.Writer
}

func (_ Simple) isCommand()   {}
func (_ Compound) isCommand() {}
func (_ If) isCommand()       {}
func (_ While) isCommand()    {}

func (c Simple) In() io.Reader    { return c.in }
func (c Simple) Out() io.Writer   { return c.out }
func (c Simple) Err() io.Writer   { return c.err }
func (c Compound) In() io.Reader  { return c.in }
func (c Compound) Out() io.Writer { return c.out }
func (c Compound) Err() io.Writer { return c.err }
func (c If) In() io.Reader        { return c.in }
func (c If) Out() io.Writer       { return c.out }
func (c If) Err() io.Writer       { return c.err }
func (c While) In() io.Reader     { return c.in }
func (c While) Out() io.Writer    { return c.out }
func (c While) Err() io.Writer    { return c.err }

func (c *Simple) SetIn(r io.Reader)    { c.in = r }
func (c *Simple) SetOut(w io.Writer)   { c.out = w }
func (c *Simple) SetErr(w io.Writer)   { c.err = w }
func (c *Compound) SetIn(r io.Reader)  { c.in = r }
func (c *Compound) SetOut(w io.Writer) { c.out = w }
func (c *Compound) SetErr(w io.Writer) { c.err = w }
func (c *If) SetIn(r io.Reader)        { c.in = r }
func (c *If) SetOut(w io.Writer)       { c.out = w }
func (c *If) SetErr(w io.Writer)       { c.err = w }
func (c *While) SetIn(r io.Reader)     { c.in = r }
func (c *While) SetOut(w io.Writer)    { c.out = w }
func (c *While) SetErr(w io.Writer)    { c.err = w }

func (c *Simple) Redirs() []Redirect   { return c.redirs }
func (c *Compound) Redirs() []Redirect { return c.redirs }
func (c *If) Redirs() []Redirect       { return c.redirs }
func (c *While) Redirs() []Redirect    { return c.redirs }

func (c *Simple) SetRedirs(rs []Redirect)   { c.redirs = rs }
func (c *Compound) SetRedirs(rs []Redirect) { c.redirs = rs }
func (c *If) SetRedirs(rs []Redirect)       { c.redirs = rs }
func (c *While) SetRedirs(rs []Redirect)    { c.redirs = rs }

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
	ToStrings(ctx context) ([]string, commandResult)
}

type Argument string

func (a Argument) ToStrings(_ context) ([]string, commandResult) {
	s, err := tildeExpand(string(a))
	if err != nil {
		return []string{}, errInternal{err}
	}
	return []string{s}, nil
}

func tildeExpand(s string) (string, error) {
	if len(s) == 0 || s[0] != '~' {
		return s, nil
	}
	i := strings.IndexByte(s, '/')
	if i == -1 {
		i = len(s)
	}

	if i == 1 {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return home + s[i:], nil
	}

	name := s[1:i]
	switch u, err := user.Lookup(name); {
	case errors.Is(err, user.UnknownUserError(name)):
		return s, nil
	case err != nil:
		return "", err
	default:
		return u.HomeDir + s[i:], nil
	}
}

type String string

func (s String) ToStrings(_ context) ([]string, commandResult) {
	return []string{string(s)}, nil
}

type VarRefType int

const (
	VrExpand VarRefType = iota
	VrFlatten
	VrLength
)

type VarRef struct {
	Ident   string
	Type    VarRefType
	Indices []Value
}

func getIndex(s string, n int) (int, commandResult) {
	i, err := strconv.Atoi(s)
	if err, ok := err.(*strconv.NumError); ok {
		var str string

		switch {
		case errors.Is(err, strconv.ErrRange):
			str = fmt.Sprintf("index ‘%s’ is out of range; what are you even doing?", s)
		case errors.Is(err, strconv.ErrSyntax):
			str = fmt.Sprintf("‘%s’ isn’t a valid index", s)
		}

		return 0, errInternal{errors.New(str)}
	}

	if i < 0 {
		i = n + i
	}
	if i < 0 || i >= n {
		str := fmt.Sprintf("invalid index ‘%s’ into list of length %d", s, n)
		return 0, errInternal{errors.New(str)}
	}

	return i, nil
}

func (vr VarRef) ToStrings(ctx context) ([]string, commandResult) {
	xs, _ := builtin.VarTable[vr.Ident]

	if vr.Indices != nil {
		ys := make([]string, 0, len(xs))
		for _, i := range vr.Indices {
			ss, err := i.ToStrings(ctx)
			if err != nil {
				return nil, err
			}
			for _, s := range ss {
				i, err := getIndex(s, len(xs))
				if err != nil {
					return nil, errInternal{err}
				}

				ys = append(ys, xs[i])
			}
		}
		xs = ys
	}

	switch vr.Type {
	case VrFlatten:
		xs = []string{strings.Join(xs, " ")}
	case VrLength:
		xs = []string{strconv.Itoa(len(xs))}
	}
	return xs, nil
}

func NewVarRef(t lexer.Token) VarRef {
	vr := VarRef{Ident: t.Val}
	switch t.Kind {
	case lexer.TokVarFlat:
		vr.Type = VrFlatten
	case lexer.TokVarLen:
		vr.Type = VrLength
	}
	return vr
}

type Concat struct {
	Lhs, Rhs Value
}

func (c Concat) ToStrings(ctx context) ([]string, commandResult) {
	xs, err := c.Lhs.ToStrings(ctx)
	if err != nil {
		return []string{}, err
	}
	ys, err := c.Rhs.ToStrings(ctx)
	if err != nil {
		return []string{}, err
	}
	zs := make([]string, 0, len(xs)*len(ys))

	for _, x := range xs {
		for _, y := range ys {
			zs = append(zs, x+y)
		}
	}

	return zs, nil
}

type List []Value

func (l List) ToStrings(ctx context) ([]string, commandResult) {
	xs := make([]string, 0, len(l))
	for _, x := range l {
		ys, err := x.ToStrings(ctx)
		if err != nil {
			return []string{}, err
		}
		xs = append(xs, ys...)
	}
	return xs, nil
}

type ProcSub struct {
	Body []CommandList
}

func (ps ProcSub) ToStrings(ctx context) ([]string, commandResult) {
	var out bytes.Buffer
	ctx.out = &out

	if res := execCmdLists(ps.Body, ctx); res.ExitCode() != 0 {
		return nil, res
	}

	s := strings.TrimRightFunc(out.String(), unicode.IsSpace)
	return []string{s}, nil
}

type ProcRedir struct {
	Type ProcRedirType
	Body []CommandList
	r, w *os.File
}

func (pr *ProcRedir) ToStrings(ctx context) ([]string, commandResult) {
	xs := make([]string, 0, 2)
	r, w, err := os.Pipe()
	if err != nil {
		return []string{}, errInternal{err}
	}
	pr.r = r
	pr.w = w

	if pr.Is(ProcRead) {
		ctx.out = w
		xs = append(xs, devFd(r))
	}
	if pr.Is(ProcWrite) {
		ctx.in = r
		xs = append(xs, devFd(w))
	}

	go func() {
		if res := execCmdLists(pr.Body, ctx); failed(res) {
			panic("TODO")
		}
		if pr.Is(ProcRead) {
			w.Close()
		}
		if pr.Is(ProcWrite) {
			r.Close()
		}
	}()

	return xs, nil
}

func (pr ProcRedir) Close() {
	if pr.Is(ProcRead) {
		pr.r.Close()
	}
	if pr.Is(ProcWrite) {
		pr.w.Close()
	}
}

func (pr ProcRedir) OpenFiles() []*os.File {
	var xs []*os.File
	if pr.Is(ProcRead) {
		xs = append(xs, pr.r)
	}
	if pr.Is(ProcWrite) {
		xs = append(xs, pr.w)
	}
	return xs
}

func devFd(f *os.File) string {
	return fmt.Sprintf("/dev/fd/%d", f.Fd())
}

func (pr ProcRedir) Is(t ProcRedirType) bool {
	return pr.Type&t != 0
}

type ProcRedirType int

const (
	ProcRead ProcRedirType = 1 << iota
	ProcWrite
)

type BinaryOp int

const (
	LAnd BinaryOp = iota
	LOr
)
