package vm

import (
	"errors"
	"fmt"
	"os"

	"git.sr.ht/~mango/andy/parser"
)

type builtin func(parser.Cmd, []string) error

var builtins = map[string]builtin{
	"echo": builtinEcho,
}

func builtinEcho(cmd parser.Cmd, argv []string) error {
	xs := make([]any, len(argv)-1)
	for i, a := range argv[1:] {
		xs[i] = a
	}

	var fp *os.File
	switch cmd.Stdout.Kind {
	case parser.RedirNone:
		fp = os.Stdout
	
	case parser.RedirClobber:
		strs := flattenStringsSlice(cmd.Stdout.File)
		if len(strs) > 1 {
			return errMultipleStrings(strs)
		}

		var err error
		fp, err = os.Create(strs[0])
		if err != nil {
			return err
		}
		defer fp.Close()

	case parser.RedirNoClobber:
		strs := flattenStringsSlice(cmd.Stdout.File)
		if len(strs) > 1 {
			return errMultipleStrings(strs)
		}

		f := strs[0]
		_, err := os.Stat(f)
		switch {
		case errors.Is(err, os.ErrNotExist):
			fp, err = os.Create(f)
			if err != nil {
				return err
			}
			defer fp.Close()
		case err != nil:
			return nil
		default: // File exists
			return errWontClobber{f, false}
		}
	}

	_, err := fmt.Fprintln(fp, xs...)
	return err
}
