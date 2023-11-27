package builtin

import (
	"fmt"
	"os/exec"
)

type builtin func(cmd *exec.Cmd) uint8

var Commands = map[string]builtin{
	"cd":    cd,
	"echo":  echo,
	"false": false_,
	"read":  read,
	"set":   set,
	"true":  true_,
}

func errorf(cmd *exec.Cmd, format string, args ...any) {
	format = fmt.Sprintf("%s: %s\n", cmd.Args[0], format)
	fmt.Fprintf(cmd.Stderr, format, args...)
}
