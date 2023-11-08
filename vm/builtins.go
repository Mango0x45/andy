package vm

import (
	"fmt"
	"os/exec"
)

type builtin func(cmd *exec.Cmd) commandResult

var builtins = map[string]builtin{
	"echo": builtinEcho,
	"false": builtinFalse,
	"true": builtinTrue,
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

func builtinFalse(_ *exec.Cmd) commandResult {
	return errExitCode(1)
}

func builtinTrue(_ *exec.Cmd) commandResult {
	return errExitCode(0)
}
