package main

import (
	"bytes"
	"os"
	"os/exec"
	"testing"
)

func init() {
	// Ensure that we are testing the current code, and are in the testing
	// directory
	exec.Command("go", "build").Run()
	os.Chdir("./testdata")
}

func runAndCapture(t *testing.T, argv []string, wantOut, wantErr string) {
	c := exec.Command("../andy", argv...)
	var out, err bytes.Buffer
	c.Stdout = &out
	c.Stderr = &err

	if err := c.Run(); err != nil {
		t.Fatalf("Command failed: %s", err)
	}
	if out.String() != wantOut {
		t.Fatalf("Stdout returned unexpected ‘%s’", out.String())
	}
	if err.String() != wantErr {
		t.Fatalf("Stderr returned unexpected ‘%s’", err.String())
	}
}

func TestSimple(t *testing.T) {
	s := "hello world\n" +
		"this is a simple builtin-command\n" +
		"this is a simple external process\n"
	runAndCapture(t, []string{"simple.an"}, s, "")
}

func TestRedirects(t *testing.T) {
	runAndCapture(t, []string{"redirects.an"}, "oof\n", "")

	foo, _ := os.ReadFile("foo")
	bar, _ := os.ReadFile("bar")

	if string(foo) != "foo\nbaz\n" {
		t.Fatalf("File ‘foo’ contained unexpected ‘%s’", foo)
	}
	if string(bar) != "bar\n" {
		t.Fatalf("File ‘bar’ contained unexpected ‘%s’", bar)
	}

	os.Remove("foo")
	os.Remove("bar")
}

func TestPipes(t *testing.T) {
	s := "rab oof\n" +
		"wOrld\n"
	runAndCapture(t, []string{"pipes.an"}, s, "")
}
