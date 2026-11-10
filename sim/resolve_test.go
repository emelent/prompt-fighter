package sim

import "testing"

func TestHitAppliesHitstunAndPushback(t *testing.T) {
	l := run(t, "punch", "", startX(30, 36), frames(20))
	h := events(l, "hit", "p1")
	if len(h) != 1 {
		t.Fatalf("hits = %v", h)
	}
	if got, want := evStr(h[0]), `f3 p1 hit {"attack":"punch","damage":5,"combo":1,"hp":95}`; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
	if h[0].Target != "p2" {
		t.Errorf("target %q", h[0].Target)
	}
	if d := p(l, 3, 1).X - p(l, 3, 0).X; d != 9 {
		t.Errorf("distance after hit = %d, want 9", d)
	}
	for f := 3; f <= 8; f++ {
		if p(l, f, 1).State != "hitstun" || p(l, f, 1).ScriptLine != 0 {
			t.Errorf("frame %d: %+v, want hitstun with no script input", f, p(l, f, 1))
		}
	}
	if p(l, 9, 1).State == "hitstun" {
		t.Errorf("hitstun lasts 5 frames after the hit frame")
	}
}

func TestHitPushbackTransfersAtWall(t *testing.T) {
	l := run(t, "", "punch", startX(3, 9), frames(10))
	if a, b := p(l, 3, 0).X, p(l, 3, 1).X; a != 2 || b != 11 {
		t.Errorf("x = %d,%d, want 2,11", a, b)
	}
}

func TestHitAntiAirKnockdown(t *testing.T) {
	l := run(t, "kick", "jump", startX(30, 40), frames(30))
	h := events(l, "hit", "p1")
	if len(h) != 1 || h[0].Frame != 6 {
		t.Fatalf("hits = %v", h)
	}
	ff := p(l, 6, 1)
	if ff.Y != 0 || ff.State != "knockdown" {
		t.Errorf("target %+v, want y=0 knockdown", ff)
	}
	if k := events(l, "knockdown", "p2"); len(k) != 1 || k[0].Detail.Reason != "airborne" {
		t.Errorf("knockdown events %v", k)
	}
}

func TestBlockHighBlockStopsKick(t *testing.T) {
	l := run(t, "kick", "high_block hold 30", startX(30, 40), frames(30))
	b := events(l, "block", "p1")
	if len(b) != 1 || evStr(b[0]) != `f6 p1 block {"attack":"kick"}` || b[0].Target != "p2" {
		t.Fatalf("blocks = %v", b)
	}
	if hp := p(l, 29, 1).HP; hp != 100 {
		t.Errorf("defender hp %d, want 100", hp)
	}
	for f := 6; f <= 9; f++ {
		if s := p(l, f, 1).State; s != "blockstun" {
			t.Errorf("defender frame %d state %q, want blockstun", f, s)
		}
	}
	if s := p(l, 10, 1).State; s != "block" {
		t.Errorf("defender keeps guarding after blockstun, got %q", s)
	}
	if d := p(l, 6, 1).X - p(l, 6, 0).X; d != 12 {
		t.Errorf("distance after block %d, want 12 (pushback 2)", d)
	}
	for f := 6; f <= 15; f++ {
		if s := p(l, f, 0).State; s != "blockstun" {
			t.Errorf("attacker frame %d state %q, want blockstun (kick recovery 9)", f, s)
		}
	}
	if s := p(l, 16, 0).State; s == "blockstun" {
		t.Errorf("attacker blockstun too long")
	}
}

func TestBlockExtendsHoldForBlockstun(t *testing.T) {
	// hold 6 guards frames 1-6 and the kick connects on frame 6, the last guard
	// frame; the 3 blockstun frames must extend the guard through frame 9.
	l := run(t, "kick", "high_block hold 6", startX(30, 40), frames(20))
	if len(events(l, "block", "p1")) != 1 {
		t.Fatalf("kick should be blocked")
	}
	for f := 7; f <= 9; f++ {
		if s := p(l, f, 1).State; s != "blockstun" {
			t.Errorf("frame %d state %q, want blockstun", f, s)
		}
	}
}

func TestBlockWrongHeight(t *testing.T) {
	l := run(t, "sweep", "high_block hold 30", startX(30, 40), frames(30))
	if h := events(l, "hit", "p1"); len(h) != 1 {
		t.Fatalf("sweep should hit through high_block, hits %v", h)
	}
	if s := p(l, 8, 1).State; s != "knockdown" {
		t.Errorf("state %q, want knockdown", s)
	}
	l = run(t, "kick", "low_block hold 30", startX(30, 40), frames(30))
	if h := events(l, "hit", "p1"); len(h) != 1 {
		t.Errorf("kick should hit through low_block")
	}
}

func TestBlockStartupDoesNotGuard(t *testing.T) {
	// Block startup is frame 5 when started on frame 5; the punch is active on 3-4.
	l := run(t, "punch", "wait 3 high_block", startX(30, 36), frames(10))
	if h := events(l, "hit", "p1"); len(h) != 1 {
		t.Errorf("block startup must not guard: hits %v", h)
	}
}

func TestThrowBeatsBlock(t *testing.T) {
	for _, blk := range []string{"high_block hold 30", "low_block hold 30"} {
		l := run(t, "throw", blk, startX(30, 34), frames(30))
		th := events(l, "throw", "p1")
		if len(th) != 1 || th[0].Detail.Outcome != "success" || *th[0].Detail.Damage != 12 {
			t.Fatalf("%s: throws = %v", blk, th)
		}
		if hp := p(l, 5, 1).HP; hp != 88 {
			t.Errorf("%s: hp %d, want 88", blk, hp)
		}
		if k := events(l, "knockdown", "p2"); len(k) != 1 {
			t.Errorf("%s: knockdown %v", blk, k)
		}
	}
}

func TestThrowAttackBeatsThrow(t *testing.T) {
	l := run(t, "throw", "punch", startX(30, 34), frames(30))
	if h := events(l, "hit", "p2"); len(h) != 1 || h[0].Frame != 3 {
		t.Fatalf("hits = %v", h)
	}
	if len(events(l, "throw", "")) != 0 {
		t.Errorf("throw must never become active: %v", events(l, "throw", ""))
	}
	if len(events(l, "whiff", "p1")) != 0 {
		t.Errorf("abandoned throw must not whiff")
	}
	if hp := p(l, 29, 1).HP; hp != 100 {
		t.Errorf("puncher hp %d", hp)
	}
}

func TestThrowAttackBeatsThrowSameFrame(t *testing.T) {
	// throw active on frame 5; punch started on frame 2 is active on 5-6.
	l := run(t, "throw", "wait 2 punch", startX(30, 34), frames(30))
	if len(events(l, "throw", "")) != 0 {
		t.Errorf("throw should not apply: %v", events(l, "throw", ""))
	}
	if h := events(l, "hit", "p2"); len(h) != 1 || h[0].Frame != 5 {
		t.Errorf("hits = %v, want the punch at frame 5", h)
	}
}

func TestThrowSimultaneousTech(t *testing.T) {
	l := run(t, "throw", "throw", startX(48, 53), frames(30))
	th := events(l, "throw", "")
	if len(th) != 2 || th[0].Detail.Outcome != "tech" || th[1].Detail.Outcome != "tech" || th[0].Actor != "p1" || th[1].Actor != "p2" {
		t.Fatalf("throws = %v", th)
	}
	if a, b := p(l, 29, 0).HP, p(l, 29, 1).HP; a != 100 || b != 100 {
		t.Errorf("hp %d,%d", a, b)
	}
	if d := p(l, 5, 1).X - p(l, 5, 0).X; d != 9 {
		t.Errorf("distance %d, want 9 (both pushed 2)", d)
	}
	if s := p(l, 6, 0).Phase; s != "recovery" {
		t.Errorf("teched throw continues into recovery, phase %q", s)
	}
}

func TestTradePunch(t *testing.T) {
	l := run(t, "punch", "punch", startX(30, 37), frames(20))
	if tr := events(l, "trade", ""); len(tr) != 1 || tr[0].Frame != 3 {
		t.Fatalf("trades = %v", tr)
	}
	if got := evStr(events(l, "trade", "")[0]); got != `f3 p1 trade {"attacks":["punch","punch"],"damages":[5,5]}` {
		t.Errorf("trade = %s", got)
	}
	if n := len(events(l, "hit", "")); n != 2 {
		t.Errorf("hit events %d, want 2", n)
	}
	if a, b := p(l, 3, 0).HP, p(l, 3, 1).HP; a != 95 || b != 95 {
		t.Errorf("hp %d,%d", a, b)
	}
}
