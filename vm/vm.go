package vm

import (
	"fmt"
	"os"

	"git.sr.ht/~mango/andy/ast"
)

type Vm struct {
	Status      uint8
	interactive bool
}

func New(interactive bool) *Vm {
	return &Vm{interactive: interactive}
}

func (vm *Vm) Run(prog ast.Program) {
	for _, cl := range prog {
		ret := vm.execCmdList(cl)
		vm.Status = ret.ExitCode()
		if _, ok := ret.(shellError); ok {
			fmt.Fprintf(os.Stderr, "andy: %s\n", ret)
			if !vm.interactive {
				os.Exit(1)
			}
		}
	}
}
