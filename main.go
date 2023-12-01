package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
)

func main() {
	switch len(os.Args) {
	case 1:
		runRepl()
	case 2:
		runFile(os.Args[1])
	default:
		fmt.Fprintln(os.Stderr, "Usage: andy [file]")
		os.Exit(1)
	}
}

func runRepl() {
	vm := vm{interactive: true}
	r := bufio.NewReader(os.Stdin)

	for {
		fmt.Fprintf(os.Stderr, "[%d] > ", vm.status)
		line, err := r.ReadString('\n')

		switch {
		case errors.Is(err, io.EOF):
			fmt.Fprintln(os.Stderr, "^D")
			os.Exit(0)
		case err != nil:
			warn(err)
		}

		l := newLexer(line)
		p := newParser(l.out)
		go l.run()
		vm.run(p.run())
	}
}

func runFile(f string) {
	vm := vm{}
	bytes, err := os.ReadFile(f)
	if err != nil {
		die(err)
	}

	l := newLexer(string(bytes))
	p := newParser(l.out)
	go l.run()
	vm.run(p.run())
}

func warn(e error) {
	fmt.Fprintf(os.Stderr, "andy: %s\n", e)
}

func die(e error) {
	fmt.Fprintf(os.Stderr, "andy: %s\n", e)
	os.Exit(1)
}
