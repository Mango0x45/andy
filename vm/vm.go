package vm

import (
	"fmt"
	"io"
	"os"
)

type Vm struct {
	Status      uint8
	interactive bool
}

type context struct {
	in       io.Reader
	out, err io.Writer
}

func New(interactive bool) *Vm {
	return &Vm{interactive: interactive}
}

func (vm *Vm) Run(prog Program) {
	for _, cl := range prog {
		ret := vm.execCmdList(cl, context{
			os.Stdin,
			os.Stdout,
			os.Stderr,
		})
		vm.Status = ret.ExitCode()
		if _, ok := ret.(shellError); ok {
			fmt.Fprintf(os.Stderr, "andy: %s\n", ret)
			if !vm.interactive {
				os.Exit(1)
			}
		}
	}
}
