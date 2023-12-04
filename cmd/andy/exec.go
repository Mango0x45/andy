package main

import (
	"cmp"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"slices"
	"sync"
)

const appendFlags = os.O_APPEND | os.O_CREATE | os.O_WRONLY

func execTopLevels(tls []astTopLevel, ctx context) commandResult {
	for _, tl := range tls {
		if res := execTopLevel(tl, ctx); cmdFailed(res) {
			return res
		}
	}
	return errExitCode(0)
}

func execTopLevel(tl astTopLevel, ctx context) commandResult {
	var res commandResult
	switch tl.(type) {
	case astFuncDef:
		res = execFuncDef(tl.(astFuncDef), ctx)
	case astCommandList:
		res = execCmdList(tl.(astCommandList), ctx)
	}

	if cmdFailed(res) {
		return res
	}
	return errExitCode(0)
}

func execFuncDef(fd astFuncDef, ctx context) commandResult {
	args := make([]string, 0, len(fd.args))
	for _, a := range fd.args {
		s, err := a.toStrings(ctx)
		if err != nil {
			return err
		}
		args = append(args, s...)
	}

	if len(args) == 0 {
		return errInternal{errors.New("attempted to define function without a name")}
	}

	f := function{args: args[1:], body: fd.body}
	globalFuncMap[args[0]] = f

	return errExitCode(0)
}

func execCmdList(cl astCommandList, ctx context) commandResult {
	if cl.lhs == nil {
		return execPipeline(cl.rhs, ctx)
	}

	res := execCmdList(*cl.lhs, ctx)
	ec := res.ExitCode()

	if cl.op == binAnd && ec == 0 || cl.op == binOr && ec != 0 {
		return execPipeline(cl.rhs, ctx)
	}

	return res
}

func execPipeline(pl astPipeline, ctx context) commandResult {
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
		pl[i].add(w)
		cs[i+1].in = r
		pl[i+1].add(r)
	}

	var wg sync.WaitGroup
	wg.Add(n - 1)

	// TODO: Go 1.22 fixed for-loops
	for i := range pl[:len(pl)-1] {
		go func(cc astCleanCommand, ctx context) {
			execCommand(cc, ctx)
			wg.Done()
		}(pl[i], cs[i])
	}

	if res := execCommand(pl[n-1], cs[n-1]); cmdFailed(res) {
		return res
	}
	wg.Wait()

	return errExitCode(0)
}

func execCommand(cc astCleanCommand, ctx context) commandResult {
	for _, re := range cc.cmd.redirs() {
		var name string

		ss, res := re.file.toStrings(ctx)
		if res != nil {
			return res
		}
		if len(ss) > 1 {
			return errExpected{
				want: "filename",
				got:  fmt.Sprintf("%d filesnames", len(ss)),
			}
		}
		name = ss[0]

		switch re.file.(type) {
		case astArgument:
			switch {
			case re.kind == redirRead && name == "_":
				name = os.DevNull
			case re.kind == redirWrite && name == "_":
				re.kind = redirClob
				name = os.DevNull
			case re.kind == redirRead:
				info, err := os.Stat(name)
				switch {
				case err != nil:
					return errInternal{err}
				case err == nil && info.Mode()&os.ModeSocket != 0:
					re.kind = redirSockRead
				}
			case re.kind == redirWrite:
				info, err := os.Stat(name)
				switch {
				case err != nil && !errors.Is(err, os.ErrNotExist):
					return errInternal{err}
				case err == nil && info.Mode()&os.ModeSocket != 0:
					re.kind = redirSockWrite
				case err == nil && !info.Mode().IsRegular():
					re.kind = redirClob
				}
			}
		}

		var f io.ReadWriteCloser
		var err error
		switch re.kind {
		case redirAppend:
			f, err = os.OpenFile(name, appendFlags, 0666)
		case redirClob:
			f, err = os.Create(name)
		case redirRead:
			f, err = os.Open(name)
		case redirWrite:
			_, err := os.Stat(name)
			switch {
			case errors.Is(err, os.ErrNotExist):
				f, err = os.Create(name)
			case err != nil:
				return errFileOp{"stat", name, err}
			default: // File exists
				return errClobber{name}
			}
		case redirSockRead, redirSockWrite:
			f, err = net.Dial("unix", name)
		}
		if err != nil {
			return errInternal{err}
		}

		cc.add(f)
		switch re.kind {
		case redirAppend, redirClob, redirWrite, redirSockWrite:
			ctx.out = f
		case redirRead, redirSockRead:
			ctx.in = f
		}
	}

	defer cc.cleanup()
	switch cc.cmd.(type) {
	case *astSimple:
		return execSimple(cc.cmd.(*astSimple), ctx)
	case *astCompound:
		return execCompound(cc.cmd.(*astCompound), ctx)
	case *astIf:
		return execIf(cc.cmd.(*astIf), ctx)
	case *astWhile:
		return execWhile(cc.cmd.(*astWhile), ctx)
	}
	panic("unreachable")
}

func execWhile(cmd *astWhile, ctx context) commandResult {
	for {
		res := execCmdList(cmd.cond, ctx)
		switch ec, ok := res.(errExitCode); {
		case !ok:
			return res
		case cmdFailed(ec):
			return errExitCode(0)
		}

		if res := execTopLevels(cmd.body, ctx); cmdFailed(res) {
			return res
		}
	}
}

func execIf(cmd *astIf, ctx context) commandResult {
	res := execCmdList(cmd.cond, ctx)
	if _, ok := res.(errExitCode); !ok {
		return res
	}

	var cmds []astTopLevel
	if cmdFailed(res) {
		cmds = cmd.else_
	} else {
		cmds = cmd.body
	}
	return execTopLevels(cmds, ctx)
}

func execCompound(cmd *astCompound, ctx context) commandResult {
	return execTopLevels(cmd.cmds, ctx)
}

func execSimple(cmd *astSimple, ctx context) commandResult {
	args := make([]string, 0, cap(cmd.args))
	extras := []*os.File{}

	for _, v := range cmd.args {
		ss, err := v.toStrings(ctx)
		if err != nil {
			return err
		}
		args = append(args, ss...)

		// Cringe code duplication, but it works
		switch v.(type) {
		case *astProcRedir:
			pr := v.(*astProcRedir)
			extras = append(extras, pr.openFiles()...)
		case astList:
			for _, x := range v.(astList) {
				if pr, ok := x.(*astProcRedir); ok {
					extras = append(extras, pr.openFiles()...)
				}
			}
		}
		defer v.Close()
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

	if f, ok := globalFuncMap[c.Args[0]]; ok {
		ctx.scope = copyMap(ctx.scope)
		args := c.Args[1:]
		for i, a := range f.args {
			if i >= len(args) {
				break
			}
			ctx.scope[a] = []string{args[i]}
		}
		if len(args) > len(f.args) {
			ctx.scope["_"] = args[len(f.args):]
		}
		return execTopLevels(f.body, ctx)
	}
	if f, ok := builtins[c.Args[0]]; ok {
		return errExitCode(f(c, ctx))
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
