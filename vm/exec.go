package vm

import (
	"cmp"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"slices"
	"sync"

	"git.sr.ht/~mango/andy/builtin"
)

const appendFlags = os.O_APPEND | os.O_CREATE | os.O_WRONLY

func execCmdLists(cls []CommandList, ctx context) commandResult {
	for _, cl := range cls {
		res := execCmdList(cl, ctx)
		if failed(res) {
			return res
		}
	}
	return errExitCode(0)
}

func execCmdList(cl CommandList, ctx context) commandResult {
	if cl.Lhs == nil {
		return execPipeline(cl.Rhs, ctx)
	}

	res := execCmdList(*cl.Lhs, ctx)
	ec := res.ExitCode()

	if cl.Op == LAnd && ec == 0 || cl.Op == LOr && ec != 0 {
		return execPipeline(cl.Rhs, ctx)
	}

	return res
}

func execPipeline(pl Pipeline, ctx context) commandResult {
	n := len(pl)
	cs := make([]context, n)

	for i := range cs {
		cs[i] = ctx
	}

	for i := range pl[:n-1] {
		r, w, err := os.Pipe()
		if err != nil {
			return errInternal{err}
		}

		cs[i].out = w
		pl[i].Add(w)
		cs[i+1].in = r
		pl[i+1].Add(r)
	}

	wg := sync.WaitGroup{}
	wg.Add(n - 1)

	// TODO: Go 1.22 fixed for-loops
	for i := range pl[:len(pl)-1] {
		go func(cc CleanCommand, ctx context) {
			execCommand(cc, ctx)
			wg.Done()
		}(pl[i], cs[i])
	}

	if res := execCommand(pl[n-1], cs[n-1]); failed(res) {
		return res
	}
	wg.Wait()

	return errExitCode(0)
}

func execCommand(cc CleanCommand, ctx context) commandResult {
	for _, re := range cc.Cmd.Redirs() {
		var name string

		ss, err := re.File.ToStrings(ctx)
		if err != nil {
			return err
		}
		if len(ss) > 1 {
			return errExpected{
				want: "filename",
				got:  fmt.Sprintf("%d filesnames", len(ss)),
			}
		}
		name = ss[0]

		switch re.File.(type) {
		case Argument:
			switch {
			case re.Type == RedirRead && name == "_":
				name = os.DevNull
			case re.Type == RedirWrite && name == "_":
				re.Type = RedirClob
				name = os.DevNull
			}
		case *ProcRedir:
			cc.Add(re.File.(*ProcRedir))
		}

		var f io.ReadWriteCloser
		switch re.Type {
		case RedirAppend:
			fp, err := os.OpenFile(name, appendFlags, 0666)
			if err != nil {
				return errInternal{err}
			}
			f = fp
		case RedirClob:
			fp, err := os.Create(name)
			if err != nil {
				return errInternal{err}
			}
			f = fp
		case RedirRead:
			fp, err := os.Open(name)
			if err != nil {
				return errInternal{err}
			}
			f = fp
		case RedirWrite:
			_, err := os.Stat(name)
			switch {
			case errors.Is(err, os.ErrNotExist):
				fp, err := os.Create(name)
				if err != nil {
					return errInternal{err}
				}
				f = fp
			case err != nil:
				return errFileOp{"stat", name, err}
			default: // File exists
				return errClobber{name}
			}
		default:
			panic("unreachable")
		}

		cc.Add(f)
		switch re.Type {
		case RedirAppend, RedirClob, RedirWrite:
			ctx.out = f
		case RedirRead:
			ctx.in = f
		}
	}

	defer cc.Cleanup()
	switch cc.Cmd.(type) {
	case *Simple:
		return execSimple(cc.Cmd.(*Simple), ctx)
	case *Compound:
		return execCompound(cc.Cmd.(*Compound), ctx)
	case *If:
		return execIf(cc.Cmd.(*If), ctx)
	case *While:
		return execWhile(cc.Cmd.(*While), ctx)
	}
	panic("unreachable")
}

func execWhile(cmd *While, ctx context) commandResult {
	for {
		res := execCmdList(cmd.Cond, ctx)
		switch ec, ok := res.(errExitCode); {
		case !ok:
			return res
		case failed(ec):
			return errExitCode(0)
		}

		if res := execCmdLists(cmd.Body, ctx); failed(res) {
			return res
		}
	}
}

func execIf(cmd *If, ctx context) commandResult {
	res := execCmdList(cmd.Cond, ctx)
	if _, ok := res.(errExitCode); !ok {
		return res
	}

	var cmds []CommandList
	if failed(res) {
		cmds = cmd.Else
	} else {
		cmds = cmd.Body
	}
	for _, cl := range cmds {
		if res := execCmdList(cl, ctx); failed(res) {
			return res
		}
	}
	return errExitCode(0)
}

func execCompound(cmd *Compound, ctx context) commandResult {
	for _, cl := range cmd.Cmds {
		if res := execCmdList(cl, ctx); failed(res) {
			return res
		}
	}
	return errExitCode(0)
}

func execSimple(cmd *Simple, ctx context) commandResult {
	args := make([]string, 0, cap(cmd.Args))
	extras := []*os.File{}

	for _, v := range cmd.Args {
		ss, err := v.ToStrings(ctx)
		if err != nil {
			return err
		}
		args = append(args, ss...)
		if pr, ok := v.(*ProcRedir); ok {
			extras = append(extras, pr.OpenFiles()...)
			defer pr.Close()
		}
	}

	// You might try to run the empty list
	if len(args) == 0 {
		return errExitCode(0)
	}

	c := exec.Command(args[0], args[1:]...)
	c.Stdin, c.Stdout, c.Stderr = ctx.in, ctx.out, ctx.err

	if len(extras) > 0 {
		maxFd := slices.MaxFunc(extras, func(a, b *os.File) int {
			return cmp.Compare(a.Fd(), b.Fd())
		}).Fd()
		c.ExtraFiles = make([]*os.File, maxFd)
		for _, e := range extras {
			c.ExtraFiles[e.Fd()-3] = e
		}
	}

	if f, ok := builtin.Commands[c.Args[0]]; ok {
		return errExitCode(f(c))
	}
	switch err := c.Run(); err.(type) {
	case nil:
		return errExitCode(0)
	case *exec.ExitError:
		return errExitCode(err.(*exec.ExitError).ExitCode())
	default:
		return errInternal{err}
	}
}
