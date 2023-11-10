package lexer

import "testing"

func TestNext(t *testing.T) {
	s := "¢ȠʗǱɓǇϴ¤Ίϑ'щƎcɛǩΟȏɁƅ"
	l := New(s)

	for _, x := range []rune(s) {
		if y := l.next(); x != y {
			t.Fatalf("Expected ‘%c’ but got ‘%c’", x, y)
		}
	}

	if r := l.next(); r != eof {
		t.Fatalf("Expected eof but got ‘%c’", r)
	}
}
