package vm

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"

	"git.sr.ht/~mango/andy/lexer"
	"git.sr.ht/~mango/andy/vm/vars"
)

type builtin func(cmd *exec.Cmd) commandResult

var builtins = map[string]builtin{
	"cd":    builtinCd,
	"echo":  builtinEcho,
	"false": builtinFalse,
	"set":   builtinSet,
	"true":  builtinTrue,
}

var cdStack *stack[string] = newStack[string](64)

func builtinCd(cmd *exec.Cmd) commandResult {
	var dst string
	switch len(cmd.Args) {
	case 1:
		cwd, err := os.Getwd()
		if err != nil {
			errorf(cmd, "%s", err)
			return errExitCode(1)
		}
		cdStack.push(cwd)

		user, err := user.Current()
		if err != nil {
			return errInternal{err}
		}
		dst = user.HomeDir
	case 2:
		dst = cmd.Args[1]
		if dst == "-" {
			maybe := cdStack.pop()
			if maybe == nil {
				errorf(cmd, "the directory stack is empty")
				return errExitCode(1)
			}
			dst = *maybe
		} else {
			cwd, err := os.Getwd()
			if err != nil {
				errorf(cmd, "%s", err)
				return errExitCode(1)
			}
			cdStack.push(cwd)
		}
	default:
		fmt.Fprintln(cmd.Stderr, "Usage: cd [directory]")
		return errExitCode(1)
	}

	if err := os.Chdir(dst); err != nil {
		return errInternal{err}
	}
	return errExitCode(0)
}

func builtinEcho(cmd *exec.Cmd) commandResult {
	// Cast to []any
	args := make([]any, len(cmd.Args)-1)
	for i := range args {
		args[i] = cmd.Args[i+1]
	}

	fmt.Fprintln(cmd.Stdout, args...)
	return errExitCode(0)
}

func builtinFalse(cmd *exec.Cmd) commandResult {
	n := len(cmd.Args) - 1
	if n > 0 {
		errorf(cmd, "%d arguments are being ignored", n)
	}
	return errExitCode(1)
}

func builtinSet(cmd *exec.Cmd) commandResult {
	argc := len(cmd.Args)
	if argc == 1 {
		fmt.Fprintf(cmd.Stderr, "Usage: set variable [value ...]\n")
		return errExitCode(1)
	}

	ident := cmd.Args[1]
	for _, r := range ident {
		if !lexer.IsRefChar(r) {
			errorf(cmd, "rune ‘%c’ is not allowed in variable names", r)
			return errExitCode(1)
		}
	}

	if argc == 2 {
		if _, ok := vars.VarTable[ident]; !ok {
			errorf(cmd, "variable ‘%s’ was already unset", ident)
			return errExitCode(1)
		}
		delete(vars.VarTable, ident)
	} else {
		vars.VarTable[ident] = cmd.Args[2:]
	}

	return errExitCode(0)
}

func builtinTrue(cmd *exec.Cmd) commandResult {
	n := len(cmd.Args) - 1
	if n > 0 {
		errorf(cmd, "%d arguments are being ignored", n)
	}
	return errExitCode(0)
}

func errorf(cmd *exec.Cmd, format string, args ...any) {
	format = fmt.Sprintf("%s: %s\n", cmd.Args[0], format)
	fmt.Fprintf(cmd.Stderr, format, args...)
}
