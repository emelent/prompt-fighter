package sim

import "promptfighter/sim/dsl"

type stanceKind int

const (
	stNone stanceKind = iota
	stJump
	stDuck
)

type stunKind int

const (
	stunNone stunKind = iota
	stunHit
	stunBlock // attacker whose attack was blocked
	stunKnockdown
	stunGuardBreak
)

type actKind int

const (
	akAttack actKind = iota
	akBlock
	akMove
	akWait
	akCheck
	akStance // the 1-frame cost of starting a stance
	akIdle   // zero-cost loop guard
)

// act is whatever currently occupies a fighter's interpreter.
type act struct {
	kind  actKind
	id    int
	start int
	line  int
	stmt  bool // completion advances the interpreter past its statement

	// attacks
	attack        dsl.Attack
	spec          AttackSpec
	zone          AirZoneKind // air attacks: fixed at start from the airborne frame
	connected     bool
	reachedActive bool
	effRange      int // range after any stale reduction, fixed at start
	reason        string

	// blocks: guarded height; attacks: effective hit height
	block  dsl.Action
	height Height
	hold   int

	// moves / waits / checks
	dir int // +1 forward, -1 back
	n   int
}

func (a *act) k(t int) int { return t - a.start + 1 }

func (a *act) total(c *Config) int {
	switch a.kind {
	case akAttack:
		return a.spec.Total()
	case akBlock:
		return c.BlockStartup + a.hold + c.BlockRecovery
	}
	return a.n
}

const (
	phStartup  = "startup"
	phActive   = "active"
	phRecovery = "recovery"
)

func (a *act) attackPhase(t int) string {
	k := a.k(t)
	switch {
	case k <= a.spec.Startup:
		return phStartup
	case k <= a.spec.Startup+a.spec.Active:
		return phActive
	}
	return phRecovery
}

func (a *act) blockPhase(t int, c *Config) string {
	k := a.k(t)
	switch {
	case k <= c.BlockStartup:
		return phStartup
	case k <= c.BlockStartup+a.hold:
		return phActive
	}
	return phRecovery
}

func isAirAttack(a dsl.Attack) bool { return a == dsl.AtkJumpKick || a == dsl.AtkDivePunch }

// view is the sensor-visible state of a fighter at the end of a tick.
type view struct {
	x, hp                                               int
	guard, passive, repeats                             int
	airborne, ducking, blocking, attacking, knockedDown bool
}

type fighter struct {
	w   *world
	idx int
	id  string

	x, y, facing, hp int

	stance   stanceKind
	stStart  int
	stID     int
	jumpDir  int // absolute x direction of jump_forward/jump_back, 0 for jump
	duckHold int
	airUsed  bool

	act *act

	stun       stunKind
	stunUntil  int // stunned (no input) through this tick
	stunID     int
	defBSUntil int // defender blockstun display/guard through this tick

	combo int
	in    *Interp

	// Anti-passivity bookkeeping (see Guard break, Passivity boost and Stale
	// attacks in docs/RULES.md).
	blockStreak  int        // attacks guarded in a row
	lastBlockT   int        // tick of the last guarded attack
	passive      int        // frames without starting an attack
	lastAtk      dsl.Attack // the most recently started attack
	repeats      int        // starts of lastAtk in a row (0 before any attack)
	attackedT    int        // tick of the most recent attack start (-1: none)
	movedAwayT   int        // tick on which move() last carried f away from the enemy
	stunnedAtTop bool       // stunned at the start of the current tick

	nextID    int
	animKey   string
	animStart int
}

func (f *fighter) newID() int { f.nextID++; return f.nextID }

func (f *fighter) enemy() *fighter { return f.w.f[1-f.idx] }

func (f *fighter) cfg() *Config { return &f.w.cfg }

func sign(v int) int {
	switch {
	case v > 0:
		return 1
	case v < 0:
		return -1
	}
	return 0
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func (f *fighter) toEnemy() int {
	if d := sign(f.enemy().x - f.x); d != 0 {
		return d
	}
	return f.facing
}

// --- stance queries ---

func (f *fighter) stanceK(t int) int { return t - f.stStart + 1 }

func (f *fighter) airborne(t int) bool {
	if f.stance != stJump {
		return false
	}
	k := f.stanceK(t)
	return k >= 1 && k <= f.cfg().airFrames()
}

func (f *fighter) landing(t int) bool {
	return f.stance == stJump && f.stanceK(t) > f.cfg().airFrames()
}

func (f *fighter) duckPhase(t int) string {
	if f.stance != stDuck {
		return ""
	}
	c, k := f.cfg(), f.stanceK(t)
	switch {
	case k <= c.DuckStartup:
		return phStartup
	case k <= c.DuckStartup+f.duckHold:
		return phActive
	case k <= c.DuckStartup+f.duckHold+c.DuckRecovery:
		return phRecovery
	}
	return ""
}

func (f *fighter) ducking(t int) bool { return f.duckPhase(t) == phActive }

func (f *fighter) stunned(t int) bool { return t <= f.stunUntil }

func (f *fighter) knockedDown(t int) bool { return f.stun == stunKnockdown && f.stunned(t) }

func (f *fighter) inHitstun(t int) bool { return f.stun == stunHit && f.stunned(t) }

func (f *fighter) guarding(t int, h Height) bool {
	a := f.act
	return a != nil && a.kind == akBlock && a.height == h && a.blockPhase(t, f.cfg()) == phActive
}

func (f *fighter) view(t int) view {
	v := view{x: f.x, hp: f.hp, guard: f.blockStreak, passive: min(f.passive, 999), repeats: f.repeats}
	v.airborne = f.airborne(t)
	v.ducking = f.ducking(t)
	if a := f.act; a != nil {
		switch a.kind {
		case akBlock:
			v.blocking = a.blockPhase(t, f.cfg()) == phActive
		case akAttack:
			p := a.attackPhase(t)
			v.attacking = p == phStartup || p == phActive
		}
	}
	v.knockedDown = f.knockedDown(t)
	return v
}

// --- host implementation (called by the interpreter during Think) ---

func (f *fighter) sense(e *dsl.SensorExpr) int {
	me, en := f.w.snap[f.idx], f.w.snap[1-f.idx]
	dist := abs(me.x - en.x)
	switch e.Sensor {
	case dsl.SenDistance:
		return dist
	case dsl.SenHealth:
		return me.hp
	case dsl.SenEnemyHealth:
		return en.hp
	case dsl.SenEnemyAirborne:
		return b2i(en.airborne)
	case dsl.SenEnemyDucking:
		return b2i(en.ducking)
	case dsl.SenEnemyBlocking:
		return b2i(en.blocking)
	case dsl.SenEnemyAttacking:
		return b2i(en.attacking)
	case dsl.SenEnemyKnockedDown:
		return b2i(en.knockedDown)
	case dsl.SenGuard:
		return me.guard
	case dsl.SenPassive:
		return me.passive
	case dsl.SenEnemyPassive:
		return en.passive
	case dsl.SenRepeats:
		return me.repeats
	case dsl.SenInRange:
		return b2i(dist <= f.cfg().Attacks[e.Arg].Range)
	}
	return 0
}

func (f *fighter) startCheck(e *dsl.SensorExpr, v int) {
	cost := f.cfg().SensorCost
	f.begin(&act{kind: akCheck, n: cost, line: e.Pos.Line})
	val := intRaw(v)
	if e.Sensor.IsBool() {
		val = boolRaw(v != 0)
	}
	f.emit("check", Detail{Sensor: e.Name(), Value: val, Line: e.Pos.Line, Cost: cost})
}

func (f *fighter) startIdle(line int) {
	f.begin(&act{kind: akIdle, n: 1, line: line})
}

func (f *fighter) emit(typ string, d Detail) {
	f.w.emit(f.idx, Event{Type: typ, Actor: f.id, Detail: d})
}

func (f *fighter) begin(a *act) {
	a.id = f.newID()
	a.start = f.w.t
	f.act = a
}

func (f *fighter) startAttack(atk dsl.Attack, line int) {
	c := f.cfg()
	spec := c.Attacks[atk]
	a := &act{kind: akAttack, stmt: true, line: line, attack: atk, spec: spec, height: spec.Height}
	if isAirAttack(atk) {
		f.airUsed = true
		a.zone = c.airZone(f.stanceK(f.w.t))
		a.height = a.zone.height()
	}
	// Starting any attack ends passivity and the block streak, and counts
	// towards staleness when it repeats the previous attack.
	f.passive, f.blockStreak, f.attackedT = 0, 0, f.w.t
	if f.repeats > 0 && f.lastAtk == atk {
		f.repeats++
	} else {
		f.lastAtk, f.repeats = atk, 1
	}
	pct := c.staleRangePct(f.repeats)
	a.effRange = spec.Range * pct / 100
	f.begin(a)
	if pct < 100 {
		f.emit("stale", Detail{Attack: atk.String(), Repeats: f.repeats, Range: intp(a.effRange)})
	}
}

func (f *fighter) tryAction(st *dsl.ActionStmt) bool {
	t, c := f.w.t, f.cfg()
	line := st.Pos.Line
	inStance := f.stance != stNone
	air := f.airborne(t)
	n := st.N
	switch st.Action {
	case dsl.ActWait:
		f.begin(&act{kind: akWait, stmt: true, line: line, n: n})
		return true
	case dsl.ActPunch:
		switch {
		case air && !f.airUsed:
			f.startAttack(dsl.AtkDivePunch, line)
		case f.ducking(t):
			f.startAttack(dsl.AtkLowJab, line)
		case !inStance:
			f.startAttack(dsl.AtkPunch, line)
		default:
			return false
		}
		return true
	case dsl.ActKick:
		switch {
		case air && !f.airUsed:
			f.startAttack(dsl.AtkJumpKick, line)
		case !inStance:
			f.startAttack(dsl.AtkKick, line)
		default:
			return false
		}
		return true
	}
	if inStance {
		return false
	}
	switch st.Action {
	case dsl.ActSweep:
		f.startAttack(dsl.AtkSweep, line)
	case dsl.ActThrow:
		f.startAttack(dsl.AtkThrow, line)
	case dsl.ActJump, dsl.ActJumpForward, dsl.ActJumpBack:
		f.stance, f.stStart, f.stID, f.airUsed = stJump, t, f.newID(), false
		switch st.Action {
		case dsl.ActJumpForward:
			f.jumpDir = f.toEnemy()
		case dsl.ActJumpBack:
			f.jumpDir = -f.toEnemy()
		default:
			f.jumpDir = 0
		}
		f.begin(&act{kind: akStance, stmt: true, line: line, n: 1})
	case dsl.ActDuck:
		hold := c.DefaultHold
		if st.HasN {
			hold = n
		}
		f.stance, f.stStart, f.stID, f.duckHold = stDuck, t, f.newID(), hold
		f.begin(&act{kind: akStance, stmt: true, line: line, n: 1})
	case dsl.ActHighBlock, dsl.ActLowBlock:
		hold := c.DefaultHold
		if st.HasN {
			hold = n
		}
		h := HeightHigh
		if st.Action == dsl.ActLowBlock {
			h = HeightLow
		}
		f.begin(&act{kind: akBlock, stmt: true, line: line, block: st.Action, height: h, hold: hold})
	case dsl.ActMoveForward, dsl.ActMoveBack:
		cnt := 1
		if st.HasN {
			cnt = n
		}
		dir := 1
		if st.Action == dsl.ActMoveBack {
			dir = -1
		}
		f.begin(&act{kind: akMove, stmt: true, line: line, dir: dir, n: cnt})
	}
	return true
}

// --- interrupts ---

func (f *fighter) abandon() {
	f.act = nil
	if f.stance == stDuck || f.landing(f.w.t) {
		f.stance = stNone
	}
}

func (f *fighter) applyHitstun(t int) {
	f.abandon()
	f.stun, f.stunUntil, f.stunID = stunHit, t+f.cfg().Hitstun, f.newID()
	f.in.Interrupt("on_hit")
}

func (f *fighter) applyKnockdown(t int) {
	f.act = nil
	f.stance = stNone
	f.y = 0
	f.combo = 0
	f.stun, f.stunUntil, f.stunID = stunKnockdown, t+f.cfg().KnockdownFrames, f.newID()
	f.in.Interrupt("on_hit")
}

// applyGuardBreak stuns a fighter whose guard was broken: no input, no
// guarding, and on_hit when it recovers.
func (f *fighter) applyGuardBreak(t int) {
	f.abandon()
	f.blockStreak = 0
	f.defBSUntil = -1
	f.stun, f.stunUntil, f.stunID = stunGuardBreak, t+f.cfg().GuardBreakStun, f.newID()
	f.in.Interrupt("on_hit")
}

func (f *fighter) applyBlocked(t, frames int) {
	f.abandon()
	f.stun, f.stunUntil, f.stunID = stunBlock, t+frames, f.newID()
	f.in.Interrupt("on_block")
}

// --- per-frame display ---

func (f *fighter) frame(t int) FighterFrame {
	c := f.cfg()
	ff := FighterFrame{X: f.x, Y: f.y, Facing: f.facing, HP: f.hp, Combo: f.combo}
	key := 0
	a := f.act
	switch {
	case f.hp == 0:
		ff.State, ff.Anim = "ko", "ko"
	case f.stunned(t) && f.stun == stunKnockdown:
		ff.State, ff.Anim, key = "knockdown", "knockdown", f.stunID
		ff.Invulnerable = true
	case f.stunned(t) && f.stun == stunHit:
		ff.State, ff.Anim, key = "hitstun", "hitstun", f.stunID
	case f.stunned(t) && f.stun == stunGuardBreak:
		ff.State, ff.Anim, key = "guard_break", "guard_break", f.stunID
	case f.stunned(t) && f.stun == stunBlock:
		ff.State, ff.Anim, key = "blockstun", "blockstun", f.stunID
	case t <= f.defBSUntil && a != nil && a.kind == akBlock:
		ff.State, ff.Anim, ff.Phase, key = "blockstun", a.block.String(), phActive, a.id
	case f.airborne(t):
		ff.State = "airborne"
		if a != nil && a.kind == akAttack {
			ff.Anim, ff.Phase, key = a.attack.String(), a.attackPhase(t), a.id
		} else {
			ff.Anim, key = "jump", f.stID
		}
	case f.landing(t) && !(a != nil && a.kind == akAttack):
		ff.State, ff.Anim, key = "landing", "landing", f.stID
	case f.stance == stDuck:
		ff.State = "duck"
		if a != nil && a.kind == akAttack {
			ff.Anim, ff.Phase, key = a.attack.String(), a.attackPhase(t), a.id
		} else {
			ff.Anim, ff.Phase, key = "duck", f.duckPhase(t), f.stID
		}
	case a != nil && a.kind == akAttack:
		ff.State, ff.Anim, ff.Phase, key = "attack", a.attack.String(), a.attackPhase(t), a.id
	case a != nil && a.kind == akBlock:
		ff.State, ff.Anim, ff.Phase, key = "block", a.block.String(), a.blockPhase(t, c), a.id
	case a != nil && a.kind == akMove:
		ff.State, key = "walk", a.id
		ff.Anim = "walk_forward"
		if a.dir < 0 {
			ff.Anim = "walk_back"
		}
	case a != nil && a.kind == akCheck:
		ff.State, ff.Anim, key = "idle", "check", a.id
	case a != nil && a.kind == akWait:
		ff.State, ff.Anim, key = "idle", "wait", a.id
	default:
		ff.State, ff.Anim = "idle", "idle"
	}
	animKey := ff.Anim + ":" + itoa(key)
	if animKey != f.animKey {
		f.animKey, f.animStart = animKey, t
	}
	ff.AnimFrame = t - f.animStart
	if !f.stunned(t) && f.hp > 0 {
		ff.ScriptLine = f.in.Line
	}
	return ff
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	neg := v < 0
	if neg {
		v = -v
	}
	for v > 0 {
		i--
		b[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
