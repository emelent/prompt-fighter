package sim

import (
	"fmt"
	"strings"

	"promptfighter/sim/dsl"
)

// Entrant is one side's script.
type Entrant struct {
	Name   string
	Source string
}

// ScriptError reports validation errors for one side.
type ScriptError struct {
	Player string // "p1" or "p2"
	Name   string
	Errors []dsl.Error
}

func (e *ScriptError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s (%s) is invalid:", e.Player, e.Name)
	for _, er := range e.Errors {
		b.WriteString("\n  " + er.String())
	}
	return b.String()
}

type world struct {
	cfg    Config
	t      int
	f      [2]*fighter
	snap   [2]view
	events [2][]Event // per-fighter buffer for the current tick
	log    *Log
}

func (w *world) emit(i int, e Event) {
	e.Frame = w.t
	w.events[i] = append(w.events[i], e)
}

func (w *world) flushEvents() {
	for i := range w.events {
		w.log.Events = append(w.log.Events, w.events[i]...)
		w.events[i] = w.events[i][:0]
	}
}

// Simulate runs one round between a (p1) and b (p2). It is a pure function
// of its inputs: the same inputs always produce a byte-identical log.
func Simulate(cfg Config, a, b Entrant) (*Log, error) {
	var scripts [2]*dsl.Script
	for i, e := range []Entrant{a, b} {
		s, errs := dsl.Parse(e.Source, cfg.Limits)
		if len(errs) > 0 {
			return nil, &ScriptError{Player: fmt.Sprintf("p%d", i+1), Name: e.Name, Errors: errs}
		}
		scripts[i] = s
	}
	budget := cfg.FrameBudget
	if budget < 1 {
		return nil, fmt.Errorf("frame budget must be >= 1, got %d", budget)
	}
	w := &world{cfg: cfg}
	w.log = &Log{
		FormatVersion: FormatVersion,
		ConfigVersion: cfg.Version,
		FrameBudget:   budget,
		Players: [2]Player{
			{ID: "p1", Name: a.Name, Script: a.Source},
			{ID: "p2", Name: b.Name, Script: b.Source},
		},
		Frames: make([]Frame, 0, budget),
		Events: []Event{},
	}
	for i := range w.f {
		w.f[i] = &fighter{
			w: w, idx: i, id: fmt.Sprintf("p%d", i+1),
			x: cfg.StartX[i], hp: cfg.MaxHP,
			stunUntil: -1, defBSUntil: -1,
			lastBlockT: -1, attackedT: -1, movedAwayT: -1,
			in: NewInterp(scripts[i]),
		}
	}
	w.f[0].facing, w.f[1].facing = sign(w.f[1].x-w.f[0].x), sign(w.f[0].x-w.f[1].x)

	ended := false
	for t := 0; t < budget && !ended; t++ {
		ended = w.tick(t)
	}
	w.finish()
	return w.log, nil
}

// tick simulates one frame and reports whether the round ended (KO).
func (w *world) tick(t int) bool {
	w.t = t
	c := &w.cfg
	// 1. Snapshot the start-of-frame state for sensors.
	for i, f := range w.f {
		w.snap[i] = f.view(t - 1)
	}
	// 2. Stance bookkeeping that depends only on time, and the block-streak lapse.
	for _, f := range w.f {
		f.preThink(t)
		f.stunnedAtTop = f.stunned(t)
		if f.blockStreak > 0 && t-f.lastBlockT >= c.GuardStreakWindow {
			f.blockStreak = 0
		}
	}
	// 3. Think: each actionable, unoccupied fighter steps its interpreter.
	for _, f := range w.f {
		if f.stunned(t) {
			continue
		}
		f.combo = 0
		if f.act == nil {
			f.in.Step(t, f)
		}
	}
	// 4. Move.
	w.move(t)
	// 5. Connect (computed from the same post-move state for both).
	conns := [2]*conn{w.f[0].connect(t), w.f[1].connect(t)}
	// 6. Resolve simultaneously.
	w.resolve(t, conns)
	// 6b. Passivity: a frame without starting an attack, not stunned at any
	// point of it, counts once, or twice when the fighter moved away.
	for _, f := range w.f {
		if f.stunnedAtTop || f.stunned(t) || f.attackedT == t {
			continue
		}
		inc := 1
		if f.movedAwayT == t {
			inc = 2
		}
		f.passive = min(f.passive+inc, 100000)
	}
	// 7. Whiffs at the end of active frames.
	for _, f := range w.f {
		if a := f.act; a != nil && a.kind == akAttack && !a.connected && a.k(t) == a.spec.Startup+a.spec.Active {
			f.whiff(a)
		}
	}
	// 8. KO check.
	ko := false
	for _, f := range w.f {
		if f.hp == 0 {
			ko = true
			f.emit("ko", Detail{})
		}
	}
	// 9. Record the frame, then retire finished actions.
	w.log.Frames = append(w.log.Frames, Frame{F: t, P1: w.f[0].frame(t), P2: w.f[1].frame(t)})
	w.flushEvents()
	for _, f := range w.f {
		if a := f.act; a != nil && a.k(t) >= a.total(c) {
			f.act = nil
			if a.stmt {
				f.in.Complete()
			}
		}
	}
	return ko
}

func (f *fighter) whiff(a *act) {
	reason := a.reason
	if reason == "" {
		reason = "out_of_range"
	}
	f.emit("whiff", Detail{Attack: a.attack.String(), Reason: reason})
}

func (f *fighter) preThink(t int) {
	c := f.cfg()
	switch f.stance {
	case stJump:
		if f.stanceK(t) > c.airFrames()+c.JumpLanding {
			f.stance = stNone
		}
	case stDuck:
		// A low_jab in progress holds the duck until it finishes.
		for f.duckPhase(t) == phRecovery && f.act != nil && f.act.kind == akAttack && f.act.attack == dsl.AtkLowJab {
			f.duckHold++
		}
		if f.duckPhase(t) == "" {
			f.stance = stNone
		}
	}
	// Air attacks end on landing, except landing-low attacks, which carry
	// through touchdown and finish on the ground.
	if a := f.act; a != nil && a.kind == akAttack && isAirAttack(a.attack) && a.zone != ZoneLandingLow && !f.airborne(t) {
		if a.reachedActive && !a.connected && a.k(t-1) < a.spec.Startup+a.spec.Active {
			f.whiff(a)
		}
		f.act = nil
		f.in.Complete()
	}
}

func (w *world) clampX(x int) int {
	lo, hi := w.cfg.HalfWidth, w.cfg.StageWidth-w.cfg.HalfWidth
	return max(lo, min(hi, x))
}

func (w *world) move(t int) {
	c := &w.cfg
	var nx [2]int
	for i, f := range w.f {
		dx := 0
		if a := f.act; a != nil && a.kind == akMove {
			dx += a.dir * f.toEnemy() * c.WalkSpeed
		}
		if f.airborne(t) {
			dx += f.jumpDir * c.JumpSpeedX
		}
		nx[i] = w.clampX(f.x + dx)
	}
	if abs(nx[0]-nx[1]) < c.minSep() {
		for i, f := range w.f {
			other := w.f[1-i]
			if (nx[i]-f.x)*sign(other.x-f.x) > 0 {
				nx[i] = f.x
			}
		}
	}
	for i, f := range w.f {
		if (nx[i]-f.x)*sign(w.f[1-i].x-f.x) < 0 {
			f.movedAwayT = t
		}
	}
	for i, f := range w.f {
		f.x = nx[i]
		f.y = 0
		if f.airborne(t) {
			k := f.stanceK(t)
			if k <= c.JumpRise {
				f.y = k * c.JumpYPerFrame
			} else {
				f.y = (c.airFrames() - k) * c.JumpYPerFrame
			}
		}
	}
	for _, f := range w.f {
		if !f.airborne(t) {
			if d := sign(f.enemy().x - f.x); d != 0 {
				f.facing = d
			}
		}
	}
}

type connKind int

const (
	cHit connKind = iota
	cBlock
	cThrow
)

type conn struct {
	kind connKind
	a    *act
}

// connect decides whether f's active attack connects with the enemy this tick.
func (f *fighter) connect(t int) *conn {
	a := f.act
	if a == nil || a.kind != akAttack || a.connected || a.attackPhase(t) != phActive {
		return nil
	}
	a.reachedActive = true
	e := f.enemy()
	reason := ""
	isThrow := a.attack == dsl.AtkThrow
	air := isAirAttack(a.attack)
	switch {
	case abs(f.x-e.x) > a.spec.Range:
		reason = "out_of_range"
	case abs(f.x-e.x) > a.effRange:
		reason = "stale"
	case e.knockedDown(t):
		reason = "invulnerable"
	case !air && a.height == HeightHigh && e.ducking(t):
		reason = "ducked"
	case a.zone == ZoneAirToAir && !e.airborne(t):
		reason = "grounded"
	case (a.zone == ZoneRisingOverhead || a.zone == ZoneFallingOverhead || a.height == HeightLow || isThrow) && e.airborne(t):
		reason = "airborne"
	case isThrow && e.inHitstun(t):
		reason = "stunned"
	}
	if reason != "" {
		a.reason = reason
		return nil
	}
	switch {
	case isThrow:
		return &conn{kind: cThrow, a: a}
	case a.height != HeightNone && e.guarding(t, a.height):
		return &conn{kind: cBlock, a: a}
	}
	return &conn{kind: cHit, a: a}
}

type outcome struct {
	kind      connKind
	atk       dsl.Attack
	dmg       int
	combo     int
	boost     int
	knockdown bool
	kdReason  string
}

func (w *world) resolve(t int, conns [2]*conn) {
	c := &w.cfg
	// Attacks beat throws.
	for i := range conns {
		if conns[i] != nil && conns[i].kind == cThrow && conns[1-i] != nil && conns[1-i].kind == cHit {
			conns[i] = nil
		}
	}
	// Simultaneous throws tech.
	if conns[0] != nil && conns[1] != nil && conns[0].kind == cThrow && conns[1].kind == cThrow {
		var push [2]int
		for i, f := range w.f {
			conns[i].a.connected = true
			f.emit("throw", Detail{Attack: "throw", Outcome: "tech"})
			push[i] = c.BlockPushback
		}
		w.pushback(push)
		return
	}
	// Compute outcomes from the pre-resolution state.
	var outs [2]*outcome
	for i, cn := range conns {
		if cn == nil {
			continue
		}
		cn.a.connected = true
		target := w.f[1-i]
		o := &outcome{kind: cn.kind, atk: cn.a.attack, boost: c.boostPct(target.passive)}
		switch cn.kind {
		case cThrow:
			o.dmg, o.combo, o.knockdown, o.kdReason = max(1, cn.a.spec.Damage*o.boost/100), 1, true, "throw"
		case cHit:
			o.combo = 1
			if target.inHitstun(t) {
				o.combo = target.combo + 1
			}
			o.dmg = max(1, cn.a.spec.Damage*c.comboPct(o.combo)*o.boost/10000)
			switch {
			case target.airborne(t):
				o.knockdown, o.kdReason = true, "airborne"
			case o.combo >= c.ComboKnockdownHits:
				o.knockdown, o.kdReason = true, "combo"
			case cn.a.spec.Knockdown:
				o.knockdown, o.kdReason = true, cn.a.attack.String()
			}
		}
		outs[i] = o
	}
	// Apply.
	var push [2]int
	for i, o := range outs {
		if o == nil {
			continue
		}
		att, tgt := w.f[i], w.f[1-i]
		switch o.kind {
		case cBlock:
			if tgt.blockStreak >= c.GuardBreakBlocks-1 {
				// The guard breaks: no damage, no blockstun for the attacker,
				// who recovers normally and can punish the stunned defender.
				w.emit(i, Event{Type: "guard_break", Actor: att.id, Target: tgt.id, Detail: Detail{Attack: o.atk.String()}})
				tgt.applyGuardBreak(t)
				push[1-i] += c.BlockPushback
				continue
			}
			tgt.blockStreak++
			tgt.lastBlockT = t
			w.emit(i, Event{Type: "block", Actor: att.id, Target: tgt.id, Detail: Detail{Attack: o.atk.String()}})
			ba := tgt.act
			ba.hold = max(ba.hold, ba.k(t)+c.Blockstun-c.BlockStartup)
			tgt.defBSUntil = t + c.Blockstun
			push[1-i] += c.BlockPushback
			att.applyBlocked(t, max(conns[i].a.spec.Recovery, c.Blockstun))
		case cHit, cThrow:
			tgt.hp = max(0, tgt.hp-o.dmg)
			tgt.blockStreak = 0
			boost := 0
			if o.boost > 100 {
				boost = o.boost
				tgt.passive = 0 // the boost is consumed
			}
			if o.kind == cHit {
				ev := Event{Type: "hit", Actor: att.id, Target: tgt.id, Detail: Detail{Attack: o.atk.String(), Damage: intp(o.dmg), Combo: o.combo, HP: intp(tgt.hp), Boost: boost}}
				w.emit(i, ev)
				push[1-i] += c.HitPushback
			} else {
				w.emit(i, Event{Type: "throw", Actor: att.id, Target: tgt.id, Detail: Detail{Attack: "throw", Outcome: "success", Damage: intp(o.dmg), HP: intp(tgt.hp), Boost: boost}})
			}
			if o.knockdown {
				tgt.applyKnockdown(t)
				tgt.emit("knockdown", Detail{Attack: o.atk.String(), Reason: o.kdReason})
			} else {
				tgt.applyHitstun(t)
				tgt.combo = o.combo
			}
		}
	}
	if outs[0] != nil && outs[1] != nil && outs[0].kind == cHit && outs[1].kind == cHit {
		w.emit(0, Event{Type: "trade", Actor: "p1", Target: "p2", Detail: Detail{
			Attacks: []string{outs[0].atk.String(), outs[1].atk.String()},
			Damages: []int{outs[0].dmg, outs[1].dmg},
		}})
	}
	w.pushback(push)
}

// pushback moves each fighter away from the other by push[i] units at the
// same time; whatever a wall prevents is transferred to the other fighter.
func (w *world) pushback(push [2]int) {
	var nx, rem [2]int
	for i, f := range w.f {
		want := sign(f.x-w.f[1-i].x) * push[i]
		nx[i] = w.clampX(f.x + want)
		rem[i] = want - (nx[i] - f.x)
	}
	for i := range w.f {
		if rem[1-i] != 0 {
			nx[i] = w.clampX(nx[i] - rem[1-i])
		}
	}
	for i, f := range w.f {
		f.x = nx[i]
	}
}

func (w *world) finish() {
	l := w.log
	last := len(l.Frames) - 1
	r := Result{EndFrame: last, HP: [2]int{w.f[0].hp, w.f[1].hp}}
	p1, p2 := "p1", "p2"
	switch {
	case w.f[0].hp == 0 && w.f[1].hp == 0:
		r.Reason = "double_ko"
	case w.f[0].hp == 0:
		r.Reason, r.Winner = "ko", &p2
	case w.f[1].hp == 0:
		r.Reason, r.Winner = "ko", &p1
	default:
		r.Reason = "timeout"
		if w.f[0].hp > w.f[1].hp {
			r.Winner = &p1
		} else if w.f[1].hp > w.f[0].hp {
			r.Winner = &p2
		}
	}
	l.Result = r
}
