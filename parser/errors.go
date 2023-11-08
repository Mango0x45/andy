package parser

import (
	"fmt"

	"git.sr.ht/~mango/andy/lexer"
)

type errExpected struct {
	want string
	got  lexer.Token
}

func (e errExpected) Error() string {
	return fmt.Sprintf("Expected %s but got %s", e.want, e.got)
}
