package vm

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"git.sr.ht/~mango/andy/parser"
)

func Exec(prog parser.Exprs) error {
	for _, e := range prog {
		var err error

		switch e.(type) {
		case parser.Cmd:
			err = execCmd(e.(parser.Cmd))
		}

		if err != nil {
			eprintln(err)
		}
	}

	return nil
}

func execCmd(cmd parser.Cmd) error {
	argv := flattenStringsSlice(cmd.Argv)
	if b, ok := builtins[argv[0]]; ok {
		return b(cmd, argv)
	}

	c := exec.Command(argv[0], argv[1:]...)
	c.Stderr = os.Stderr

	switch cmd.Stdin.Kind {
	case parser.RedirNone:
		c.Stdin = os.Stdin

	case parser.RedirNoClobber:
		strs := flattenStringsSlice(cmd.Stdin.File)
		if len(strs) > 1 {
			return errMultipleStrings(strs)
		}

		fp, err := os.Open(strs[0])
		if err != nil {
			return err
		}
		defer fp.Close()
		c.Stdin = fp
	}

	switch cmd.Stdout.Kind {
	case parser.RedirNone:
		c.Stdout = os.Stdout
	
	case parser.RedirClobber:
		strs := flattenStringsSlice(cmd.Stdout.File)
		if len(strs) > 1 {
			return errMultipleStrings(strs)
		}

		fp, err := os.Create(strs[0])
		if err != nil {
			return err
		}
		defer fp.Close()
		c.Stdout = fp

	case parser.RedirNoClobber:
		strs := flattenStringsSlice(cmd.Stdout.File)
		if len(strs) > 1 {
			return errMultipleStrings(strs)
		}

		f := strs[0]
		_, err := os.Stat(f)
		switch {
		case errors.Is(err, os.ErrNotExist):
			fp, err := os.Create(f)
			if err != nil {
				return err
			}
			defer fp.Close()
			c.Stdout = fp
		case err != nil:
			return nil
		default: // File exists
			return errWontClobber{f, false}
		}
	}

	return c.Run()
}

func flattenStringsSlice(src []parser.Strings) []string {
	dst := make([]string, 0, len(src))

	for _, s := range src {
		dst = append(dst, s.ToStrings()...)
	}

	return dst
}

func eprintln(err error) {
	fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], err)
}
