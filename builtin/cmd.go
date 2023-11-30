package builtin

import (
	"fmt"
	"os/exec"
)

func cmd(cmd *exec.Cmd) uint8 {
	if len(cmd.Args) < 2 {
		fmt.Fprintln(cmd.Stderr, "Usage: cmd command [args ...]")
		return 1
	}

	c := exec.Command(cmd.Args[1], cmd.Args[2:]...)
	c.Stdin, c.Stdout, c.Stderr = cmd.Stdin, cmd.Stdout, cmd.Stderr
	c.ExtraFiles = cmd.ExtraFiles
	err := c.Run()
	code := c.ProcessState.ExitCode()

	if err != nil && code == -1 {
		errorf(cmd, "%s", err)
		return 1
	}
	return uint8(code)
}
