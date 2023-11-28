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

func (p *Parser) parseCommand() vm.CleanCommand {
	var cmd vm.Command

	switch t := p.peek(); {
	case t.Kind == lexer.TokArg && t.Val == "if":
		p.next()
		cmd = p.parseIf()
	case t.Kind == lexer.TokArg && t.Val == "while":
		p.next()
		cmd = p.parseWhile()
	case t.Kind == lexer.TokBcOpen:
		p.next()
		cmd = p.parseCompound()
	default:
		cmd = p.parseSimple()
	}

	redirs := []vm.Redirect{}
	for {
		switch t := p.peek(); {
		case lexer.IsRedir(t.Kind):
			p.next()
			r := vm.NewRedir(t.Kind)

			switch {
			case lexer.IsValue(p.peek().Kind):
				r.File = p.parseValue()
			default:
				die(errExpected{"file after redirect", t})
			}

			redirs = append(redirs, r)
		case lexer.IsValue(t.Kind):
			die(errExpected{"semicolon or newline", t})
		default:
			cmd.SetRedirs(redirs)
			return vm.CleanCommand{Cmd: cmd}
		}
	}
}

func (p *Parser) parseWhile() *vm.While {
	w := vm.While{}
	w.Cond = p.parseCommandList()
	if t := p.next(); t.Kind != lexer.TokBcOpen {
		die(errExpected{"opening brace", t})
	}
	w.Body = p.parseBody()
	return &w
}

func (p *Parser) parseIf() *vm.If {
	cond := vm.If{}
	cond.Cond = p.parseCommandList()

	if t := p.next(); t.Kind != lexer.TokBcOpen {
		die(errExpected{"opening brace", t})
	}
	cond.Body = p.parseBody()

	if t := p.peek(); t.Kind != lexer.TokArg || t.Val != "else" {
		goto out
	}
	p.next() // Consume ‘else’
	if t := p.peek(); t.Kind == lexer.TokArg && t.Val == "if" {
		p.next() // Consume ‘if’
		cond.Else = append(cond.Else, vm.CommandList{
			Rhs: vm.Pipeline{vm.CleanCommand{Cmd: p.parseIf()}},
		})
	} else {
		if t := p.next(); t.Kind != lexer.TokBcOpen {
			die(errExpected{"opening brace", t})
		}
		cond.Else = p.parseBody()
	}

out:
	return &cond
}

func (p *Parser) parseBody() []vm.CommandList {
	xs := []vm.CommandList{}

	for {
		switch t := p.peek(); t.Kind {
		case lexer.TokEndStmt:
			p.next()
		case lexer.TokBcClose:
			p.next()
			return xs
		case lexer.TokEof:
			die(errExpected{"closing brace", t})
		default:
			xs = append(xs, p.parseCommandList())
		}
	}
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

	args = append(args, p.parseValue())
	for lexer.IsValue(p.peek().Kind) {
		args = append(args, p.parseValue())
	}

	return &vm.Simple{Args: args}
}

func (p *Parser) parseValue() vm.Value {
	var v vm.Value

	switch t := p.next(); t.Kind {
	case lexer.TokArg:
		v = vm.Argument(t.Val)
	case lexer.TokString:
		v = vm.String(t.Val)
	case lexer.TokVarRef, lexer.TokVarFlat, lexer.TokVarLen:
		vr := vm.NewVarRef(t)
		if p.peek().Kind == lexer.TokBkOpen {
			vr.Indices = p.parseIndices()
		}
		v = vr
	case lexer.TokPOpen:
		v = p.parseList()
	case lexer.TokProcRead:
		v = &vm.ProcRedir{Type: vm.ProcRead, Body: p.parseBody()}
	case lexer.TokProcWrite:
		v = &vm.ProcRedir{Type: vm.ProcWrite, Body: p.parseBody()}
	case lexer.TokProcRdWr:
		v = &vm.ProcRedir{
			Type: vm.ProcRead | vm.ProcWrite,
			Body: p.parseBody(),
		}
	case lexer.TokProcSub:
		v = vm.ProcSub{Body: p.parseBody()}
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
	for lexer.IsValue(p.peek().Kind) {
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
		case lexer.TokArg:
			xs = append(xs, vm.Argument(t.Val))
		case lexer.TokString:
			xs = append(xs, vm.String(t.Val))
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
