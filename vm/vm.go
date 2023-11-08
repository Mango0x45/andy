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

var (
	CrashOnError = false
	Interactive  = false

	status = 0
)

func Exec(prog ast.Program) {
	for _, cl := range prog {
		ret := execCmdList(cl)
		status = ret.ExitCode()
		if _, ok := ret.(shellError); ok {
			fmt.Fprintf(os.Stderr, "andy: %s\n", ret)
			if CrashOnError {
				os.Exit(1)
			}
		}
	}
}

func execCmdList(cl ast.CommandList) commandResult {
	if cl.Lhs == nil {
		return execPipeline(cl.Rhs)
	}

	res := execCmdList(*cl.Lhs)
	ec := res.ExitCode()

	if cl.Op == ast.LAnd && ec == 0 || cl.Op == ast.LOr && ec != 0 {
		return execPipeline(cl.Rhs)
	}

	return res
}

func execPipeline(pl ast.Pipeline) commandResult {
	res := errExitCode(0)
	wg := sync.WaitGroup{}

	for i := range pl[:len(pl)-1] {
		r, w, err := os.Pipe()
		if err != nil {
			return errInternal{err}
		}

		pl[i+0].Out = w
		pl[i+1].In = r

		go func(i int) {
			wg.Add(1)
			if ec := execSimple(pl[i]).ExitCode(); res == 0 && ec != 0 {
				res = errExitCode(ec)
			}
			wg.Done()
		}(i)
	}

	if ec := execSimple(pl[len(pl)-1]).ExitCode(); res == 0 && ec != 0 {
		res = errExitCode(ec)
	}
	wg.Wait()
	return res
}

func execSimple(cmd ast.Simple) commandResult {
	if cmd.In != os.Stdin {
		defer cmd.In.Close()
	}
	if cmd.Out != os.Stdout {
		defer cmd.Out.Close()
	}
	if cmd.Err != os.Stderr {
		defer cmd.Err.Close()
	}

	args := make([]string, 0, cap(cmd.Args))
	for _, v := range cmd.Args {
		switch v.(type) {
		case ast.Argument:
			args = append(args, string(v.(ast.Argument)))
		case ast.String:
			args = append(args, string(v.(ast.String)))
		default:
			panic("unreachable")
		}
	}

	c := exec.Command(args[0], args[1:]...)
	c.Stdin, c.Stdout, c.Stderr = cmd.In, cmd.Out, cmd.Err

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
		case ast.String:
			name = string(r.File.(ast.String))
		default:
			panic("unreachable")
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
		return err.(*exec.ExitError)
	default:
		return errInternal{err}
	}
}
