package stack

func (s Stack[T]) TopIs(x T, xs ...T) bool {
	xs = append([]T{x}, xs...)
	if len(s) < len(xs) {
		return false
	}
	for i := range xs {
		if xs[i] != s[len(s)-i-1] {
			return false
		}
	}
	return true
}
