package parser

import (
	"git.sr.ht/~mango/andy/ast"
	"git.sr.ht/~mango/andy/lexer"
	"git.sr.ht/~mango/andy/log"
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
		log.Err("Expected command but found ‘%s’", t)
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
		log.Err("Expected operator or newline but found ‘%s’", t)
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
				log.Err("Expected file after redirect but got ‘%s’", t)
			}

			redirs = append(redirs, r)
		default:
			return ast.Simple{Args: args, Redirs: redirs}
		}
	}
}
