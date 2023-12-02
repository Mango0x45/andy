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

	globalVariableMap["status"] = []string{"0"}
	globalVariableMap["pid"] = []string{strconv.Itoa(os.Getpid())}
}

func (vm *vm) run(prog astProgram) {
	for _, tl := range prog {
		ret := execTopLevel(tl, context{
			os.Stdin,
			os.Stdout,
			os.Stderr,
			nil,
		})
		code := int(ret.ExitCode())
		globalVariableMap["status"] = []string{strconv.Itoa(code)}
		if _, ok := ret.(shellError); ok {
			warn(ret)
			if !vm.interactive {
				os.Exit(1)
			}
		}
	}
}
