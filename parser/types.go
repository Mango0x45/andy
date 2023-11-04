package parser

type Exprs []Expr

type Expr interface {
	isExpression()
}

type Pipe struct {
	L, R Expr
}

type Cmd struct {
	Argv   []Strings
	Stdin  Redirection
	Stdout Redirection
}

func (_ Pipe) isExpression() {}
func (_ Cmd) isExpression()  {}

type Strings interface {
	ToStrings() []string
}

type Concat struct {
	L, R Strings
}

type String string

func (s String) ToStrings() []string {
	return []string{string(s)}
}

func (c Concat) ToStrings() []string {
	s1 := c.L.ToStrings()
	s2 := c.R.ToStrings()

	dst := make([]string, 0, len(s1)*len(s2))
	for i := range s1 {
		for j := range s2 {
			dst = append(dst, s1[i]+s2[j])
		}
	}

	return dst
}

type RedirMode int

const (
	RedirNone RedirMode = iota

	RedirAppend
	RedirAppendClobber
	RedirClobber
	RedirNoClobber
)

type Redirection struct {
	Kind RedirMode
	File []Strings
}
