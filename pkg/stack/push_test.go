package stack

import "testing"

func assertPush[T comparable](t *testing.T, s Stack[T], x T) {
	y := s[len(s)-1]
	if x != y {
		t.Fatalf("Expected top of stack to be ‘%+v’ but got ‘%+v’", x, y)
	}
}

func TestPush(t *testing.T) {
	s := New[int](0)
	s.Push(1)
	assertPush(t, s, 1)
	s.Push(69)
	assertPush(t, s, 69)
	s.Push(420)
	assertPush(t, s, 420)
}
