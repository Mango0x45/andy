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

var CrashOnError = false

func Exec(prog ast.Program) {
	for _, cmd := range prog {
		err := execCommand(cmd, streams{os.Stdin, os.Stdout, os.Stderr})
		if err != nil {
			fmt.Fprintf(os.Stderr, "andy: %s\n", err)

			if CrashOnError {
				os.Exit(1)
			}
		}
	}
}

func execCommand(cmd ast.Command, s streams) error {
	switch cmd.(type) {
	case ast.Simple:
		return execSimple(cmd.(ast.Simple), s)
	case ast.Compound:
		return execCompound(cmd.(ast.Compound), s)
	}
	panic("unreachable")
}

func execSimple(cmd ast.Simple, s streams) error {
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
	c.Stdin, c.Stdout, c.Stderr = s.in, s.out, s.err

	for _, r := range cmd.Redirs {
		var name string

		switch r.File.(type) {
		case ast.Argument:
			name = string(r.File.(ast.Argument))

			switch {
			case r.Mode == ast.RedirRead && name == "_":
				name = os.DevNull
			case r.Mode == ast.RedirWrite && name == "!":
				r.Mode = ast.RedirWriteClob
				name = os.Stderr.Name()
			case r.Mode == ast.RedirWrite && name == "_":
				r.Mode = ast.RedirWriteClob
				name = os.DevNull
			}
		case ast.String:
			name = string(r.File.(ast.String))
		default:
			panic("unreachable")
		}

		switch r.Mode {
		case ast.RedirAppend:
			fp, err := os.OpenFile(name, appendFlags, 0666)
			if err != nil {
				return err
			}
			defer fp.Close()
			c.Stdout = fp
		case ast.RedirRead:
			fp, err := os.Open(name)
			if err != nil {
				return err
			}
			defer fp.Close()
			c.Stdin = fp
		case ast.RedirWrite:
			_, err := os.Stat(name)
			switch {
			case errors.Is(err, os.ErrNotExist):
				fp, err := os.Create(name)
				if err != nil {
					return err
				}
				defer fp.Close()
				c.Stdout = fp
			case err != nil:
				return errFileOp{"stat", name, err}
			default: // File exists
				return errClobber{name}
			}
		case ast.RedirWriteClob:
			fp, err := os.Create(name)
			if err != nil {
				return err
			}
			defer fp.Close()
			c.Stdout = fp
		default:
			panic("unreachable")
		}
	}

	if s.in != os.Stdin {
		defer s.in.Close()
	}
	if s.out != os.Stdout {
		defer s.out.Close()
	}
	if s.err != os.Stderr {
		defer s.err.Close()
	}

	if f, ok := builtins[c.Args[0]]; ok {
		return f(c)
	}
	return c.Run()
}

func execCompound(cmd ast.Compound, s streams) error {
	switch cmd.Op {
	case ast.CompoundPipe:
		return execPipe(cmd, s)
	}
	panic("unreachable")
}

func execPipe(cmd ast.Compound, s streams) error {
	r, w, err := os.Pipe()
	if err != nil {
		return errors.New("Failed to create pipe")
	}

	var el, er error
	wg := sync.WaitGroup{}
	go func() {
		wg.Add(1)
		el = execCommand(cmd.Lhs, streams{s.in, w, s.err})
		wg.Done()
	}()
	er = execCommand(cmd.Rhs, streams{r, s.out, s.err})
	wg.Wait()
	return errors.Join(el, er)
}
