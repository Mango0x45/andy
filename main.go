package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"

	"git.sr.ht/~mango/andy/lexer"
	"git.sr.ht/~mango/andy/parser"
	"git.sr.ht/~mango/andy/vm"
)

func main() {
	r := bufio.NewReader(os.Stdin)

	for {
		fmt.Fprint(os.Stderr, "$ ")
		line, err := r.ReadString('\n')

		switch {
		case errors.Is(err, io.EOF):
			fmt.Fprintln(os.Stderr, "^D")
			os.Exit(0)
		case err != nil:
			fmt.Fprintf(os.Stderr, "andy: %s\n", err)
		}

		l := lexer.New(line)
		p := parser.New(l.Out)

		go l.Run()
		vm.Exec(p.Run())
	}
}
