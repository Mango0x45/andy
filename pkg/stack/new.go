package stack

func New[T comparable](n int) Stack[T] {
	return make(Stack[T], 0, n)
}
