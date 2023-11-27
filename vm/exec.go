package vm

import (
	"cmp"
	"errors"
	"fmt"
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

	for i := range pl[:n-1] {
		r, w, err := os.Pipe()
		if err != nil {
			return errInternal{err}
		}

		pl[i+0].SetOut(w)
		pl[i+1].SetIn(r)
	}

	c := make(chan commandResult, n)
	wg := sync.WaitGroup{}
	wg.Add(n)

	for _, cmd := range pl {
		go func(cmd Command) {
			c <- execCommand(cmd, ctx)
			wg.Done()
		}(cmd)
	}

	wg.Wait()
	close(c)

	for res := range c {
		if failed(res) {
			return res
		}
	}

	return errExitCode(0)
}

func execCommand(cmd Command, ctx context) commandResult {
	switch cmd.In() {
	case nil:
		cmd.SetIn(ctx.in)
	case ctx.in:
	default:
		if f, ok := cmd.In().(*os.File); ok {
			defer f.Close()
		}
	}

	switch cmd.Out() {
	case nil:
		cmd.SetOut(ctx.out)
	case ctx.out:
	default:
		if f, ok := cmd.Out().(*os.File); ok {
			defer f.Close()
		}
	}

	switch cmd.Err() {
	case nil:
		cmd.SetErr(ctx.err)
	case ctx.err:
	default:
		if f, ok := cmd.Err().(*os.File); ok {
			defer f.Close()
		}
	}

	for _, re := range cmd.Redirs() {
		var name string
		switch re.File.(type) {
		case Argument:
			name = string(re.File.(Argument))

			switch {
			case re.Type == RedirRead && name == "_":
				name = os.DevNull
			case re.Type == RedirWrite && name == "_":
				re.Type = RedirClob
				name = os.DevNull
			}
		default:
			xs, err := re.File.ToStrings(ctx)
			if err != nil {
				return err
			}
			if pr, ok := re.File.(*ProcRedir); ok {
				defer pr.Close()
			}
			if len(xs) > 1 {
				return errExpected{
					want: "filename",
					got:  fmt.Sprintf("%d filesnames", len(xs)),
				}
			}
			name = xs[0]
		}

		switch re.Type {
		case RedirAppend:
			fp, err := os.OpenFile(name, appendFlags, 0666)
			if err != nil {
				return errInternal{err}
			}
			defer fp.Close()
			cmd.SetOut(fp)
		case RedirClob:
			fp, err := os.Create(name)
			if err != nil {
				return errInternal{err}
			}
			defer fp.Close()
			cmd.SetOut(fp)
		case RedirRead:
			fp, err := os.Open(name)
			if err != nil {
				return errInternal{err}
			}
			defer fp.Close()
			cmd.SetIn(fp)
		case RedirWrite:
			_, err := os.Stat(name)
			switch {
			case errors.Is(err, os.ErrNotExist):
				fp, err := os.Create(name)
				if err != nil {
					return errInternal{err}
				}
				defer fp.Close()
				cmd.SetOut(fp)
			case err != nil:
				return errFileOp{"stat", name, err}
			default: // File exists
				return errClobber{name}
			}
		default:
			panic("unreachable")
		}
	}

	switch cmd.(type) {
	case *Simple:
		return execSimple(cmd.(*Simple), ctx)
	case *Compound:
		return execCompound(cmd.(*Compound))
	case *If:
		return execIf(cmd.(*If))
	case *While:
		return execWhile(cmd.(*While))
	}
	panic("unreachable")
}

func execWhile(cmd *While) commandResult {
	ctx := context{cmd.In(), cmd.Out(), cmd.Err()}
	for {
		res := execCmdList(cmd.Cond, ctx)
		switch ec, ok := res.(errExitCode); {
		case !ok:
			return res
		case failed(ec):
			return errExitCode(0)
		}

		for _, cl := range cmd.Body {
			if res := execCmdList(cl, ctx); failed(res) {
				return res
			}
		}
	}
}

func execIf(cmd *If) commandResult {
	ctx := context{cmd.In(), cmd.Out(), cmd.Err()}
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

func execCompound(cmd *Compound) commandResult {
	ctx := context{cmd.In(), cmd.Out(), cmd.Err()}
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
	c.Stdin, c.Stdout, c.Stderr = cmd.In(), cmd.Out(), cmd.Err()

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
