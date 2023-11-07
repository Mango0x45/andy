package ast

type Value interface {
	isValue()
}

// String is a quoted string
type String string

// Argument is an unquoted string
type Argument string

type Concat struct {
	Lhs, Rhs Value
}

func (_ String) isValue()   {}
func (_ Argument) isValue() {}
func (_ Concat) isValue()   {}
