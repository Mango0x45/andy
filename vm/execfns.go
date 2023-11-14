package vm

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"

	"git.sr.ht/~mango/andy/ast"
)

const appendFlags = os.O_APPEND | os.O_CREATE | os.O_WRONLY

func (vm *Vm) execCmdList(cl ast.CommandList, ctx context) commandResult {
	if cl.Lhs == nil {
		return vm.execPipeline(cl.Rhs, ctx)
	}

	res := vm.execCmdList(*cl.Lhs, ctx)
	ec := res.ExitCode()

	if cl.Op == ast.LAnd && ec == 0 || cl.Op == ast.LOr && ec != 0 {
		return vm.execPipeline(cl.Rhs, ctx)
	}

	return res
}

func (vm *Vm) execPipeline(pl ast.Pipeline, ctx context) commandResult {
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
		go func(cmd ast.Command) {
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

func (vm *Vm) execCommand(cmd ast.Command, ctx context) commandResult {
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
	case *ast.If:
		return vm.execIf(cmd.(*ast.If))
	case *ast.Compound:
		return vm.execCompound(cmd.(*ast.Compound))
	case *ast.Simple:
		return vm.execSimple(cmd.(*ast.Simple))
	}
	panic("unreachable")
}

func (vm *Vm) execIf(cmd *ast.If) commandResult {
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

func (vm *Vm) execCompound(cmd *ast.Compound) commandResult {
	ctx := context{cmd.In(), cmd.Out(), cmd.Err()}
	for _, cl := range cmd.Cmds {
		if res := vm.execCmdList(cl, ctx); res.ExitCode() != 0 {
			return res
		}
	}
	return errExitCode(0)
}

func (vm *Vm) execSimple(cmd *ast.Simple) commandResult {
	args := make([]string, 0, cap(cmd.Args))
	for _, v := range cmd.Args {
		args = append(args, v.ToStrings()...)
	}

	c := exec.Command(args[0], args[1:]...)
	c.Stdin, c.Stdout, c.Stderr = cmd.In(), cmd.Out(), cmd.Err()

	for _, r := range cmd.Redirs {
		var name string
		switch r.File.(type) {
		case ast.Argument:
			name = string(r.File.(ast.Argument))

			switch {
			case r.Type == ast.RedirRead && name == "_":
				name = os.DevNull
			case r.Type == ast.RedirWrite && name == "_":
				r.Type = ast.RedirClob
				name = os.DevNull
			}
		default:
			xs := r.File.ToStrings()
			if len(xs) > 1 {
				return errExpected{
					want: "filename",
					got:  fmt.Sprintf("%d filesnames", len(xs)),
				}
			}
			name = xs[0]
		}

		switch r.Type {
		case ast.RedirAppend:
			fp, err := os.OpenFile(name, appendFlags, 0666)
			if err != nil {
				return errInternal{err}
			}
			defer fp.Close()
			c.Stdout = fp
		case ast.RedirClob:
			fp, err := os.Create(name)
			if err != nil {
				return errInternal{err}
			}
			defer fp.Close()
			c.Stdout = fp
		case ast.RedirRead:
			fp, err := os.Open(name)
			if err != nil {
				return errInternal{err}
			}
			defer fp.Close()
			c.Stdin = fp
		case ast.RedirWrite:
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

	if f, ok := builtins[c.Args[0]]; ok {
		return f(c)
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
