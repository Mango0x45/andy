package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
)

var globalVm vm

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
	r := bufio.NewReader(os.Stdin)
	globalVm.interactive = true

	for {
		fmt.Fprintf(os.Stderr, "[%d] > ", globalVm.status)
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
		globalVm.run(p.run())
	}
}

func runFile(f string) {
	bytes, err := os.ReadFile(f)
	if err != nil {
		die(err)
	}

	l := newLexer(string(bytes))
	p := newParser(l.out)
	go l.run()
	globalVm.run(p.run())
}

func warn(e error) {
	fmt.Fprintf(os.Stderr, "andy: %s\n", e)
}

func die(e error) {
	fmt.Fprintf(os.Stderr, "andy: %s\n", e)
	os.Exit(1)
}