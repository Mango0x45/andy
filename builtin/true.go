package builtin

import "os/exec"

func true_(cmd *exec.Cmd) uint8 {
	n := len(cmd.Args) - 1
	if n > 0 {
		errorf(cmd, "%d arguments are being ignored", n)
	}
	return 0
}
