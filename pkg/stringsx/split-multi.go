package stringsx

import "strings"

func SplitMulti(s string, seps []string) []string {
	out := make([]string, 0, 8)

	var i int

	for j := 0; j < len(s); j++ {
		for _, sep := range seps {
			if !strings.HasPrefix(s[j:], sep) {
				continue
			}
			out = append(out, s[i:j])
			j += len(sep)
			i = j
		}
	}
	if i < len(s) {
		out = append(out, s[i:])
	}

	return out
}
