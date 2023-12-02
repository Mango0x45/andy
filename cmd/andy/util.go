package main

func copyMap[K comparable, V any](m map[K]V) map[K]V {
	n := make(map[K]V, len(m))
	for k, v := range m {
		n[k] = v
	}
	return n
}

func longestRunBytes(s string, b byte) int {
	var n, m int

	for _, b_ := range []byte(s) {
		if b == b_ {
			n++
		} else {
			m = max(n, m)
			n = 0
		}
	}

	return max(n, m)
}
