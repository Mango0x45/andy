package main

import (
	"io"
	"os"
)

type context struct {
	in       io.Reader
	out, err io.Writer
	scope    map[string][]string
}

type vm struct {
	status      uint8
	interactive bool
}

type function struct {
	args []string
	body astProgram
}

var (
	funcMap map[string]function = make(map[string]function, 64)
	varMap  map[string][]string = make(map[string][]string, 64)
)

func (vm *vm) run(prog astProgram) {
	for _, tl := range prog {
		ret := execTopLevel(tl, context{
			os.Stdin,
			os.Stdout,
			os.Stderr,
			nil,
		})
		vm.status = ret.ExitCode()
		if _, ok := ret.(shellError); ok {
			warn(ret)
			if !vm.interactive {
				os.Exit(1)
			}
		}
	}
}
