package stack

func (s *Stack[T]) Pop() *T {
	if len(*s) == 0 {
		return nil
	}
	n := len(*s) - 1
	x := (*s)[n]
	*s = (*s)[:n]
	return &x
}
