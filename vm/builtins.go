package vm

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
)

type builtin func(cmd *exec.Cmd) commandResult

var builtins = map[string]builtin{
	"cd":    builtinCd,
	"echo":  builtinEcho,
	"false": builtinFalse,
	"true":  builtinTrue,
}

func builtinCd(cmd *exec.Cmd) commandResult {
	var dst string
	switch len(cmd.Args) {
	case 1:
		user, err := user.Current()
		if err != nil {
			return errInternal{err}
		}
		dst = user.HomeDir
	case 2:
		dst = cmd.Args[1]
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
		fmt.Fprintf(cmd.Stderr, "andy: %d arguments to ‘false’ are being ignored\n", n)
	}
	return errExitCode(1)
}

func builtinTrue(cmd *exec.Cmd) commandResult {
	n := len(cmd.Args) - 1
	if n > 0 {
		fmt.Fprintf(cmd.Stderr, "andy: %d arguments to ‘true’ are being ignored\n", n)
	}
	return errExitCode(0)
}
