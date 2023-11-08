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
			Op: op,
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
		rhs := p.parseXCommandList()
		cmdList.Rhs = &rhs
	}
}

func (p* Parser) parsePipeline() ast.Pipeline {
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

func (p* Parser) parseSimple() ast.Simple {
	args := make([]ast.Value, 0, 4) // Add a little capacity
	var redirs []ast.Redirect

	switch t := p.next(); t.Kind {
	case lexer.TokArg, lexer.TokString:
		args = append(args, ast.NewValue(t))
	default:
		die(errExpected{"command", t})
	}

outer:
	for {
		switch t := p.peek(); t.Kind {
		case lexer.TokArg, lexer.TokString:
			args = append(args, ast.NewValue(t))
		default:
			break outer
		}

		p.next()
	}

	for {
		switch t := p.peek(); {
		case ast.IsRedir(t.Kind):
			p.next()
			r := ast.NewRedir(t.Kind)

			switch t := p.next(); t.Kind {
			case lexer.TokArg, lexer.TokString:
				r.File = ast.NewValue(t)
			default:
				die(errExpected{"file after redirect", t})
			}

			redirs = append(redirs, r)
		default:
			return ast.Simple{
				Args: args,
				Redirs: redirs,
				In: os.Stdin,
				Out: os.Stdout,
				Err: os.Stderr,
			}
		}
	}
}

func die(e error) {
	fmt.Fprintf(os.Stderr, "andy: %s\n", e)
	os.Exit(1)
}
