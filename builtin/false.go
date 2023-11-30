package builtin

import "os/exec"

func false_(cmd *exec.Cmd) uint8 {
	return 1
}
