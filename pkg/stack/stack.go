package stack

type Stack[T any] struct {
	xs []T
}

func New[T any](n int) Stack[T] {
	return Stack[T]{make([]T, 0, n)}
}

func (s *Stack[T]) Push(x T) {
	s.xs = append(s.xs, x)
}

func (s Stack[T]) Peek() *T {
	if len(s.xs) == 0 {
		return nil
	}
	return &s.xs[len(s.xs)-1]
}

func (s *Stack[T]) Pop() *T {
	if len(s.xs) == 0 {
		return nil
	}
	n := len(s.xs) - 1
	x := s.xs[n]
	s.xs = s.xs[:n]
	return &x
}
