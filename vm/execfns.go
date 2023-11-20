package vm

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"

	"git.sr.ht/~mango/andy/builtin"
)

const appendFlags = os.O_APPEND | os.O_CREATE | os.O_WRONLY

func (vm *Vm) execCmdList(cl CommandList, ctx context) commandResult {
	if cl.Lhs == nil {
		return vm.execPipeline(cl.Rhs, ctx)
	}

	res := vm.execCmdList(*cl.Lhs, ctx)
	ec := res.ExitCode()

	if cl.Op == LAnd && ec == 0 || cl.Op == LOr && ec != 0 {
		return vm.execPipeline(cl.Rhs, ctx)
	}

	return res
}

func (vm *Vm) execPipeline(pl Pipeline, ctx context) commandResult {
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
			c <- vm.execCommand(cmd, ctx)
			wg.Done()
		}(cmd)
	}

	wg.Wait()
	close(c)

	for res := range c {
		if res.ExitCode() != 0 {
			return res
		}
	}

	return errExitCode(0)
}

func (vm *Vm) execCommand(cmd Command, ctx context) commandResult {
	switch cmd.In() {
	case nil:
		cmd.SetIn(ctx.in)
	case ctx.in:
	default:
		defer cmd.In().Close()
	}

	switch cmd.Out() {
	case nil:
		cmd.SetOut(ctx.out)
	case ctx.out:
	default:
		defer cmd.Out().Close()
	}

	switch cmd.Err() {
	case nil:
		cmd.SetErr(ctx.err)
	case ctx.err:
	default:
		defer cmd.Err().Close()
	}

	switch cmd.(type) {
	case *Simple:
		return vm.execSimple(cmd.(*Simple))
	case *Compound:
		return vm.execCompound(cmd.(*Compound))
	case *If:
		return vm.execIf(cmd.(*If))
	case *While:
		return vm.execWhile(cmd.(*While))
	}
	panic("unreachable")
}

func (vm *Vm) execWhile(cmd *While) commandResult {
	ctx := context{cmd.In(), cmd.Out(), cmd.Err()}
	for {
		res := vm.execCmdList(cmd.Cond, ctx)
		switch ec, ok := res.(errExitCode); {
		case !ok:
			return res
		case ec != 0:
			return errExitCode(0)
		}

		if res := vm.execCmdList(cmd.Body, ctx); res.ExitCode() != 0 {
			return res
		}
	}
}

func (vm *Vm) execIf(cmd *If) commandResult {
	ctx := context{cmd.In(), cmd.Out(), cmd.Err()}
	res := vm.execCmdList(cmd.Cond, ctx)
	switch ec, ok := res.(errExitCode); {
	case !ok:
		return res
	case ec == 0:
		return vm.execCmdList(cmd.Body, ctx)
	case cmd.Else != nil:
		return vm.execCmdList(*cmd.Else, ctx)
	}
	return errExitCode(0)
}

func (vm *Vm) execCompound(cmd *Compound) commandResult {
	ctx := context{cmd.In(), cmd.Out(), cmd.Err()}
	for _, cl := range cmd.Cmds {
		if res := vm.execCmdList(cl, ctx); res.ExitCode() != 0 {
			return res
		}
	}
	return errExitCode(0)
}

func (vm *Vm) execSimple(cmd *Simple) commandResult {
	args := make([]string, 0, cap(cmd.Args))
	for _, v := range cmd.Args {
		ss, err := v.ToStrings()
		if err != nil {
			return err
		}
		args = append(args, ss...)
	}

	c := exec.Command(args[0], args[1:]...)
	c.Stdin, c.Stdout, c.Stderr = cmd.In(), cmd.Out(), cmd.Err()

	for _, r := range cmd.Redirs {
		var name string
		switch r.File.(type) {
		case Argument:
			name = string(r.File.(Argument))

			switch {
			case r.Type == RedirRead && name == "_":
				name = os.DevNull
			case r.Type == RedirWrite && name == "_":
				r.Type = RedirClob
				name = os.DevNull
			}
		default:
			xs, err := r.File.ToStrings()
			if err != nil {
				return err
			}
			if len(xs) > 1 {
				return errExpected{
					want: "filename",
					got:  fmt.Sprintf("%d filesnames", len(xs)),
				}
			}
			name = xs[0]
		}

		switch r.Type {
		case RedirAppend:
			fp, err := os.OpenFile(name, appendFlags, 0666)
			if err != nil {
				return errInternal{err}
			}
			defer fp.Close()
			c.Stdout = fp
		case RedirClob:
			fp, err := os.Create(name)
			if err != nil {
				return errInternal{err}
			}
			defer fp.Close()
			c.Stdout = fp
		case RedirRead:
			fp, err := os.Open(name)
			if err != nil {
				return errInternal{err}
			}
			defer fp.Close()
			c.Stdin = fp
		case RedirWrite:
			_, err := os.Stat(name)
			switch {
			case errors.Is(err, os.ErrNotExist):
				fp, err := os.Create(name)
				if err != nil {
					return errInternal{err}
				}
				defer fp.Close()
				c.Stdout = fp
			case err != nil:
				return errFileOp{"stat", name, err}
			default: // File exists
				return errClobber{name}
			}
		default:
			panic("unreachable")
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
