package dsl

import (
	"fmt"
	"sort"
)

var sprintf = fmt.Sprintf

// Limits are the static script limits enforced by the validator.
type Limits struct {
	MaxBytes      int `json:"max_bytes"`
	MaxStatements int `json:"max_statements"`
	MaxDepth      int `json:"max_depth"`
	MaxCount      int `json:"max_count"`   // upper bound for xN, hold N, wait N, repeat N (lower bound is 1)
	MaxLiteral    int `json:"max_literal"` // upper bound for integer literals in conditions (lower bound is 0)
	MaxErrors     int `json:"max_errors"`
}

// DefaultLimits returns the build-spec limits.
func DefaultLimits() Limits {
	return Limits{MaxBytes: 4096, MaxStatements: 40, MaxDepth: 4, MaxCount: 100, MaxLiteral: 999, MaxErrors: 10}
}

// Parse lexes, parses and validates src. The script is only usable when the
// returned error list is empty.
func Parse(src string, lim Limits) (*Script, []Error) {
	if len(src) > lim.MaxBytes {
		return nil, []Error{{Line: 1, Col: 1, Msg: sprintf("script is %d bytes; the maximum is %d bytes — shorten it (remove comments, use `loop`/`repeat`)", len(src), lim.MaxBytes)}}
	}
	s, errs := parse(src)
	v := &validator{lim: lim}
	v.script(s)
	errs = append(errs, v.errs...)
	return s, sortErrors(errs, lim.MaxErrors)
}

// Validate returns the validation errors for src (empty when valid).
func Validate(src string, lim Limits) []Error {
	_, errs := Parse(src, lim)
	return errs
}

type validator struct {
	lim      Limits
	errs     []Error
	counted  []Pos
	inHandle bool
}

func (v *validator) errorf(p Pos, format string, args ...any) {
	v.errs = append(v.errs, errAt(p, format, args...))
}

func (v *validator) script(s *Script) {
	v.block(s.Main, 0)
	v.inHandle = true
	if s.OnHit != nil {
		v.body(s.OnHit, 1)
	}
	if s.OnBlock != nil {
		v.body(s.OnBlock, 1)
	}
	if len(v.counted) > v.lim.MaxStatements {
		sort.Slice(v.counted, func(i, j int) bool { return v.counted[i].before(v.counted[j]) })
		v.errorf(v.counted[v.lim.MaxStatements], "script has %d statements; the maximum is %d — use `repeat` or `loop` to shorten it", len(v.counted), v.lim.MaxStatements)
	}
}

// body validates a braced block whose statements sit at the given depth.
func (v *validator) body(b *Block, depth int) {
	if depth > v.lim.MaxDepth {
		v.errorf(b.LBrace, "blocks are nested too deeply (depth %d); the maximum nesting depth is %d — flatten the script", depth, v.lim.MaxDepth)
		return
	}
	v.block(b.Stmts, depth)
}

func (v *validator) count(p Pos) { v.counted = append(v.counted, p) }

func (v *validator) checkCount(n int, p Pos, what string) {
	if n < 1 || n > v.lim.MaxCount {
		v.errorf(p, "%s count must be between 1 and %d (got %d)", what, v.lim.MaxCount, n)
	}
}

func (v *validator) block(stmts []Stmt, depth int) {
	for _, st := range stmts {
		switch s := st.(type) {
		case *ActionStmt:
			v.count(s.Pos)
			if s.HasN {
				v.checkCount(s.N, s.NPos, "`"+s.Action.String()+"`")
			}
		case *IfStmt:
			for _, br := range s.Branches {
				v.count(br.Pos)
				v.cond(br.Cond)
				v.body(br.Body, depth+1)
			}
			if s.Else != nil {
				v.body(s.Else, depth+1)
			}
		case *RepeatStmt:
			v.count(s.Pos)
			v.checkCount(s.N, s.NPos, "`repeat`")
			v.body(s.Body, depth+1)
		case *LoopStmt:
			v.count(s.Pos)
			if v.inHandle {
				v.errorf(s.Pos, "`loop` is not allowed inside handlers because a handler must finish so the main script can resume")
			}
			if len(s.Body.Stmts) == 0 && !s.Body.HadErrors {
				v.errorf(s.Pos, "a `loop` body must contain at least one statement")
			}
			v.body(s.Body, depth+1)
		}
	}
}

type exprType int

const (
	tUnknown exprType = iota
	tInt
	tBool
)

func exprName(e Expr) string {
	switch x := e.(type) {
	case *SensorExpr:
		return x.Name()
	case *IntLit:
		return sprintf("%d", x.Val)
	}
	return ""
}

// cond validates a branch condition, which must be boolean.
func (v *validator) cond(e Expr) {
	v.wantBool(e)
}

func (v *validator) wantBool(e Expr) {
	if e == nil {
		return
	}
	if t := v.typeOf(e); t == tInt {
		if n := exprName(e); n != "" {
			if _, isLit := e.(*IntLit); isLit {
				v.errorf(e.ExprPos(), "a number cannot be used as a condition; compare a sensor to it, e.g. `distance < %s`", n)
			} else {
				v.errorf(e.ExprPos(), "`%s` is a number and must be compared, e.g. `%s < 30`", n, n)
			}
		} else {
			v.errorf(e.ExprPos(), "this is a number and must be compared to be used as a condition")
		}
	}
}

func (v *validator) wantInt(e Expr) {
	if e == nil {
		return
	}
	if t := v.typeOf(e); t == tBool {
		if n := exprName(e); n != "" {
			v.errorf(e.ExprPos(), "`%s` is true/false and should be used directly, e.g. `if %s { ... }`", n, n)
		} else {
			v.errorf(e.ExprPos(), "comparisons need numbers on both sides; this side is true/false")
		}
	}
}

// typeOf returns the type of e, reporting errors in its sub-expressions.
func (v *validator) typeOf(e Expr) exprType {
	switch x := e.(type) {
	case nil:
		return tUnknown
	case *IntLit:
		if x.Val > v.lim.MaxLiteral {
			v.errorf(x.Pos, "number %d is too large; numbers in conditions must be between 0 and %d", x.Val, v.lim.MaxLiteral)
		}
		return tInt
	case *SensorExpr:
		if x.Sensor.IsBool() {
			return tBool
		}
		return tInt
	case *NotExpr:
		v.wantBool(x.X)
		return tBool
	case *LogicExpr:
		v.wantBool(x.L)
		v.wantBool(x.R)
		return tBool
	case *CmpExpr:
		v.wantInt(x.L)
		v.wantInt(x.R)
		return tBool
	}
	return tUnknown
}
