package main

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
)

// See grammar.ebnf in the project root for details

type astProgram = []astTopLevel

type astTopLevel interface {
	isTopLevel()
}

type astFuncDef struct {
	args []astValue
	body []astTopLevel
}

type astCommandList struct {
	lhs *astCommandList
	op  astBinaryOp
	rhs astPipeline
}

type astXCommandList struct {
	lhs astPipeline
	op  astBinaryOp
	rhs *astXCommandList
}

func (_ astFuncDef) isTopLevel()     {}
func (_ astCommandList) isTopLevel() {}

type astPipeline []astCleanCommand

type astCleanCommand struct {
	cmd astCommand
	xs  []io.Closer
}

func (cc astCleanCommand) cleanup() {
	for _, x := range cc.xs {
		x.Close()
	}
}

func (cc *astCleanCommand) add(x io.Closer) {
	cc.xs = append(cc.xs, x)
}

type astCommand interface {
	isCommand()

	redirs() []astRedirect
	setRedirs([]astRedirect)
}

type astSimple struct {
	args []astValue
	rs   []astRedirect
}

type astCompound struct {
	cmds []astTopLevel
	rs   []astRedirect
}

type astIf struct {
	cond        astCommandList
	body, else_ []astTopLevel
	rs          []astRedirect
}

type astWhile struct {
	cond astCommandList
	body []astTopLevel
	rs   []astRedirect
}

func (_ astSimple) isCommand()   {}
func (_ astCompound) isCommand() {}
func (_ astIf) isCommand()       {}
func (_ astWhile) isCommand()    {}

func (c *astSimple) redirs() []astRedirect   { return c.rs }
func (c *astCompound) redirs() []astRedirect { return c.rs }
func (c *astIf) redirs() []astRedirect       { return c.rs }
func (c *astWhile) redirs() []astRedirect    { return c.rs }

func (c *astSimple) setRedirs(rs []astRedirect)   { c.rs = rs }
func (c *astCompound) setRedirs(rs []astRedirect) { c.rs = rs }
func (c *astIf) setRedirs(rs []astRedirect)       { c.rs = rs }
func (c *astWhile) setRedirs(rs []astRedirect)    { c.rs = rs }

type astRedirect struct {
	kind redirKind
	file astValue
}

type redirKind int

const (
	redirAppend redirKind = iota
	redirClob
	redirRead
	redirWrite
)

func newRedir(k tokenKind) astRedirect {
	switch k {
	case tokAppend:
		return astRedirect{kind: redirAppend}
	case tokClobber:
		return astRedirect{kind: redirClob}
	case tokRead:
		return astRedirect{kind: redirRead}
	case tokWrite:
		return astRedirect{kind: redirWrite}
	}
	panic("unreachable")
}

type astValue interface {
	toStrings(ctx context) ([]string, commandResult)
}

type astArgument string

func (a astArgument) toStrings(_ context) ([]string, commandResult) {
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

type astString string

func (s astString) toStrings(_ context) ([]string, commandResult) {
	return []string{string(s)}, nil
}

type varRefKind int

const (
	vrExpand varRefKind = iota
	vrFlatten
	vrLength
)

type astVarRef struct {
	ident   string
	kind    varRefKind
	indices []astValue
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

func (vr astVarRef) toStrings(ctx context) ([]string, commandResult) {
	xs, ok := ctx.scope[vr.ident]
	if !ok {
		xs, ok = varMap[vr.ident]
	}
	if !ok {
		xs = []string{os.Getenv(vr.ident)}
	}

	if vr.indices != nil {
		ys := make([]string, 0, len(xs))
		for _, i := range vr.indices {
			ss, err := i.toStrings(ctx)
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

	switch vr.kind {
	case vrFlatten:
		xs = []string{strings.Join(xs, " ")}
	case vrLength:
		xs = []string{strconv.Itoa(len(xs))}
	}
	return xs, nil
}

func newVarRef(t token) astVarRef {
	vr := astVarRef{ident: t.val}
	switch t.kind {
	case tokVarFlat:
		vr.kind = vrFlatten
	case tokVarLen:
		vr.kind = vrLength
	}
	return vr
}

type astConcat struct {
	lhs, rhs astValue
}

func (c astConcat) toStrings(ctx context) ([]string, commandResult) {
	xs, err := c.lhs.toStrings(ctx)
	if err != nil {
		return []string{}, err
	}
	ys, err := c.rhs.toStrings(ctx)
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

type astList []astValue

func (l astList) toStrings(ctx context) ([]string, commandResult) {
	xs := make([]string, 0, len(l))
	for _, x := range l {
		ys, err := x.toStrings(ctx)
		if err != nil {
			return []string{}, err
		}
		xs = append(xs, ys...)
	}
	return xs, nil
}

type astProcSub struct {
	body []astTopLevel
}

func (ps astProcSub) toStrings(ctx context) ([]string, commandResult) {
	var out bytes.Buffer
	ctx.out = &out

	if res := execTopLevels(ps.body, ctx); cmdFailed(res) {
		return nil, res
	}

	s := strings.TrimRightFunc(out.String(), unicode.IsSpace)
	return []string{s}, nil
}

type astProcRedir struct {
	kind procRedirKind
	body []astTopLevel
	r, w *os.File
}

func (pr *astProcRedir) toStrings(ctx context) ([]string, commandResult) {
	xs := make([]string, 0, 2)
	r, w, err := os.Pipe()
	if err != nil {
		return []string{}, errInternal{err}
	}
	pr.r = r
	pr.w = w

	if pr.is(procRead) {
		ctx.out = w
		xs = append(xs, devFd(r))
	}
	if pr.is(procWrite) {
		ctx.in = r
		xs = append(xs, devFd(w))
	}

	go func() {
		_ = execTopLevels(pr.body, ctx)
		if pr.is(procRead) {
			w.Close()
		}
		if pr.is(procWrite) {
			r.Close()
		}
	}()

	return xs, nil
}

func (pr astProcRedir) Close() error {
	var e1, e2 error
	if pr.is(procRead) {
		e1 = pr.r.Close()
	}
	if pr.is(procWrite) {
		e2 = pr.w.Close()
	}
	return errors.Join(e1, e2)
}

func (pr astProcRedir) openFiles() []*os.File {
	var xs []*os.File
	if pr.is(procRead) {
		xs = append(xs, pr.r)
	}
	if pr.is(procWrite) {
		xs = append(xs, pr.w)
	}
	return xs
}

func devFd(f *os.File) string {
	return fmt.Sprintf("/dev/fd/%d", f.Fd())
}

func (pr astProcRedir) is(t procRedirKind) bool {
	return pr.kind&t != 0
}

type procRedirKind int

const (
	procRead procRedirKind = 1 << iota
	procWrite
)

type astBinaryOp int

const (
	binAnd astBinaryOp = iota
	binOr
)
