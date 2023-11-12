package parser

import (
	"fmt"
	"os"

	"git.sr.ht/~mango/andy/ast"
	"git.sr.ht/~mango/andy/lexer"
)

func (p *Parser) parseProgram() ast.Program {
	prog := ast.Program{}

	for {
		switch p.peek().Kind {
		case lexer.TokEndStmt:
			p.next()
		case lexer.TokEof:
			return prog
		default:
			prog = append(prog, p.parseCommandList())
		}
	}
}

func (p *Parser) parseCommandList() ast.CommandList {
	xlist := p.parseXCommandList()
	cmdList := ast.CommandList{Lhs: nil, Rhs: xlist.Lhs}
	op := xlist.Op

	for xlist.Rhs != nil {
		xlist = *xlist.Rhs
		tmp := cmdList
		cmdList = ast.CommandList{
			Lhs: &tmp,
			Op:  op,
			Rhs: xlist.Lhs,
		}
		op = xlist.Op
	}

	return cmdList
}

func (p *Parser) parseXCommandList() ast.XCommandList {
	cmdList := ast.XCommandList{Lhs: p.parsePipeline()}
	for {
		switch p.peek().Kind {
		case lexer.TokLAnd:
			cmdList.Op = ast.LAnd
		case lexer.TokLOr:
			cmdList.Op = ast.LOr
		default:
			return cmdList
		}

		p.next() // Consume operator
		for p.peek().Kind == lexer.TokEndStmt {
			p.next()
		}

		rhs := p.parseXCommandList()
		cmdList.Rhs = &rhs
	}
}

func (p *Parser) parsePipeline() ast.Pipeline {
	pipe := ast.Pipeline{p.parseSimple()}

	for {
		switch p.peek().Kind {
		case lexer.TokPipe:
			p.next()
			pipe = append(pipe, p.parseSimple())
		case lexer.TokEndStmt:
			p.next()
		default:
			return pipe
		}
	}
}

func (p *Parser) parseSimple() ast.Simple {
	args := make([]ast.Value, 0, 4) // Add a little capacity
	var redirs []ast.Redirect

	args = append(args, p.parseValue())
	for ast.IsValue(p.peek().Kind) {
		args = append(args, p.parseValue())
	}

	for {
		switch t := p.peek(); {
		case ast.IsRedir(t.Kind):
			p.next()
			r := ast.NewRedir(t.Kind)

			switch {
			case ast.IsValue(p.peek().Kind):
				r.File = p.parseValue()
			default:
				die(errExpected{"file after redirect", t})
			}

			redirs = append(redirs, r)
		case ast.IsValue(t.Kind):
			die(errExpected{"semicolon or newline", t})
		default:
			return ast.Simple{
				Args:   args,
				Redirs: redirs,
				In:     os.Stdin,
				Out:    os.Stdout,
				Err:    os.Stderr,
			}
		}
	}
}

func (p *Parser) parseValue() ast.Value {
	var v ast.Value

	switch t := p.next(); t.Kind {
	case lexer.TokArg, lexer.TokString:
		v = ast.NewValue(t)
	case lexer.TokPOpen:
		v = p.parseList()
	default:
		die(errExpected{"value", t})
	}

	if p.peek().Kind == lexer.TokConcat {
		p.next()
		v = ast.Concat{Lhs: v, Rhs: p.parseValue()}
	}

	return v
}

func (p *Parser) parseList() ast.List {
	xs := ast.List{}

	for {
		switch t := p.next(); t.Kind {
		case lexer.TokPClose:
			return xs
		case lexer.TokArg, lexer.TokString:
			xs = append(xs, ast.NewValue(t))
		case lexer.TokPOpen:
			xs = append(xs, p.parseList()...)
		default:
			die(errExpected{"list item", t})
		}
	}
}

func die(e error) {
	fmt.Fprintf(os.Stderr, "andy: %s\n", e)
	os.Exit(1)
}
