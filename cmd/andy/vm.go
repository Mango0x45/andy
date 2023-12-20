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
	file        bool
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
	globalVariableMap = map[string][]string{
		"_":      {}, // Other shells export this
		"pid":    {strconv.Itoa(os.Getpid())},
		"ppid":   {strconv.Itoa(os.Getppid())},
		"status": {"0"},
	}
}

func (vm *vm) run(prog astProgram) {
	if vm.file {
		globalVariableMap["args"] = os.Args[1:]
	}

	var failed bool
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
				failed = true
				break
			}
		}
	}
	if f, ok := globalFuncMap["sigexit"]; ok {
		res := execTopLevels(f.body, context{
			os.Stdin,
			os.Stdout,
			os.Stderr,
			map[string][]string{"_": {}},
		})
		if cmdFailed(res) {
			failed = true
		}
	}

	if failed {
		os.Exit(1)
	}
}
