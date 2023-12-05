package main

import (
	"io"
	"os"
	"strconv"
)

type context struct {
	in       io.Reader
	out, err io.Writer
	scope    map[string][]string
}

type vm struct {
	interactive bool
}

type function struct {
	args []string
	body astProgram
}

var (
	globalFuncMap     map[string]function
	globalVariableMap map[string][]string
)

func init() {
	globalFuncMap = make(map[string]function, 64)
	globalVariableMap = make(map[string][]string, 64)

	globalVariableMap["_"] = []string{} // Other shells export this
	globalVariableMap["status"] = []string{"0"}
	globalVariableMap["pid"] = []string{strconv.Itoa(os.Getpid())}
	globalVariableMap["ppid"] = []string{strconv.Itoa(os.Getppid())}
}

func (vm *vm) run(prog astProgram) {
	for _, tl := range prog {
		res := execTopLevel(tl, context{
			os.Stdin,
			os.Stdout,
			os.Stderr,
			nil,
		})
		code := int(res.ExitCode())
		globalVariableMap["status"] = []string{strconv.Itoa(code)}
		if cmdFailed(res) {
			if _, ok := res.(errExitCode); !ok {
				warn(res)
			}
			if !vm.interactive {
				os.Exit(1)
			}
		}
	}
}
