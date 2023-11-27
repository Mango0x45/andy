package vm

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
	"sync"
	"unicode"

	"git.sr.ht/~mango/andy/builtin"
)

const appendFlags = os.O_APPEND | os.O_CREATE | os.O_WRONLY

func (vm *Vm) execCmdLists(cls []CommandList, ctx context) commandResult {
	for _, cl := range cls {
		res := vm.execCmdList(cl, ctx)
		if res.ExitCode() != 0 {
			return res
		}
	}
	return errExitCode(0)
}

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
		case ProcSub:
			var out bytes.Buffer
			ctx := ctx
			ctx.out = &out

			res := vm.execCmdLists(re.File.(ProcSub).Body, ctx)
			if res.ExitCode() != 0 {
				return res
			}

			s := strings.TrimRightFunc(out.String(), unicode.IsSpace)
			name = s
		case ProcRedir:
			pr := re.File.(ProcRedir)
			r, w, err := os.Pipe()
			if err != nil {
				return errInternal{err}
			}

			ctx := ctx
			if pr.Is(ProcRead) && pr.Is(ProcWrite) {
				var preposition string
				if re.Type == RedirRead {
					preposition = "from"
				} else {
					preposition = "to"
				}
				s := fmt.Sprintf("redirect %s read+write process substitution", preposition)
				return errUnsupported(s)
			}
			if pr.Is(ProcRead) {
				ctx.out = w
				name = devFd(w)
				defer r.Close()
			} else {
				ctx.in = r
				name = devFd(r)
				defer w.Close()
			}

			// TODO: go 1.22 fixed range loops
			go func(pr ProcRedir) {
				res := vm.execCmdLists(pr.Body, ctx)
				if res.ExitCode() != 0 {
					panic("TODO")
				}
				if pr.Is(ProcRead) {
					w.Close()
				}
				if pr.Is(ProcWrite) {
					r.Close()
				}
			}(pr)
		default:
			xs, err := re.File.ToStrings()
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
		return vm.execSimple(cmd.(*Simple), ctx)
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

		for _, cl := range cmd.Body {
			if res := vm.execCmdList(cl, ctx); res.ExitCode() != 0 {
				return res
			}
		}
	}
}

func (vm *Vm) execIf(cmd *If) commandResult {
	ctx := context{cmd.In(), cmd.Out(), cmd.Err()}
	res := vm.execCmdList(cmd.Cond, ctx)
	ec, ok := res.(errExitCode)
	if !ok {
		return res
	}

	var cmds []CommandList
	if ec == 0 {
		cmds = cmd.Body
	} else {
		cmds = cmd.Else
	}
	for _, cl := range cmds {
		if res := vm.execCmdList(cl, ctx); res.ExitCode() != 0 {
			return res
		}
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

func (vm *Vm) execSimple(cmd *Simple, ctx context) commandResult {
	args := make([]string, 0, cap(cmd.Args))
	extras := []*os.File{}

	for _, v := range cmd.Args {
		switch v.(type) {
		case ProcSub:
			var out bytes.Buffer
			ctx := ctx
			ctx.out = &out

			res := vm.execCmdLists(v.(ProcSub).Body, ctx)
			if res.ExitCode() != 0 {
				return res
			}

			s := strings.TrimRightFunc(out.String(), unicode.IsSpace)
			args = append(args, s)
		case ProcRedir:
			pr := v.(ProcRedir)
			r, w, err := os.Pipe()
			if err != nil {
				return errInternal{err}
			}

			ctx := ctx
			if pr.Is(ProcRead) {
				ctx.out = w
				extras = append(extras, r)
				args = append(args, devFd(r))
				defer r.Close()
			}
			if pr.Is(ProcWrite) {
				ctx.in = r
				extras = append(extras, w)
				args = append(args, devFd(w))
				defer w.Close()
			}

			// TODO: go 1.22 fixed range loops
			go func(pr ProcRedir) {
				res := vm.execCmdLists(pr.Body, ctx)
				if res.ExitCode() != 0 {
					panic("TODO")
				}
				if pr.Is(ProcRead) {
					w.Close()
				}
				if pr.Is(ProcWrite) {
					r.Close()
				}
			}(pr)
		default:
			ss, err := v.ToStrings()
			if err != nil {
				return err
			}
			args = append(args, ss...)
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
			return int(a.Fd() - b.Fd())
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

func devFd(f *os.File) string {
	return fmt.Sprintf("/dev/fd/%d", f.Fd())
}
