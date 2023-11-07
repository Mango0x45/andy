package vm

import (
	"fmt"
	"os/exec"
)

type builtin func(cmd *exec.Cmd) error

var builtins = map[string]builtin{
	"echo": builtinEcho,
}

func builtinEcho(cmd *exec.Cmd) error {
	// Cast to []any
	args := make([]any, len(cmd.Args)-1)
	for i := range args {
		args[i] = cmd.Args[i+1]
	}

	fmt.Fprintln(cmd.Stdout, args...)
	return nil
}
