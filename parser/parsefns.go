package parser

import (
	"os"

	"git.sr.ht/~mango/andy/lexer"
)

func Parse(toks <-chan lexer.Token) Exprs {
	p := &parser{toks: toks}
	prog := []Expr{}

	for {
		if p.peek().Kind == lexer.TokEof {
			return prog
		}
		prog = append(prog, p.parseExpr())
	}
}

func (p *parser) parseExpr() Expr {
	var e Expr

	switch t := p.peek(); t.Kind {
	case lexer.TokString:
		e = p.parseCmd()
	default:
		eprintln(errExpected{e: "command", g: t})
	}

	for p.peek().Kind == lexer.TokEndStmt {
		p.next()
	}

	return e
}

func (p *parser) parseCmd() Expr {
	c := Cmd{Argv: p.parseStrings()}

	for {
		t := p.peek()

		switch {
		case t.Kind.IsRead() && c.Stdin.Kind != RedirNone:
			eprintln(errRedirect(c.Stdin.Kind))
		case t.Kind.IsWrite() && c.Stdout.Kind != RedirNone:
			eprintln(errRedirect(c.Stdout.Kind))
		}

		switch t.Kind {
		case lexer.TokRead:
			p.next()
			c.Stdin = Redirection{RedirNoClobber, p.parseStrings()}
		case lexer.TokReadNull:
			p.next()
			dst := []Strings{String(os.DevNull)}
			c.Stdin = Redirection{RedirNoClobber, dst}
		case lexer.TokWrite:
			p.next()
			c.Stdout = Redirection{RedirNoClobber, p.parseStrings()}
		case lexer.TokWriteClob:
			p.next()
			c.Stdout = Redirection{RedirClobber, p.parseStrings()}
		case lexer.TokWriteErr:
			p.next()
			dst := []Strings{String(os.Stderr.Name())}
			c.Stdout = Redirection{RedirClobber, dst}
		case lexer.TokWriteNull:
			p.next()
			dst := []Strings{String(os.DevNull)}
			c.Stdout = Redirection{RedirClobber, dst}
		default:
			return c
		}
	}
}

func (p *parser) parseStrings() []Strings {
	t := p.next()
	if t.Kind != lexer.TokString {
		eprintln(errExpected{e: "string", g: t})
	}

	strs := []Strings{String(t.Val)}

	for {
		switch t := p.peek(); t.Kind {
		case lexer.TokString:
			p.next()
			strs = append(strs, String(t.Val))
		case lexer.TokConcat:
			p.next()
			s := strs[len(strs)-1]

			switch t := p.next(); t.Kind {
			case lexer.TokString:
				strs[len(strs)-1] = Concat{
					L: s,
					R: String(t.Val),
				}
			default:
				eprintln(errExpected{e: "string", g: t})
			}
		default:
			return strs
		}
	}
}
