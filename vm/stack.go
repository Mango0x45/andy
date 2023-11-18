package vm

type stack[T any] struct {
	xs []T
}

func newStack[T any](n int) *stack[T] {
	return &stack[T]{make([]T, 0, n)}
}

func (s *stack[T]) push(x T) {
	s.xs = append(s.xs, x)
}

func (s *stack[T]) pop() (T, bool) {
	var x T
	if len(s.xs) == 0 {
		return x, false
	}
	n := len(s.xs) - 1
	x = s.xs[n]
	s.xs = s.xs[:n]
	return x, true
}
