package stack

func (s *Stack[T]) Push(x T) {
	*s = append(*s, x)
}
