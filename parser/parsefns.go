package parser

import (
	"fmt"
	"os"

	"git.sr.ht/~mango/andy/lexer"
	"git.sr.ht/~mango/andy/vm"
)

func (p *Parser) parseProgram() vm.Program {
	prog := vm.Program{}

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

func (p *Parser) parseCommandList() vm.CommandList {
	xlist := p.parseXCommandList()
	cmdList := vm.CommandList{Lhs: nil, Rhs: xlist.Lhs}
	op := xlist.Op

	for xlist.Rhs != nil {
		xlist = *xlist.Rhs
		tmp := cmdList
		cmdList = vm.CommandList{
			Lhs: &tmp,
			Op:  op,
			Rhs: xlist.Lhs,
		}
		op = xlist.Op
	}

	return cmdList
}

func (p *Parser) parseXCommandList() vm.XCommandList {
	cmdList := vm.XCommandList{Lhs: p.parsePipeline()}
	for {
		switch p.peek().Kind {
		case lexer.TokLAnd:
			cmdList.Op = vm.LAnd
		case lexer.TokLOr:
			cmdList.Op = vm.LOr
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

func (p *Parser) parsePipeline() vm.Pipeline {
	pipe := vm.Pipeline{p.parseCommand()}

	for {
		switch p.peek().Kind {
		case lexer.TokPipe:
			p.next()
			pipe = append(pipe, p.parseCommand())
		case lexer.TokEndStmt:
			p.next()
		default:
			return pipe
		}
	}
}

func (p *Parser) parseCommand() vm.Command {
	switch t := p.peek(); {
	case t.Kind == lexer.TokArg && t.Val == "if":
		p.next()
		return p.parseIf()
	case t.Kind == lexer.TokArg && t.Val == "while":
		p.next()
		return p.parseWhile()
	case t.Kind == lexer.TokBcOpen:
		p.next()
		return p.parseCompound()
	}
	return p.parseSimple()
}

func (p *Parser) parseWhile() *vm.While {
	return &vm.While{
		Cond: p.parseCommandList(),
		Body: p.parseCommandList(),
	}
}

func (p *Parser) parseIf() *vm.If {
	if_ := vm.If{
		Cond: p.parseCommandList(),
		Body: p.parseCommandList(),
	}

	if t := p.peek(); t.Kind == lexer.TokArg && t.Val == "else" {
		p.next()
		for p.peek().Kind == lexer.TokEndStmt {
			p.next()
		}
		cl := p.parseCommandList()
		if_.Else = &cl
	}

	return &if_
}

func (p *Parser) parseCompound() *vm.Compound {
	cmds := make([]vm.CommandList, 0, 4) // Add a little capacity

	for {
		switch p.peek().Kind {
		case lexer.TokBcClose:
			p.next()
			return &vm.Compound{Cmds: cmds}
		case lexer.TokEndStmt:
			p.next()
		case lexer.TokEof:
			die(errExpected{"closing brace", p.peek()})
		default:
			cmds = append(cmds, p.parseCommandList())
		}
	}
}

func (p *Parser) parseSimple() *vm.Simple {
	args := make([]vm.Value, 0, 4) // Add a little capacity
	var redirs []vm.Redirect

	args = append(args, p.parseValue())
	for vm.IsValue(p.peek().Kind) {
		args = append(args, p.parseValue())
	}

	for {
		switch t := p.peek(); {
		case vm.IsRedir(t.Kind):
			p.next()
			r := vm.NewRedir(t.Kind)

			switch {
			case vm.IsValue(p.peek().Kind):
				r.File = p.parseValue()
			default:
				die(errExpected{"file after redirect", t})
			}

			redirs = append(redirs, r)
		case vm.IsValue(t.Kind):
			die(errExpected{"semicolon or newline", t})
		default:
			return &vm.Simple{
				Args:   args,
				Redirs: redirs,
			}
		}
	}
}

func (p *Parser) parseValue() vm.Value {
	var v vm.Value

	switch t := p.next(); t.Kind {
	case lexer.TokArg, lexer.TokString:
		v = vm.NewValue(t)
	case lexer.TokVarRef, lexer.TokVarFlat, lexer.TokVarLen:
		vr := vm.NewVarRef(t)
		if p.peek().Kind == lexer.TokBkOpen {
			vr.Indices = p.parseIndices()
		}
		v = vr
	case lexer.TokPOpen:
		v = p.parseList()
	default:
		die(errExpected{"value", t})
	}

	if p.peek().Kind == lexer.TokConcat {
		p.next()
		v = vm.Concat{Lhs: v, Rhs: p.parseValue()}
	}

	return v
}

func (p *Parser) parseIndices() []vm.Value {
	p.next() // Consume ‘[’
	xs := []vm.Value{}
	for vm.IsValue(p.peek().Kind) {
		xs = append(xs, p.parseValue())
	}
	if p.peek().Kind != lexer.TokBkClose {
		die(errExpected{"closing bracket", p.next()})
	}
	p.next()
	return xs
}

func (p *Parser) parseList() vm.List {
	xs := vm.List{}

	for {
		switch t := p.next(); t.Kind {
		case lexer.TokPClose:
			return xs
		case lexer.TokArg, lexer.TokString:
			xs = append(xs, vm.NewValue(t))
		case lexer.TokVarRef, lexer.TokVarFlat, lexer.TokVarLen:
			xs = append(xs, vm.NewVarRef(t))
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
