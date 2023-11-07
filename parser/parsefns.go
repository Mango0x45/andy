package parser

import (
	"fmt"
	"os"

	"git.sr.ht/~mango/andy/ast"
	"git.sr.ht/~mango/andy/lexer"
)

func (p *Parser) parseCommands() []ast.Command {
	cmds := []ast.Command{}

	for {
		if k := p.peek().Kind; k == lexer.TokEof {
			return cmds
		}

		cmds = append(cmds, p.parseCommand())
	}
}

func (p *Parser) parseCommand() ast.Command {
	var cmd ast.Command

	switch t := p.peek(); {
	case ast.IsValue(t.Kind):
		cmd = p.parseSimple()
	default:
		die(errExpected{"command", t})
	}

	switch t := p.next(); t.Kind {
	case lexer.TokPipe:
		rhs := p.parseCommand()
		cmd = ast.Compound{
			Lhs: cmd,
			Rhs: rhs,
			Op:  ast.CompoundPipe,
		}
	case lexer.TokEndStmt, lexer.TokEof:
	default:
		die(errExpected{"operator or newline", t})
	}

	return cmd
}

func (p *Parser) parseSimple() ast.Simple {
	args := make([]ast.Value, 0, 4) // Add a little capacity
	var redirs []ast.Redir

outer:
	for {
		switch t := p.peek(); t.Kind {
		case lexer.TokArg:
			args = append(args, ast.Argument(t.Val))
		case lexer.TokString:
			args = append(args, ast.String(t.Val))
		default:
			break outer
		}

		p.next()
	}

	for {
		switch t := p.peek(); {
		case ast.IsRedir(t.Kind):
			p.next() // Consume token
			r := ast.NewRedir(t.Kind)

			switch t := p.next(); t.Kind {
			case lexer.TokArg:
				r.File = ast.Argument(t.Val)
			case lexer.TokString:
				r.File = ast.String(t.Val)
			default:
				die(errExpected{"file after redirect", t})
			}

			redirs = append(redirs, r)
		default:
			return ast.Simple{Args: args, Redirs: redirs}
		}
	}
}

func die(e error) {
	fmt.Fprintf(os.Stderr, "andy: %s\n", e)
	os.Exit(1)
}
