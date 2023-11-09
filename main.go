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
	switch len(os.Args) {
	case 1:
		repl()
	case 2:
		file(os.Args[1])
	default:
		fmt.Fprintln(os.Stderr, "Usage: andy [file]")
		os.Exit(1)
	}
}

func repl() {
	r := bufio.NewReader(os.Stdin)

	for {
		fmt.Fprint(os.Stderr, "$ ")
		line, err := r.ReadString('\n')

		switch {
		case errors.Is(err, io.EOF):
			fmt.Fprintln(os.Stderr, "^D")
			os.Exit(0)
		case err != nil:
			eprintln(err)
		}

		exec(line)
	}
}

func file(f string) {
	bytes, err := os.ReadFile(f)
	if err != nil {
		eprintln(err)
		os.Exit(1)
	}

	exec(string(bytes))
}

func exec(s string) {
	l := lexer.New(s)
	p := parser.New(l.Out)
	go l.Run()
	vm.Exec(p.Run())
}

func eprintln(e error) {
	fmt.Fprintf(os.Stderr, "andy: %s\n", e)
}
