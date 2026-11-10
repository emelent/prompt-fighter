package sim

import "promptfighter/sim/dsl"

// host is what the interpreter needs from the fighter that runs it.
type host interface {
	// sense reads a sensor from the start-of-frame snapshot (bools as 0/1).
	sense(e *dsl.SensorExpr) int
	// startCheck occupies the interpreter for one sensor read and records a check event.
	startCheck(e *dsl.SensorExpr, value int)
	// tryAction starts the action if the current stance allows it. It returns
	// false when the action must wait for the stance to end.
	tryAction(st *dsl.ActionStmt) bool
	// startIdle occupies the interpreter for one idle frame (zero-cost loop guard).
	startIdle(line int)
	// emit records an interpreter event (script_end, handler_enter).
	emit(typ string, d Detail)
}

// --- compiled conditions ---

type opcode int

const (
	opSense opcode = iota
	opInt
	opCmp
	opNot
	opAndJump // if top is false jump to target keeping it, else pop
	opOrJump  // if top is true jump to target keeping it, else pop
)

type op struct {
	code   opcode
	sensor *dsl.SensorExpr
	val    int
	cmp    string
	target int
}

type program []op

func compile(e dsl.Expr) program {
	var p program
	var emit func(e dsl.Expr)
	emit = func(e dsl.Expr) {
		switch x := e.(type) {
		case *dsl.SensorExpr:
			p = append(p, op{code: opSense, sensor: x})
		case *dsl.IntLit:
			p = append(p, op{code: opInt, val: x.Val})
		case *dsl.NotExpr:
			emit(x.X)
			p = append(p, op{code: opNot})
		case *dsl.CmpExpr:
			emit(x.L)
			emit(x.R)
			p = append(p, op{code: opCmp, cmp: x.Op})
		case *dsl.LogicExpr:
			emit(x.L)
			j := len(p)
			code := opAndJump
			if x.Op == dsl.OpOr {
				code = opOrJump
			}
			p = append(p, op{code: code})
			emit(x.R)
			p[j].target = len(p)
		}
	}
	emit(e)
	return p
}

type evalState struct {
	prog  program
	pc    int
	stack []int
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// run executes until the condition finishes (done=true) or a sensor read
// starts (done=false; the interpreter is occupied for the read's cost).
func (ev *evalState) run(h host) (result, done bool) {
	for ev.pc < len(ev.prog) {
		o := ev.prog[ev.pc]
		ev.pc++
		switch o.code {
		case opSense:
			v := h.sense(o.sensor)
			ev.stack = append(ev.stack, v)
			h.startCheck(o.sensor, v)
			return false, false
		case opInt:
			ev.stack = append(ev.stack, o.val)
		case opNot:
			n := len(ev.stack) - 1
			ev.stack[n] = b2i(ev.stack[n] == 0)
		case opCmp:
			n := len(ev.stack)
			l, r := ev.stack[n-2], ev.stack[n-1]
			ev.stack = ev.stack[:n-2]
			var b bool
			switch o.cmp {
			case "<":
				b = l < r
			case "<=":
				b = l <= r
			case ">":
				b = l > r
			case ">=":
				b = l >= r
			case "==":
				b = l == r
			case "!=":
				b = l != r
			}
			ev.stack = append(ev.stack, b2i(b))
		case opAndJump, opOrJump:
			top := ev.stack[len(ev.stack)-1]
			if (o.code == opAndJump && top == 0) || (o.code == opOrJump && top != 0) {
				ev.pc = o.target
			} else {
				ev.stack = ev.stack[:len(ev.stack)-1]
			}
		}
	}
	return ev.stack[len(ev.stack)-1] != 0, true
}

// --- continuation stack ---

type frameKind int

const (
	fkMain frameKind = iota
	fkBlock
	fkLoop
	fkRepeat
	fkHandler
)

type iframe struct {
	kind      frameKind
	body      []dsl.Stmt
	idx       int
	left      int // repeat iterations remaining (including current)
	iterStart int // tick on which the current loop iteration started
	line      int // line of the owning statement (loop guard display)

	started bool       // the statement at idx is in progress
	branch  int        // if-chain branch being evaluated
	eval    *evalState // condition evaluation in progress
}

// Interp runs one fighter's script.
type Interp struct {
	script *dsl.Script
	progs  map[dsl.Expr]program

	stack     []*iframe
	saved     []*iframe // main-program resume point while handling an interrupt
	hasSaved  bool
	inHandler string // "" or "on_hit"/"on_block" while a handler runs
	pending   string // handler to run once actionable again
	endSent   bool

	// Line is the source line of the statement currently executing (0 if none).
	Line int
}

// NewInterp prepares an interpreter for a validated script.
func NewInterp(s *dsl.Script) *Interp {
	in := &Interp{script: s, progs: map[dsl.Expr]program{}}
	in.stack = []*iframe{{kind: fkMain, body: s.Main}}
	return in
}

func (in *Interp) prog(e dsl.Expr) program {
	p, ok := in.progs[e]
	if !ok {
		p = compile(e)
		in.progs[e] = p
	}
	return p
}

func lineOf(st dsl.Stmt) int { return st.StmtPos().Line }

// Ended reports whether the main program has finished.
func (in *Interp) Ended() bool { return in.endSent && len(in.stack) == 0 && in.pending == "" }

// Complete is called when the action started by the current statement finishes.
func (in *Interp) Complete() {
	if len(in.stack) == 0 {
		return
	}
	in.advance(in.stack[len(in.stack)-1])
}

func (in *Interp) advance(f *iframe) {
	f.idx++
	f.started = false
	f.eval = nil
	f.branch = 0
}

func (in *Interp) push(f *iframe) { in.stack = append(in.stack, f) }

// pop removes the top frame and advances past its owning statement.
func (in *Interp) pop() {
	in.stack = in.stack[:len(in.stack)-1]
	if n := len(in.stack); n > 0 {
		in.advance(in.stack[n-1])
	}
}

// Interrupt abandons the current statement because the fighter was hit
// (handler "on_hit") or its attack was blocked ("on_block").
func (in *Interp) Interrupt(handler string) {
	if !in.hasSaved && in.inHandler == "" {
		saved := make([]*iframe, len(in.stack))
		for i, f := range in.stack {
			c := *f
			saved[i] = &c
		}
		if n := len(saved); n > 0 && saved[n-1].started {
			in.advance(saved[n-1])
		}
		in.saved, in.hasSaved = saved, true
	}
	in.stack = nil
	in.inHandler = ""
	in.pending = handler
	in.Line = 0
}

// Pending reports whether an interrupt is waiting to be resumed.
func (in *Interp) Pending() bool { return in.pending != "" }

func (in *Interp) restoreMain() {
	in.stack, in.saved, in.hasSaved = in.saved, nil, false
	in.inHandler = ""
}

// resume runs the pending handler (if defined) or returns to the main program.
func (in *Interp) resume(h host) {
	kind := in.pending
	in.pending = ""
	var body *dsl.Block
	if kind == "on_hit" {
		body = in.script.OnHit
	} else {
		body = in.script.OnBlock
	}
	if body == nil {
		in.restoreMain()
		return
	}
	h.emit("handler_enter", Detail{Handler: kind})
	in.inHandler = kind
	in.stack = []*iframe{{kind: fkHandler, body: body.Stmts}}
}

const maxStepsPerFrame = 4096

// Step advances the interpreter on tick t until something occupies the
// fighter for the frame (an action, a sensor read, an idle frame) or the
// statement must wait for a stance to end.
func (in *Interp) Step(t int, h host) {
	if in.pending != "" {
		in.resume(h)
	}
	for n := 0; n < maxStepsPerFrame; n++ {
		if len(in.stack) == 0 {
			if in.inHandler != "" {
				in.restoreMain()
				continue
			}
			if !in.endSent {
				in.endSent = true
				h.emit("script_end", Detail{})
			}
			in.Line = 0
			return
		}
		top := in.stack[len(in.stack)-1]
		if top.idx >= len(top.body) {
			switch top.kind {
			case fkLoop:
				if top.iterStart == t {
					top.idx, top.iterStart = 0, t+1
					in.Line = top.line
					h.startIdle(top.line)
					return
				}
				top.idx, top.iterStart = 0, t
			case fkRepeat:
				top.left--
				if top.left > 0 {
					top.idx = 0
				} else {
					in.pop()
				}
			default:
				in.pop()
			}
			continue
		}
		st := top.body[top.idx]
		switch s := st.(type) {
		case *dsl.ActionStmt:
			top.started = true
			in.Line = s.Pos.Line
			h.tryAction(s)
			return
		case *dsl.LoopStmt:
			top.started = true
			in.push(&iframe{kind: fkLoop, body: s.Body.Stmts, iterStart: t, line: lineOf(s)})
		case *dsl.RepeatStmt:
			top.started = true
			in.push(&iframe{kind: fkRepeat, body: s.Body.Stmts, left: s.N, line: lineOf(s)})
		case *dsl.IfStmt:
			if !top.started {
				top.started = true
				top.branch = 0
				top.eval = &evalState{prog: in.prog(s.Branches[0].Cond)}
			}
			if top.eval == nil { // defensive: should not happen
				in.advance(top)
				continue
			}
			res, done := top.eval.run(h)
			if !done {
				in.Line = top.eval.prog[top.eval.pc-1].sensor.Pos.Line
				return
			}
			if res {
				top.eval = nil
				in.push(&iframe{kind: fkBlock, body: s.Branches[top.branch].Body.Stmts})
				continue
			}
			top.branch++
			if top.branch < len(s.Branches) {
				top.eval = &evalState{prog: in.prog(s.Branches[top.branch].Cond)}
				continue
			}
			top.eval = nil
			if s.Else != nil {
				in.push(&iframe{kind: fkBlock, body: s.Else.Stmts})
			} else {
				in.advance(top)
			}
		}
	}
	h.startIdle(in.Line)
}
