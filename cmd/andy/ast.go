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

	"git.sr.ht/~mango/andy/pkg/stringsx"
)

// See grammar.ebnf in the project root for details

type astProgram = []astTopLevel

type astTopLevel interface {
	isTopLevel()
}

type astFuncDef struct {
	args astList
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
	args astList
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

type astFor struct {
	bind astValue
	vals astList
	body []astTopLevel
	rs   []astRedirect
}

func (_ astSimple) isCommand()   {}
func (_ astCompound) isCommand() {}
func (_ astIf) isCommand()       {}
func (_ astWhile) isCommand()    {}
func (_ astFor) isCommand()      {}

func (c *astSimple) redirs() []astRedirect   { return c.rs }
func (c *astCompound) redirs() []astRedirect { return c.rs }
func (c *astIf) redirs() []astRedirect       { return c.rs }
func (c *astWhile) redirs() []astRedirect    { return c.rs }
func (c *astFor) redirs() []astRedirect      { return c.rs }

func (c *astSimple) setRedirs(rs []astRedirect)   { c.rs = rs }
func (c *astCompound) setRedirs(rs []astRedirect) { c.rs = rs }
func (c *astIf) setRedirs(rs []astRedirect)       { c.rs = rs }
func (c *astWhile) setRedirs(rs []astRedirect)    { c.rs = rs }
func (c *astFor) setRedirs(rs []astRedirect)      { c.rs = rs }

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
	redirSockRead
	redirSockWrite
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
	io.Closer
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
	ident   astValue
	repl    astValue
	kind    varRefKind
	indices astList
}

func stoi(s string) (int, commandResult) {
	n, err := strconv.Atoi(s)
	err, ok := err.(*strconv.NumError)
	if !ok {
		return n, nil
	}

	var es string
	switch {
	case errors.Is(err, strconv.ErrRange):
		es = fmt.Sprintf("index ‘%s’ is out of range; what are you even doing?", s)
	case errors.Is(err, strconv.ErrSyntax):
		es = fmt.Sprintf("‘%s’ isn’t a valid index", s)
	}

	return 0, errInternal{errors.New(es)}
}

func getIndexRange(s string, n int) (int, int, commandResult) {
	b, a, f := strings.Cut(s, "..")
	i, err := stoi(b)
	if cmdFailed(err) {
		return 0, 0, err
	}

	var j int
	switch {
	case f && a != "":
		j, err = stoi(a)
		if cmdFailed(err) {
			return 0, 0, err
		}
	case f:
		j = n
	default:
		j = i + 1
	}

	return i, j, nil
}

func (vr astVarRef) toStrings(ctx context) ([]string, commandResult) {
	ss, res := vr.ident.toStrings(ctx)
	defer vr.ident.Close()
	if cmdFailed(res) {
		return nil, res
	}
	if len(ss) > 2 {
		return nil, errInternal{errors.New("not implemented")}
	} else if len(ss) == 0 {
		return []string{}, nil
	}

	ident := ss[0]
	xs, ok := ctx.scope[ident]
	if !ok {
		xs, ok = globalVariableMap[ident]
	}
	if !ok {
		if x, ok := os.LookupEnv(ident); ok {
			xs = []string{x}
		}
	}

	if vr.repl != nil && (len(xs) == 0 || xs[0] == "") {
		var res commandResult
		xs, res = vr.repl.toStrings(ctx)
		defer vr.repl.Close()
		if cmdFailed(res) {
			return nil, res
		}
	}

	if vr.indices != nil {
		defer vr.indices.Close()
		ss, res := vr.indices.toStrings(ctx)
		if cmdFailed(res) {
			return nil, res
		}

		n := len(xs)
		ys := make([]string, 0, n)
		for _, s := range ss {
			i, j, res := getIndexRange(s, n)
			if cmdFailed(res) {
				return nil, res
			}

			I, J := i, j
			if I < 0 {
				I += n
			}
			if J < 0 {
				J += n
			}

			if i < j {
				switch {
				case I < 0, I >= n:
					return nil, errInvalidIndex{i, n}
				case J < 0, J > n:
					return nil, errInvalidIndex{j, n}
				}

				for k := i; k < j; k++ {
					k := k
					if k < 0 {
						k += n
					}
					ys = append(ys, xs[k])
				}
			} else {
				switch {
				case I < 0, I > n:
					return nil, errInvalidIndex{i, n}
				case J < 0, J >= n:
					return nil, errInvalidIndex{j, n}
				}

				for k := i - 1; k >= j; k-- {
					k := k
					if k < 0 {
						k += n
					}
					ys = append(ys, xs[k])
				}
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
	var vr astVarRef
	if t.val == "" {
		vr.ident = nil
	} else {
		vr.ident = astArgument(t.val)
	}
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
	xs, res := c.lhs.toStrings(ctx)
	if cmdFailed(res) {
		return nil, res
	}
	ys, res := c.rhs.toStrings(ctx)
	if cmdFailed(res) {
		return nil, res
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
		ys, res := x.toStrings(ctx)
		if cmdFailed(res) {
			return nil, res
		}
		xs = append(xs, ys...)
	}
	return xs, nil
}

type astProcSub struct {
	seps astList
	body []astTopLevel
}

func (ps astProcSub) toStrings(ctx context) ([]string, commandResult) {
	seps, res := ps.seps.toStrings(ctx)
	defer ps.seps.Close()
	if cmdFailed(res) {
		return nil, res
	}

	var out bytes.Buffer
	ctx.out = &out

	if res := execTopLevels(ps.body, ctx); cmdFailed(res) {
		return nil, res
	}

	s := strings.TrimSuffix(out.String(), "\n")
	return stringsx.SplitMulti(s, seps), nil
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

func (_ astArgument) Close() error { return nil }
func (_ astString) Close() error   { return nil }
func (_ astVarRef) Close() error   { return nil }
func (_ astProcSub) Close() error  { return nil }
func (c astConcat) Close() error {
	return errors.Join(c.lhs.Close(), c.rhs.Close())
}
func (xs astList) Close() error {
	errs := make([]error, len(xs))
	for i, x := range xs {
		errs[i] = x.Close()
	}
	return errors.Join(errs...)
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

type astBinaryOp int

const (
	binAnd astBinaryOp = iota
	binOr
)
