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
		processStdin()
	case 2:
		processFile(os.Args[1])
	default:
		fmt.Fprintf(os.Stderr, "Usage: %s [file]\n", os.Args[0])
		os.Exit(1)
	}
}

func processStdin() {
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

		l := lexer.New(line)
		go l.Run()
		vm.Exec(parser.Parse(l.Out))
	}
}

func processFile(filename string) {
	f, err := os.Open(os.Args[1])
	if err != nil {
		eprintln(err)
	}
	defer f.Close()

	bytes, err := io.ReadAll(f)
	if err != nil {
		eprintln(err)
	}

	l := lexer.New(string(bytes))
	go l.Run()
	vm.Exec(parser.Parse(l.Out))
}

func eprintln(err error) {
	fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], err)
	os.Exit(1)
}
