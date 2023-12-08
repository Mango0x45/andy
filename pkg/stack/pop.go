package stack

func (s *Stack[T]) Pop() (T, bool) {
	if len(*s) == 0 {
		var zero T
		return zero, false
	}
	n := len(*s) - 1
	x := (*s)[n]
	*s = (*s)[:n]
	return x, true
}
