package vm

import (
	"errors"
	"os"
	"os/exec"
	"sync"

	"git.sr.ht/~mango/andy/ast"
	"git.sr.ht/~mango/andy/log"
)

const appendFlags = os.O_APPEND | os.O_CREATE | os.O_WRONLY

func Exec(prog ast.Program) {
	for _, cmd := range prog {
		execCommand(cmd, streams{os.Stdin, os.Stdout, os.Stderr})
	}
}

func execCommand(cmd ast.Command, s streams) {
	switch cmd.(type) {
	case ast.Simple:
		execSimple(cmd.(ast.Simple), s)
	case ast.Compound:
		execCompound(cmd.(ast.Compound), s)
	default:
		panic("unreachable")
	}
}

func execSimple(cmd ast.Simple, s streams) {
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
					log.Err("Failed to open file ‘%s’: %s", name, err)
					return
			}
			defer fp.Close()
			c.Stdout = fp
		case ast.RedirRead:
			fp, err := os.Open(name)
			if err != nil {
					log.Err("Failed to open file ‘%s’: %s", name, err)
					return
			}
			defer fp.Close()
			c.Stdin = fp
		case ast.RedirWrite:
			_, err := os.Stat(name)
			switch {
			case errors.Is(err, os.ErrNotExist):
				fp, err := os.Create(name)
				if err != nil {
					log.Err("Failed to create file ‘%s’: %s", name, err)
					return
				}
				defer fp.Close()
				c.Stdout = fp
			case err != nil:
				log.Err("Failed to stat file ‘%s’: %s", name, err)
				return
			default: // File exists
				log.Err("Won’t clobber file ‘%s’; did you mean to use ‘>|’?", name)
				return
			}
		case ast.RedirWriteClob:
			fp, err := os.Create(name)
			if err != nil {
				log.Err("Failed to create file ‘%s’: %s", name, err)
				return
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
		f(c)
	} else {
		c.Run()
	}
}

func execCompound(cmd ast.Compound, s streams) {
	switch cmd.Op {
	case ast.CompoundPipe:
		execPipe(cmd, s)
	default:
		panic("unreachable")
	}
}

func execPipe(cmd ast.Compound, s streams) {
	r, w, err := os.Pipe()
	if err != nil {
		log.Err("Failed to create pipe")
		return
	}

	wg := sync.WaitGroup{}
	go func() {
		wg.Add(1)
		execCommand(cmd.Lhs, streams{s.in, w, s.err})
		wg.Done()
	}()
	execCommand(cmd.Rhs, streams{r, s.out, s.err})
	wg.Wait()
}
