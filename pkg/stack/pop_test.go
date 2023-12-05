package stack

import "testing"

func assertPop[T comparable](t *testing.T, s *Stack[T], x *T) {
	y := s.Pop()
	if x == nil && y == nil || x != nil && y != nil && *x == *y {
		return
	}
	t.Fatalf("Expected top of stack to be ‘%+v’ but got ‘%+v’", *x, *y)
}

func TestPop(t *testing.T) {
	x := 1
	y := 69
	z := 420
	s := New[int](0)
	s.Push(x)
	s.Push(y)
	s.Push(z)
	assertPop(t, &s, &z)
	assertPop(t, &s, &y)
	assertPop(t, &s, &x)
	assertPop(t, &s, nil)
}
