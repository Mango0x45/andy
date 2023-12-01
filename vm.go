package main

import (
	"io"
	"os"
)

type context struct {
	in       io.Reader
	out, err io.Writer
}

type vm struct {
	status      uint8
	interactive bool
}

func (vm *vm) run(prog astProgram) {
	for _, cl := range prog {
		ret := execCmdList(cl, context{
			os.Stdin,
			os.Stdout,
			os.Stderr,
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
