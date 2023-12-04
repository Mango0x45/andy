package stack

type Stack[T comparable] []T

func New[T comparable](n int) Stack[T] {
	return make(Stack[T], 0, n)
}

func (s *Stack[T]) Push(x T) {
	*s = append(*s, x)
}

func (s Stack[T]) Peek() *T {
	if len(s) == 0 {
		return nil
	}
	return &s[len(s)-1]
}

func (s *Stack[T]) Pop() *T {
	if len(*s) == 0 {
		return nil
	}
	n := len(*s) - 1
	x := (*s)[n]
	*s = (*s)[:n]
	return &x
}

func (s *Stack[T]) TopIs(x T) bool {
	y := s.Peek()
	return y != nil && x == *y
}
