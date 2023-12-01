package main

type parser struct {
	stream <-chan token
	cache  *token
}

func newParser(c <-chan token) parser {
	return parser{stream: c}
}

func (p *parser) run() astProgram {
	return p.parseProgram()
}

func (p *parser) next() token {
	var t token
	if p.cache != nil {
		t, p.cache = *p.cache, nil
	} else {
		t = <-p.stream
	}
	return t
}

func (p *parser) peek() token {
	if p.cache != nil {
		return *p.cache
	}

	t, ok := <-p.stream
	if !ok {
		t = token{kind: tokEof}
	}
	p.cache = &t
	return t
}

func (p *parser) parseProgram() astProgram {
	var prog astProgram

	for {
		switch p.peek().kind {
		case tokEndStmt:
			p.next()
		case tokEof:
			return prog
		default:
			prog = append(prog, p.parseCommandList())
		}
	}
}

func (p *parser) parseCommandList() astCommandList {
	xlist := p.parseXCommandList()
	cmdList := astCommandList{lhs: nil, rhs: xlist.lhs}
	op := xlist.op

	for xlist.rhs != nil {
		xlist = *xlist.rhs
		tmp := cmdList
		cmdList = astCommandList{
			lhs: &tmp,
			op:  op,
			rhs: xlist.lhs,
		}
		op = xlist.op
	}

	return cmdList
}

func (p *parser) parseXCommandList() astXCommandList {
	cmdList := astXCommandList{lhs: p.parsePipeline()}

	for {
		switch p.peek().kind {
		case tokLAnd:
			cmdList.op = binAnd
		case tokLOr:
			cmdList.op = binOr
		default:
			return cmdList
		}

		p.next() // Consume operator
		for p.peek().kind == tokEndStmt {
			p.next()
		}

		rhs := p.parseXCommandList()
		cmdList.rhs = &rhs
	}
}

func (p *parser) parsePipeline() astPipeline {
	pipe := astPipeline{p.parseCommand()}

	for {
		switch p.peek().kind {
		case tokPipe:
			p.next()
			pipe = append(pipe, p.parseCommand())
		case tokEndStmt:
			p.next()
		default:
			return pipe
		}
	}
}

func (p *parser) parseCommand() astCleanCommand {
	var cmd astCommand

	switch t := p.peek(); {
	case t.kind == tokArg && t.val == "if":
		p.next()
		cmd = p.parseIf()
	case t.kind == tokArg && t.val == "while":
		p.next()
		cmd = p.parseWhile()
	case t.kind == tokBraceOpen:
		p.next()
		cmd = p.parseCompound()
	default:
		cmd = p.parseSimple()
	}

	var redirs []astRedirect
	for {
		switch t := p.peek(); {
		case isRedirTok(t.kind):
			p.next()
			r := newRedir(t.kind)

			switch {
			case isValueTok(p.peek().kind):
				r.file = p.parseValue()
			default:
				die(errExpected{"file after redirect", t})
			}

			redirs = append(redirs, r)
		case isValueTok(t.kind):
			die(errExpected{"semicolon or newline", t})
		default:
			cmd.setRedirs(redirs)
			return astCleanCommand{cmd: cmd}
		}
	}
}

func (p *parser) parseWhile() *astWhile {
	var w astWhile
	w.cond = p.parseCommandList()
	if t := p.next(); t.kind != tokBraceOpen {
		die(errExpected{"opening brace", t})
	}
	w.body = p.parseBody()
	return &w
}

func (p *parser) parseIf() *astIf {
	cond := astIf{cond: p.parseCommandList()}

	if t := p.next(); t.kind != tokBraceOpen {
		die(errExpected{"opening brace", t})
	}
	cond.body = p.parseBody()

	if t := p.peek(); t.kind != tokArg || t.val != "else" {
		goto out
	}
	p.next() // Consume ‘else’
	if t := p.peek(); t.kind == tokArg && t.val == "if" {
		p.next() // Consume ‘if’
		cond.else_ = append(cond.else_, astCommandList{
			rhs: astPipeline{astCleanCommand{cmd: p.parseIf()}},
		})
	} else {
		if t := p.next(); t.kind != tokBraceOpen {
			die(errExpected{"opening brace", t})
		}
		cond.else_ = p.parseBody()
	}

out:
	return &cond
}

func (p *parser) parseBody() []astCommandList {
	xs := []astCommandList{}

	for {
		switch t := p.peek(); t.kind {
		case tokEndStmt:
			p.next()
		case tokBraceClose:
			p.next()
			return xs
		case tokEof:
			die(errExpected{"closing brace", t})
		default:
			xs = append(xs, p.parseCommandList())
		}
	}
}

func (p *parser) parseCompound() *astCompound {
	cmds := make([]astCommandList, 0, 4) // Add a little capacity

	for {
		switch p.peek().kind {
		case tokBraceClose:
			p.next()
			return &astCompound{cmds: cmds}
		case tokEndStmt:
			p.next()
		case tokEof:
			die(errExpected{"closing brace", p.peek()})
		default:
			cmds = append(cmds, p.parseCommandList())
		}
	}
}

func (p *parser) parseSimple() *astSimple {
	args := make([]astValue, 0, 4) // Add a little capacity

	args = append(args, p.parseValue())
	for isValueTok(p.peek().kind) {
		args = append(args, p.parseValue())
	}

	return &astSimple{args: args}
}

func (p *parser) parseValue() astValue {
	var v astValue

	switch t := p.next(); t.kind {
	case tokArg:
		v = astArgument(t.val)
	case tokString:
		v = astString(t.val)
	case tokVarRef, tokVarFlat, tokVarLen:
		vr := newVarRef(t)
		if p.peek().kind == tokBracketOpen {
			vr.indices = p.parseIndices()
		}
		v = vr
	case tokParenOpen:
		v = p.parseList()
	case tokProcRead:
		v = &astProcRedir{kind: procRead, body: p.parseBody()}
	case tokProcWrite:
		v = &astProcRedir{kind: procWrite, body: p.parseBody()}
	case tokProcRdWr:
		v = &astProcRedir{
			kind: procRead | procWrite,
			body: p.parseBody(),
		}
	case tokProcSub:
		v = astProcSub{body: p.parseBody()}
	default:
		die(errExpected{"value", t})
	}

	if p.peek().kind == tokConcat {
		p.next()
		v = astConcat{lhs: v, rhs: p.parseValue()}
	}

	return v
}

func (p *parser) parseIndices() []astValue {
	var xs []astValue
	p.next() // Consume ‘[’
	for isValueTok(p.peek().kind) {
		xs = append(xs, p.parseValue())
	}
	if p.peek().kind != tokBracketClose {
		die(errExpected{"closing bracket", p.next()})
	}
	p.next()
	return xs
}

func (p *parser) parseList() astList {
	var xs astList

	for {
		switch t := p.next(); t.kind {
		case tokParenClose:
			return xs
		case tokArg:
			xs = append(xs, astArgument(t.val))
		case tokString:
			xs = append(xs, astString(t.val))
		case tokVarRef, tokVarFlat, tokVarLen:
			xs = append(xs, newVarRef(t))
		case tokParenOpen:
			xs = append(xs, p.parseList()...)
		default:
			die(errExpected{"list item", t})
		}
	}
}
