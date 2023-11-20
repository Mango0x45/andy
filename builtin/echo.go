package builtin

import (
	"fmt"
	"os/exec"
)

func echo(cmd *exec.Cmd) uint8 {
	// Cast to []any
	args := make([]any, len(cmd.Args)-1)
	for i := range args {
		args[i] = cmd.Args[i+1]
	}

	if _, err := fmt.Fprintln(cmd.Stdout, args...); err != nil {
		errorf(cmd, "%s", err)
		return 1
	}
	return 0
}