package stack

import "testing"

func TestTopIs(t *testing.T) {
	x := 1
	y := 69
	z := 420
	s := New[int](0)
	s.Push(x)
	s.Push(y)
	s.Push(z)

	if !s.TopIs(z) {
		t.Fatalf("Expected top to be [%d]", z)
	}
	if !s.TopIs(z, y) {
		t.Fatalf("Expected top to be [%d, %d]", z, y)
	}
	if !s.TopIs(z, y, x) {
		t.Fatalf("Expected top to be [%d, %d, %d]", z, y, x)
	}
	if s.TopIs(z, y, x, 1337) {
		t.Fatalf("Expected stack to have len(s) == 3")
	}
	s.Pop()
	if !s.TopIs(y) {
		t.Fatalf("Expected top to be [%d]", y)
	}
	if !s.TopIs(y, x) {
		t.Fatalf("Expected top to be [%d, %d]", y, x)
	}
	if s.TopIs(y, x, 1337) {
		t.Fatalf("Expected stack to have len(s) == 2")
	}
	s.Pop()
	if !s.TopIs(x) {
		t.Fatalf("Expected top to be [%d]", x)
	}
	if s.TopIs(y, x, 1337) {
		t.Fatalf("Expected stack to have len(s) == 1")
	}
}
