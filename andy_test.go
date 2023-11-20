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

func runAndCapture(t *testing.T, name string, wantOut, wantErr string) {
	c := exec.Command("../andy", name+".an")
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
	runAndCapture(t, "simple", s, "")
}

func TestRedirects(t *testing.T) {
	runAndCapture(t, "redirects", "oof\n", "")

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
	runAndCapture(t, "pipes", s, "")
}

func TestLogical(t *testing.T) {
	s := "foo1\n" +
		"bar1\n" +
		"baz1\n" +
		"foo2\n" +
		"bar2\n" +
		"baz2\n" +
		"chain failed3\n" +
		"bar4\n" +
		"chain failed4\n"
	runAndCapture(t, "logical", s, "")
}

func TestConcat(t *testing.T) {
	s := "foo bar baz\n" +
		"foobarbaz\n" +
		"a.c b.c c.c\n" +
		"a-c a-b a-a b-c b-b b-a c-c c-b c-a\n" +
		"a b c c b a\n"
	runAndCapture(t, "concat", s, "")
}

func TestStrings(t *testing.T) {
	s := "foo\tbar\tbaz\n" +
		"hello\nworld\n" +
		"foo\\tbar\\tbaz\n" +
		"you shouldn't have done that\n" +
		"\n" +
		"foo\nbar baz\ntext\twith\ttabs\n" +
		"foo bar\n" +
		"foo\tbar\n"
	runAndCapture(t, "strings", s, "")
}

func TestTilde(t *testing.T) {
	dir, _ := os.UserHomeDir()
	s := dir + "/foo/bar\n" +
		dir + "\n" +
		"/root\n" +
		"~ \n" +
		"~\n" +
		" ~\n"
	runAndCapture(t, "tilde", s, "")
}

func TestVariables(t *testing.T) {
	s := "0\n" +
		"foo.c\n" +
		"bar.c\n" +
		"baz.c\n" +
		"foo bar baz.c\n" +
		"3.c\n" +
		"barb\n" +
		"That barb was sharp\n" +
		"1 2 3\n" +
		"foo bar baz foo bar baz\n"
	runAndCapture(t, "variables", s, "")
}

func TestIndex(t *testing.T) {
	out := "1\n2\n3\n1\n2\n3\n"
	err := "bad index\n" +
		"out of range\n"
	runAndCapture(t, "index", out, err)
}
