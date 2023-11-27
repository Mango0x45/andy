package vm

import (
	"fmt"
	"io"
	"os"
)

var (
	Status      uint8
	Interactive bool
)

type context struct {
	in       io.Reader
	out, err io.Writer
}

func Run(prog Program) {
	for _, cl := range prog {
		ret := execCmdList(cl, context{
			os.Stdin,
			os.Stdout,
			os.Stderr,
		})
		Status = ret.ExitCode()
		if _, ok := ret.(shellError); ok {
			fmt.Fprintf(os.Stderr, "andy: %s\n", ret)
			if !Interactive {
				os.Exit(1)
			}
		}
	}
}
