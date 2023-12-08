package stack

import "testing"

func assertPop[T comparable](t *testing.T, s *Stack[T], x T, b1 bool) {
	y, b2 := s.Pop()
	if b1 != b2 || b1 && b2 && x != y {
		t.Fatalf("Expected top of stack to be ‘%+v’ but got ‘%+v’", x, y)
	}
}

func TestPop(t *testing.T) {
	x := 1
	y := 69
	z := 420
	s := New[int](0)
	s.Push(x)
	s.Push(y)
	s.Push(z)
	assertPop(t, &s, z, true)
	assertPop(t, &s, y, true)
	assertPop(t, &s, x, true)
	assertPop(t, &s, 0, false)
}
