package sim

import (
	"fmt"
	"strings"
	"testing"

	"promptfighter/sim/dsl"
)

// stubHost is a minimal world for interpreter tests: every action occupies
// the interpreter for a fixed number of frames, sensors read from a function
// of the frame on which the read starts.
type stubHost struct {
	t      int
	cost   map[dsl.Action]int
	sensor func(t int, name string) int
	busy   int
	stmt   bool
	trace  []string
}

func (h *stubHost) sense(e *dsl.SensorExpr) int {
	if h.sensor == nil {
		return 0
	}
	return h.sensor(h.t, e.Name())
}

func (h *stubHost) startCheck(e *dsl.SensorExpr, v int) {
	h.busy, h.stmt = 2, false
	h.trace = append(h.trace, fmt.Sprintf("%d:check(%s=%d)", h.t, e.Name(), v))
}

func (h *stubHost) tryAction(st *dsl.ActionStmt) bool {
	n := 1
	if c, ok := h.cost[st.Action]; ok {
		n = c
	}
	if st.HasN && (st.Action == dsl.ActWait || st.Action == dsl.ActMoveForward || st.Action == dsl.ActMoveBack) {
		n = st.N
	}
	h.busy, h.stmt = n, true
	h.trace = append(h.trace, fmt.Sprintf("%d:%s", h.t, st.Action))
	return true
}

func (h *stubHost) startIdle(int) {
	h.busy, h.stmt = 1, false
	h.trace = append(h.trace, fmt.Sprintf("%d:idle", h.t))
}

func (h *stubHost) emit(typ string, d Detail) {
	s := fmt.Sprintf("%d:%s", h.t, typ)
	if d.Handler != "" {
		s += "(" + d.Handler + ")"
	}
	h.trace = append(h.trace, s)
}

// drive runs src for n frames against the stub host and returns the trace.
func drive(t *testing.T, src string, n int, h *stubHost) string {
	t.Helper()
	s, errs := dsl.Parse(src, dsl.DefaultLimits())
	if len(errs) > 0 {
		t.Fatalf("parse: %v", errs)
	}
	in := NewInterp(s)
	for h.t = 0; h.t < n; h.t++ {
		if h.busy == 0 {
			in.Step(h.t, h)
		}
		if h.busy > 0 {
			h.busy--
			if h.busy == 0 && h.stmt {
				in.Complete()
			}
		}
	}
	return strings.Join(h.trace, " ")
}

func TestInterpStatementCosts(t *testing.T) {
	cost := map[dsl.Action]int{dsl.ActPunch: 10, dsl.ActKick: 18}
	tests := []struct {
		name, src, want string
		n               int
	}{
		{"attack then wait", "punch wait 1", "0:punch 10:wait 11:script_end", 12},
		{"move count", "move_forward x3 kick", "0:move_forward 3:kick 21:script_end", 22},
		{"wait n", "wait 4 punch", "0:wait 4:punch 14:script_end", 15},
		{"repeat", "repeat 3 { punch }", "0:punch 10:punch 20:punch 30:script_end", 31},
		{"if bookkeeping is free", "if 2 > 1 { punch } kick", "0:punch 10:kick 28:script_end", 29},
		{"else branch", "if 1 > 2 { punch } else { kick } wait 1", "0:kick 18:wait 19:script_end", 20},
		{"elif chain without sensors", "if 1 > 2 { punch } elif 3 > 2 { kick } else { wait 1 }", "0:kick 18:script_end", 19},
		{"empty script ends at once", "", "0:script_end", 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := drive(t, tt.src, tt.n, &stubHost{cost: cost})
			if got != tt.want {
				t.Errorf("got  %s\nwant %s", got, tt.want)
			}
		})
	}
}

func TestInterpZeroCostLoopGuard(t *testing.T) {
	got := drive(t, "loop { if 1 > 2 { kick } }", 4, &stubHost{})
	if want := "0:idle 1:idle 2:idle 3:idle"; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
	// A loop whose body consumes frames gets no extra idle frame.
	got = drive(t, "loop { wait 2 }", 6, &stubHost{})
	if want := "0:wait 2:wait 4:wait"; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestInterpSensorCostAndShortCircuit(t *testing.T) {
	src := "if in_range(sweep) and not enemy.airborne { sweep } wait 1"
	vals := func(inRange int) func(int, string) int {
		return func(_ int, name string) int {
			if name == "in_range(sweep)" {
				return inRange
			}
			return 0
		}
	}
	cost := map[dsl.Action]int{dsl.ActSweep: 25}
	// true: two reads (4 frames) then sweep
	got := drive(t, src, 31, &stubHost{cost: cost, sensor: vals(1)})
	if want := "0:check(in_range(sweep)=1) 2:check(enemy.airborne=0) 4:sweep 29:wait 30:script_end"; got != want {
		t.Errorf("true path:\ngot  %s\nwant %s", got, want)
	}
	// false: short-circuit after one read (2 frames)
	got = drive(t, src, 4, &stubHost{cost: cost, sensor: vals(0)})
	if want := "0:check(in_range(sweep)=0) 2:wait 3:script_end"; got != want {
		t.Errorf("short-circuit:\ngot  %s\nwant %s", got, want)
	}
	// or short-circuits on true
	got = drive(t, "if enemy.attacking or enemy.ducking { wait 1 }", 4, &stubHost{sensor: func(int, string) int { return 1 }})
	if want := "0:check(enemy.attacking=1) 2:wait 3:script_end"; got != want {
		t.Errorf("or short-circuit:\ngot  %s\nwant %s", got, want)
	}
}

func TestInterpLiteralsAreFree(t *testing.T) {
	got := drive(t, "if health < 30 { wait 1 }", 4, &stubHost{sensor: func(int, string) int { return 10 }})
	if want := "0:check(health=10) 2:wait 3:script_end"; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestInterpSensorSnapshotOnFirstFrame(t *testing.T) {
	// The sensor value is the one at the start of the read; a change on the
	// read's second frame is not seen.
	sensor := func(t int, _ string) int {
		if t >= 1 {
			return 1
		}
		return 0
	}
	got := drive(t, "if enemy.airborne { kick } else { wait 1 }", 4, &stubHost{sensor: sensor})
	if want := "0:check(enemy.airborne=0) 2:wait 3:script_end"; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestInterpStaleSnapshotInSimulation(t *testing.T) {
	// p2 jumps on frame 1, so it is airborne from frame 1. p1's read starts on
	// frame 0, when p2 is still grounded, and must return false.
	l := run(t, "if enemy.airborne { kick } else { wait 1 }", "wait 1 jump", frames(5))
	checks := events(l, "check", "p1")
	if len(checks) != 1 || string(checks[0].Detail.Value) != "false" || checks[0].Frame != 0 {
		t.Fatalf("checks = %v", checks)
	}
	if p(l, 1, 1).State != "airborne" {
		t.Fatalf("p2 should be airborne on frame 1, got %+v", p(l, 1, 1))
	}
	if got := p(l, 2, 0).Anim; got != "wait" {
		t.Errorf("p1 should take the else branch on frame 2, anim %q", got)
	}
}

func TestInterpCheckEvents(t *testing.T) {
	l := run(t, "wait 1\nif distance > 10 { wait 1 }", "", frames(5))
	c := events(l, "check", "p1")
	if len(c) != 1 {
		t.Fatalf("checks = %v", c)
	}
	got := evStr(c[0])
	if want := `f1 p1 check {"sensor":"distance","value":40,"line":2,"cost":2}`; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}
