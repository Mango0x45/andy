package builtin

import (
	"errors"
	"fmt"
	"os/exec"
	"syscall"
)

func echo(cmd *exec.Cmd) uint8 {
	// Cast to []any
	args := make([]any, len(cmd.Args)-1)
	for i := range args {
		args[i] = cmd.Args[i+1]
	}

	_, err := fmt.Fprintln(cmd.Stdout, args...)
	if err != nil && !errors.Is(err, syscall.EPIPE) {
		errorf(cmd, "%s", err)
		return 1
	}
	return 0
}
