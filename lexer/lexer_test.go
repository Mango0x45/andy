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
		t.Fatalf("Expected ‘eof’ but got ‘%c’", r)
	}
}

func TestPeek(t *testing.T) {
	s := "¢ȠʗǱɓǇϴ¤Ίϑ'щƎcɛǩΟȏɁƅ"
	l := New(s)
	chk := func(x, y rune) {
		if x != y {
			t.Fatalf("Expected ‘%c’ but got ‘%c’", x, y)
		}
	}

	rs := []rune(s)
	chk(l.peek(), rs[0])
	chk(l.peek(), rs[0])

	l.next()
	l.next()

	chk(l.peek(), rs[2])
	chk(l.peek(), rs[2])

	l.next()
	l.next()
	l.next()
	l.next()

	chk(l.peek(), rs[6])
	chk(l.peek(), rs[6])
}
