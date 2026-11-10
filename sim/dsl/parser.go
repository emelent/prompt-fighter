package dsl

import (
	"fmt"
	"strings"
)

const maxParseErrors = 50

type parser struct {
	toks []Token
	i    int
	errs []Error
	s    *Script
}

func (p *parser) peek() Token { return p.toks[p.i] }
func (p *parser) peekN(n int) Token {
	if p.i+n < len(p.toks) {
		return p.toks[p.i+n]
	}
	return p.toks[len(p.toks)-1]
}
func (p *parser) next() Token {
	t := p.toks[p.i]
	if t.Kind != TEOF {
		p.i++
	}
	return t
}
func (p *parser) isIdent(text string) bool {
	t := p.peek()
	return t.Kind == TIdent && t.Text == text
}
func (p *parser) errorf(pos Pos, format string, args ...any) {
	if len(p.errs) < maxParseErrors {
		p.errs = append(p.errs, errAt(pos, format, args...))
	}
}

func describe(t Token) string {
	switch t.Kind {
	case TEOF:
		return "end of script"
	case TInt, TCount, TIdent:
		return fmt.Sprintf("%q", t.Text)
	}
	return "`" + t.Text + "`"
}

func isStmtStartWord(w string) bool {
	if _, ok := ActionByName(w); ok {
		return true
	}
	switch w {
	case "if", "repeat", "loop", "on_hit", "on_block":
		return true
	}
	return false
}

// sync skips tokens until something that can start a statement, a `}` or EOF.
func (p *parser) sync() {
	for {
		t := p.peek()
		if t.Kind == TEOF || t.Kind == TRBrace || (t.Kind == TIdent && isStmtStartWord(t.Text)) {
			return
		}
		p.next()
	}
}

// parse builds the AST for src (syntax errors only; see Parse for full validation).
func parse(src string) (*Script, []Error) {
	toks, lexErrs := Lex(src)
	p := &parser{toks: toks, s: &Script{}}
	p.errs = append(p.errs, lexErrs...)
	for p.peek().Kind != TEOF {
		t := p.peek()
		switch {
		case t.Kind == TRBrace:
			p.errorf(t.Pos, "unexpected `}` — there is no open block to close")
			p.next()
		case t.Kind == TIdent && (t.Text == "on_hit" || t.Text == "on_block"):
			p.parseHandler(true)
		default:
			if st := p.parseStmt(); st != nil {
				p.s.Main = append(p.s.Main, st)
			}
		}
	}
	return p.s, p.errs
}

func (p *parser) parseHandler(top bool) {
	t := p.next()
	if !top {
		p.errorf(t.Pos, "handlers must be declared at the top level of the script, not inside a block")
		p.parseBlock(t.Text)
		return
	}
	b := p.parseBlock(t.Text)
	if t.Text == "on_hit" {
		if p.s.OnHit != nil {
			p.errorf(t.Pos, "only one `on_hit` handler is allowed")
			return
		}
		p.s.OnHit, p.s.OnHitPos = b, t.Pos
	} else {
		if p.s.OnBlock != nil {
			p.errorf(t.Pos, "only one `on_block` handler is allowed")
			return
		}
		p.s.OnBlock, p.s.OnBlockPos = b, t.Pos
	}
}

func (p *parser) parseBlock(owner string) *Block {
	t := p.peek()
	if t.Kind != TLBrace {
		p.errorf(t.Pos, "expected `{` to start the block of `%s`, found %s", owner, describe(t))
		return &Block{LBrace: t.Pos}
	}
	p.next()
	b := &Block{LBrace: t.Pos}
	for {
		n := p.peek()
		switch {
		case n.Kind == TRBrace:
			p.next()
			return b
		case n.Kind == TEOF:
			p.errorf(n.Pos, "missing `}` to close the block opened at %d:%d", t.Pos.Line, t.Pos.Col)
			return b
		case n.Kind == TIdent && (n.Text == "on_hit" || n.Text == "on_block"):
			p.parseHandler(false)
			b.HadErrors = true
		default:
			if st := p.parseStmt(); st != nil {
				b.Stmts = append(b.Stmts, st)
			} else {
				b.HadErrors = true
			}
		}
	}
}

var overlayHints = map[string]string{
	"jump_kick":  "`jump_kick` is not a statement — it happens when you `kick` while airborne, e.g. `jump_forward kick`",
	"dive_punch": "`dive_punch` is not a statement — it happens when you `punch` while airborne, e.g. `jump_forward punch`",
	"low_jab":    "`low_jab` is not a statement — it happens when you `punch` while ducking, e.g. `duck hold 10 punch`",
}

// skipArgs consumes argument-like tokens after a bad statement word.
func (p *parser) skipArgs() {
	for {
		t := p.peek()
		if t.Kind == TCount || t.Kind == TInt {
			p.next()
			continue
		}
		if t.Kind == TIdent && t.Text == "hold" {
			p.next()
			continue
		}
		return
	}
}

func (p *parser) parseStmt() Stmt {
	t := p.next()
	if t.Kind != TIdent {
		p.errorf(t.Pos, "expected a statement (an action like `kick`, or `if`/`repeat`/`loop`), found %s", describe(t))
		p.sync()
		return nil
	}
	switch t.Text {
	case "if":
		return p.parseIf(t.Pos)
	case "elif", "else":
		p.errorf(t.Pos, "`%s` must follow an `if` or `elif` block", t.Text)
		if t.Text == "elif" || p.isIdent("if") {
			if t.Text == "else" {
				p.next()
			}
			p.parseExpr()
		}
		p.parseBlock(t.Text)
		return nil
	case "repeat":
		st := &RepeatStmt{Pos: t.Pos}
		n := p.peek()
		switch n.Kind {
		case TInt:
			p.next()
			st.N, st.NPos = n.Val, n.Pos
		case TCount:
			p.next()
			p.errorf(n.Pos, "write `repeat %d` without the `x`", n.Val)
			st.N, st.NPos = n.Val, n.Pos
		default:
			p.errorf(n.Pos, "`repeat` requires a count, e.g. `repeat 3 { kick }`")
			st.N, st.NPos = 1, n.Pos
		}
		st.Body = p.parseBlock("repeat")
		return st
	case "loop":
		return &LoopStmt{Pos: t.Pos, Body: p.parseBlock("loop")}
	case "while":
		p.errorf(t.Pos, "there is no `while`; use `loop { ... }` with an `if` inside, or `repeat N { ... }`")
		p.parseExpr()
		p.parseBlock("while")
		return nil
	case "for":
		p.errorf(t.Pos, "there is no `for`; use `repeat N { ... }`")
		p.sync()
		return nil
	}
	if hint, ok := overlayHints[t.Text]; ok {
		p.errorf(t.Pos, "%s", hint)
		p.skipArgs()
		return nil
	}
	a, ok := ActionByName(t.Text)
	if !ok {
		if s := suggest(t.Text, actionAliases, actionNames[:], stmtKeywords); s != "" {
			p.errorf(t.Pos, "unknown action %q — did you mean %q?", t.Text, s)
		} else {
			p.errorf(t.Pos, "unknown action %q; valid actions are: %s", t.Text, strings.Join(actionNames[:], ", "))
		}
		p.skipArgs()
		return nil
	}
	return p.parseActionArgs(t.Pos, a)
}

func (p *parser) parseActionArgs(pos Pos, a Action) *ActionStmt {
	st := &ActionStmt{Pos: pos, Action: a}
	n := p.peek()
	set := func(tok Token) { st.N, st.HasN, st.NPos = tok.Val, true, tok.Pos }
	isHold := n.Kind == TIdent && n.Text == "hold"
	switch a.Arg() {
	case ArgNone:
		switch {
		case n.Kind == TCount:
			p.next()
			p.errorf(n.Pos, "`%s` takes no count — use `repeat %d { %s }`", a, n.Val, a)
		case n.Kind == TInt:
			p.next()
			p.errorf(n.Pos, "`%s` takes no number — use `repeat %d { %s }` to do it several times", a, n.Val, a)
		case isHold:
			p.next()
			p.errorf(n.Pos, "`%s` does not take `hold`; only `duck`, `high_block` and `low_block` do", a)
			if p.peek().Kind == TInt {
				p.next()
			}
		}
	case ArgHold:
		switch {
		case isHold:
			p.next()
			if v := p.peek(); v.Kind == TInt {
				p.next()
				set(v)
			} else {
				p.errorf(v.Pos, "`hold` must be followed by a frame count, e.g. `hold 6`")
			}
		case n.Kind == TCount:
			p.next()
			p.errorf(n.Pos, "`%s` takes `hold N`, not `xN` — e.g. `%s hold %d`", a, a, n.Val)
			set(n)
		case n.Kind == TInt:
			p.next()
			p.errorf(n.Pos, "write `%s hold %d` — the frame count needs `hold` in front", a, n.Val)
			set(n)
		}
	case ArgCount:
		switch {
		case n.Kind == TCount:
			p.next()
			set(n)
		case n.Kind == TInt:
			p.next()
			p.errorf(n.Pos, "write `%s x%d` — the count needs an `x` in front", a, n.Val)
			set(n)
		case isHold:
			p.next()
			p.errorf(n.Pos, "`%s` does not take `hold`; use a count like `%s x3`", a, a)
			if v := p.peek(); v.Kind == TInt {
				p.next()
				set(v)
			}
		}
	case ArgWait:
		switch {
		case n.Kind == TInt:
			p.next()
			set(n)
		case n.Kind == TCount:
			p.next()
			p.errorf(n.Pos, "write `wait %d` without the `x`", n.Val)
			set(n)
		default:
			p.errorf(pos, "`wait` requires a frame count such as `wait 4`")
		}
	}
	return st
}

func (p *parser) parseIf(pos Pos) Stmt {
	st := &IfStmt{Pos: pos}
	cond := p.parseExpr()
	st.Branches = append(st.Branches, Branch{Pos: pos, Cond: cond, Body: p.parseBlock("if")})
	for {
		switch {
		case p.isIdent("elif"):
			t := p.next()
			c := p.parseExpr()
			st.Branches = append(st.Branches, Branch{Pos: t.Pos, Cond: c, Body: p.parseBlock("elif")})
		case p.isIdent("else"):
			p.next()
			if p.isIdent("if") {
				t := p.next()
				p.errorf(t.Pos, "use `elif` instead of `else if`")
				c := p.parseExpr()
				st.Branches = append(st.Branches, Branch{Pos: t.Pos, Cond: c, Body: p.parseBlock("elif")})
				continue
			}
			st.Else = p.parseBlock("else")
			return st
		default:
			return st
		}
	}
}

// --- expressions: or > and > not > comparison > primary ---

func (p *parser) parseExpr() Expr { return p.parseOr() }

func (p *parser) parseOr() Expr {
	l := p.parseAnd()
	for p.isIdent("or") {
		t := p.next()
		r := p.parseAnd()
		l = &LogicExpr{Pos: t.Pos, Op: OpOr, L: l, R: r}
	}
	return l
}

func (p *parser) parseAnd() Expr {
	l := p.parseNot()
	for p.isIdent("and") {
		t := p.next()
		r := p.parseNot()
		l = &LogicExpr{Pos: t.Pos, Op: OpAnd, L: l, R: r}
	}
	return l
}

func (p *parser) parseNot() Expr {
	if p.isIdent("not") {
		t := p.next()
		return &NotExpr{Pos: t.Pos, X: p.parseNot()}
	}
	return p.parseCmp()
}

func (p *parser) parseCmp() Expr {
	l := p.parsePrimary()
	if p.peek().Kind != TCmp {
		return l
	}
	op := p.next()
	r := p.parsePrimary()
	e := &CmpExpr{Pos: op.Pos, Op: op.Text, L: l, R: r}
	for p.peek().Kind == TCmp {
		op2 := p.next()
		p.errorf(op2.Pos, "comparisons cannot be chained; write `a < b and b < c` instead")
		p.parsePrimary()
	}
	return e
}

func (p *parser) parsePrimary() Expr {
	t := p.peek()
	switch t.Kind {
	case TLParen:
		p.next()
		e := p.parseOr()
		if c := p.peek(); c.Kind == TRParen {
			p.next()
		} else {
			p.errorf(c.Pos, "missing `)` to close the `(` at %d:%d", t.Pos.Line, t.Pos.Col)
		}
		return e
	case TInt:
		p.next()
		return &IntLit{Pos: t.Pos, Val: t.Val}
	case TIdent:
		p.next()
		switch t.Text {
		case "in_range":
			return p.parseInRange(t)
		case "true", "false":
			p.errorf(t.Pos, "`%s` is not supported; use a sensor such as `enemy.airborne` or a comparison such as `distance < 10`", t.Text)
			return nil
		}
		if s, ok := SensorByName(t.Text); ok {
			return &SensorExpr{Pos: t.Pos, Sensor: s}
		}
		if _, ok := AttackByName(t.Text); ok {
			p.errorf(t.Pos, "%q is an attack, not a condition — did you mean `in_range(%s)`?", t.Text, t.Text)
			return nil
		}
		if s := suggest(t.Text, sensorAliases, sensorNames[:]); s != "" {
			p.errorf(t.Pos, "unknown sensor %q — did you mean %q?", t.Text, s)
		} else {
			p.errorf(t.Pos, "unknown sensor %q; valid sensors are: %s", t.Text, strings.Join(sensorNames[:], ", "))
		}
		return nil
	case TLBrace:
		p.errorf(t.Pos, "missing condition before `{`")
		return nil
	}
	p.errorf(t.Pos, "expected a condition, found %s", describe(t))
	if t.Kind != TEOF && t.Kind != TRBrace {
		p.next()
	}
	return nil
}

func (p *parser) parseInRange(t Token) Expr {
	attackList := strings.Join(attackNames[:], ", ")
	if p.peek().Kind != TLParen {
		p.errorf(p.peek().Pos, "`in_range` needs an attack name in parentheses, e.g. `in_range(kick)`")
		return nil
	}
	p.next()
	a := p.peek()
	var e Expr
	if a.Kind == TIdent {
		p.next()
		if atk, ok := AttackByName(a.Text); ok {
			e = &SensorExpr{Pos: t.Pos, Sensor: SenInRange, Arg: atk}
		} else {
			p.errorf(a.Pos, "`in_range` needs an attack name: %s", attackList)
		}
	} else {
		p.errorf(a.Pos, "`in_range` needs an attack name: %s", attackList)
	}
	if c := p.peek(); c.Kind == TRParen {
		p.next()
	} else {
		p.errorf(c.Pos, "missing `)` after the attack name in `in_range(...)`")
	}
	return e
}
