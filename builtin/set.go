package builtin

import (
	"fmt"
	"os/exec"

	"git.sr.ht/~mango/andy/lexer"
)

var VarTable map[string][]string = make(map[string][]string, 64)

func set(cmd *exec.Cmd) uint8 {
	argc := len(cmd.Args)
	if argc == 1 {
		fmt.Fprintf(cmd.Stderr, "Usage: set variable [value ...]\n")
		return 1
	}

	ident := cmd.Args[1]
	for _, r := range ident {
		if !lexer.IsRefChar(r) {
			errorf(cmd, "rune ‘%c’ is not allowed in variable names", r)
			return 1
		}
	}

	if argc == 2 {
		if _, ok := VarTable[ident]; !ok {
			errorf(cmd, "variable ‘%s’ was already unset", ident)
			return 1
		}
		delete(VarTable, ident)
	} else {
		VarTable[ident] = cmd.Args[2:]
	}

	return 0
}
