package builtin

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
)

type stack struct {
	dirs []string
}

var dirStack stack

func init() {
	dirStack.dirs = make([]string, 0, 64)
}

func (s *stack) push(dir string) {
	s.dirs = append(s.dirs, dir)
}

func (s *stack) pop() (string, bool) {
	if len(s.dirs) == 0 {
		return "", false
	}
	n := len(s.dirs) - 1
	d := s.dirs[n]
	s.dirs = s.dirs[:n]
	return d, true
}

func cd(cmd *exec.Cmd) uint8 {
	var dst string
	switch len(cmd.Args) {
	case 1:
		user, err := user.Current()
		if err != nil {
			errorf(cmd, "%s", err)
			return 1
		}
		dst = user.HomeDir
	case 2:
		dst = cmd.Args[1]
		if dst == "-" {
			return cdPop(cmd)
		}
	default:
		fmt.Fprintln(cmd.Stderr, "Usage: cd [directory]")
		return 1
	}

	if cwd, err := os.Getwd(); err != nil {
		errorf(cmd, "%s", err)
	} else {
		dirStack.push(cwd)
	}

	if err := os.Chdir(dst); err != nil {
		dirStack.pop()
		errorf(cmd, "%s", err)
		return 1
	}
	return 0
}

func cdPop(cmd *exec.Cmd) uint8 {
	dst, ok := dirStack.pop()
	if !ok {
		errorf(cmd, "the directory stack is empty")
		return 1
	}

	if err := os.Chdir(dst); err != nil {
		errorf(cmd, "%s", err)
		return 1
	}
	return 0
}
