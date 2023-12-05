package stringsx

import "testing"

func TestSplitMultiSimple(t *testing.T) {
	s := "foo::bar::baz"
	xs := SplitMulti(s, []string{"::"})
	if len(xs) != 3 {
		t.Fatalf("Expected len(xs) == 3 but got %d", len(xs))
	}
	if xs[0] != "foo" {
		t.Fatalf("Expected xs[0] == \"foo\" but got ‘%s’\n", xs[0])
	}
	if xs[1] != "bar" {
		t.Fatalf("Expected xs[1] == \"bar\" but got ‘%s’\n", xs[1])
	}
	if xs[2] != "baz" {
		t.Fatalf("Expected xs[2] == \"baz\" but got ‘%s’\n", xs[2])
	}
}

func TestSplitMulti(t *testing.T) {
	s := "foo::bar--::baz"
	xs := SplitMulti(s, []string{"::", "--"})
	if len(xs) != 4 {
		t.Fatalf("Expected len(xs) == 4 but got %d", len(xs))
	}
	if xs[0] != "foo" {
		t.Fatalf("Expected xs[0] == \"foo\" but got ‘%s’\n", xs[0])
	}
	if xs[1] != "bar" {
		t.Fatalf("Expected xs[1] == \"bar\" but got ‘%s’\n", xs[1])
	}
	if xs[2] != "" {
		t.Fatalf("Expected xs[2] == \"\" but got ‘%s’\n", xs[2])
	}
	if xs[3] != "baz" {
		t.Fatalf("Expected xs[3] == \"baz\" but got ‘%s’\n", xs[3])
	}
}

func TestSplitOverlapping1(t *testing.T) {
	s := "foo::bar"
	xs := SplitMulti(s, []string{"::", ":b"})
	ys := SplitMulti(s, []string{":b", "::"})
	if len(xs) != 2 {
		t.Fatalf("Expected len(xs) == 2 but got %d", len(xs))
	}
	if len(ys) != 2 {
		t.Fatalf("Expected len(ys) == 2 but got %d", len(ys))
	}
	if xs[0] != "foo" {
		t.Fatalf("Expected xs[0] == \"foo\" but got ‘%s’\n", xs[0])
	}
	if ys[0] != "foo" {
		t.Fatalf("Expected ys[0] == \"foo\" but got ‘%s’\n", ys[0])
	}
	if xs[1] != "bar" {
		t.Fatalf("Expected xs[1] == \"bar\" but got ‘%s’\n", xs[1])
	}
	if ys[1] != "bar" {
		t.Fatalf("Expected ys[1] == \"bar\" but got ‘%s’\n", ys[1])
	}
}

func TestSplitOverlapping2(t *testing.T) {
	s := "foo:::bar"
	xs := SplitMulti(s, []string{"::", ":::"})
	ys := SplitMulti(s, []string{":::", "::"})
	if len(xs) != 2 {
		t.Fatalf("Expected len(xs) == 2 but got %d", len(xs))
	}
	if len(ys) != 2 {
		t.Fatalf("Expected len(ys) == 2 but got %d", len(ys))
	}
	if xs[0] != "foo" {
		t.Fatalf("Expected xs[0] == \"foo\" but got ‘%s’\n", xs[0])
	}
	if ys[0] != "foo" {
		t.Fatalf("Expected ys[0] == \"foo\" but got ‘%s’\n", ys[0])
	}
	if xs[1] != ":bar" {
		t.Fatalf("Expected xs[1] == \":bar\" but got ‘%s’\n", xs[1])
	}
	if ys[1] != "bar" {
		t.Fatalf("Expected ys[1] == \"bar\" but got ‘%s’\n", ys[1])
	}
}
