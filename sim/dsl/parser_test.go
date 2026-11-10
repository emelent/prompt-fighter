package dsl

import (
	"fmt"
	"strings"
	"testing"
)

func dumpExpr(e Expr) string {
	switch x := e.(type) {
	case nil:
		return "<nil>"
	case *LogicExpr:
		op := "and"
		if x.Op == OpOr {
			op = "or"
		}
		return fmt.Sprintf("(%s %s %s)", op, dumpExpr(x.L), dumpExpr(x.R))
	case *NotExpr:
		return fmt.Sprintf("(not %s)", dumpExpr(x.X))
	case *CmpExpr:
		return fmt.Sprintf("(%s %s %s)", x.Op, dumpExpr(x.L), dumpExpr(x.R))
	case *SensorExpr:
		return x.Name()
	case *IntLit:
		return fmt.Sprint(x.Val)
	}
	return "?"
}

func dumpStmts(ss []Stmt) string {
	var parts []string
	for _, s := range ss {
		parts = append(parts, dumpStmt(s))
	}
	return strings.Join(parts, " ")
}

func dumpStmt(s Stmt) string {
	switch x := s.(type) {
	case *ActionStmt:
		if x.HasN {
			return fmt.Sprintf("%s:%d", x.Action, x.N)
		}
		return x.Action.String()
	case *IfStmt:
		var b strings.Builder
		for i, br := range x.Branches {
			kw := "if"
			if i > 0 {
				kw = "elif"
			}
			fmt.Fprintf(&b, "%s %s [%s] ", kw, dumpExpr(br.Cond), dumpStmts(br.Body.Stmts))
		}
		if x.Else != nil {
			fmt.Fprintf(&b, "else [%s] ", dumpStmts(x.Else.Stmts))
		}
		return "{" + strings.TrimSpace(b.String()) + "}"
	case *RepeatStmt:
		return fmt.Sprintf("repeat %d [%s]", x.N, dumpStmts(x.Body.Stmts))
	case *LoopStmt:
		return fmt.Sprintf("loop [%s]", dumpStmts(x.Body.Stmts))
	}
	return "?"
}

func dumpScript(s *Script) string {
	out := dumpStmts(s.Main)
	if s.OnHit != nil {
		out += " on_hit[" + dumpStmts(s.OnHit.Stmts) + "]"
	}
	if s.OnBlock != nil {
		out += " on_block[" + dumpStmts(s.OnBlock.Stmts) + "]"
	}
	return out
}

func TestParseShapes(t *testing.T) {
	tests := []struct {
		name, src, want string
	}{
		{"actions", "kick move_forward x3 move_back high_block hold 6 duck wait 4 jump_forward",
			"kick move_forward:3 move_back high_block:6 duck wait:4 jump_forward"},
		{"build spec example", "loop {\n  if in_range(sweep) and not enemy.airborne { sweep }\n  elif enemy.airborne { high_block hold 6 }\n  elif health < 30 { move_back x10 }\n  else { move_forward x3 }\n}\non_hit { move_back x2 }\non_block { throw }",
			"loop [{if (and in_range(sweep) (not enemy.airborne)) [sweep] elif enemy.airborne [high_block:6] elif (< health 30) [move_back:10] else [move_forward:3]}] on_hit[move_back:2] on_block[throw]"},
		{"single line equals multi line", "if enemy.ducking { sweep } else { kick }", "{if enemy.ducking [sweep] else [kick]}"},
		{"precedence or/and/not/cmp", "if not distance < 5 or enemy.ducking and enemy.blocking { kick }",
			"{if (or (not (< distance 5)) (and enemy.ducking enemy.blocking)) [kick]}"},
		{"parens", "if (distance <= 12 or enemy.attacking) and not enemy.knocked_down { kick }",
			"{if (and (or (<= distance 12) enemy.attacking) (not enemy.knocked_down)) [kick]}"},
		{"repeat", "repeat 3 { punch }", "repeat 3 [punch]"},
		{"handlers anywhere", "on_block { throw } kick on_hit { wait 1 }", "kick on_hit[wait:1] on_block[throw]"},
		{"empty script", "# nothing\n", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, errs := Parse(tt.src, DefaultLimits())
			if len(errs) != 0 {
				t.Fatalf("unexpected errors: %v", errs)
			}
			if got := dumpScript(s); got != tt.want {
				t.Errorf("got  %s\nwant %s", got, tt.want)
			}
		})
	}
}

func TestParseSingleLineMatchesMultiLine(t *testing.T) {
	a, _ := Parse("if enemy.ducking { sweep } else { kick }", DefaultLimits())
	b, _ := Parse("if enemy.ducking {\n  sweep\n}\nelse {\n  kick\n}\n", DefaultLimits())
	if dumpScript(a) != dumpScript(b) {
		t.Errorf("%s != %s", dumpScript(a), dumpScript(b))
	}
}

func TestParseRecoveryMultipleErrors(t *testing.T) {
	tests := []struct {
		name, src string
		want      []string
	}{
		{"unknown action and missing brace", "kick\nkik\nloop { punch",
			[]string{`2:1: unknown action "kik" — did you mean "kick"?`, "3:13: missing `}` to close the block opened at 3:6"}},
		{"recovers after bad token", "kick 5 punch\n} wait 2",
			[]string{"1:6: `kick` takes no number — use `repeat 5 { kick }` to do it several times", "2:1: unexpected `}` — there is no open block to close"}},
		{"while hint", "while enemy.ducking { sweep }",
			[]string{"1:1: there is no `while`; use `loop { ... }` with an `if` inside, or `repeat N { ... }`"}},
		{"chained comparison", "if 1 < distance < 9 { kick }",
			[]string{"1:17: comparisons cannot be chained; write `a < b and b < c` instead"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := Validate(tt.src, DefaultLimits())
			var got []string
			for _, e := range errs {
				got = append(got, e.String())
			}
			if strings.Join(got, "\n") != strings.Join(tt.want, "\n") {
				t.Errorf("got:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(tt.want, "\n"))
			}
		})
	}
}

func TestParseErrorCap(t *testing.T) {
	src := strings.Repeat("kik\n", 30)
	errs := Validate(src, DefaultLimits())
	if len(errs) != 10 {
		t.Fatalf("got %d errors, want 10", len(errs))
	}
	if errs[0].Line != 1 || errs[9].Line != 10 {
		t.Errorf("errors not sorted by position: %v", errs)
	}
}
