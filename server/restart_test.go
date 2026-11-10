package server

import "testing"

func TestRestartMidMatch(t *testing.T) {
	dir := t.TempDir()
	s1, err := New(Options{DataDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	id, ta, _ := create(t, s1, "alice", "wait 1")
	tb, _ := join(t, s1, id, "bob", "wait 1")
	submit(t, s1, id, ta, "wait 4")
	before := do(t, s1, "GET", "/matches/"+id, "").Body.String()

	// "Restart": a fresh server over the same data directory.
	s2, err := New(Options{DataDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if after := do(t, s2, "GET", "/matches/"+id, "").Body.String(); after != before {
		t.Fatalf("state changed across restart:\n%s\n%s", before, after)
	}
	v := submit(t, s2, id, tb, "wait 1")
	if len(v.Rounds) != 2 {
		t.Fatalf("rounds %d", len(v.Rounds))
	}
	if l := roundLog(t, s2, id, 2); l.Players[1].Name != "alice" || l.Players[1].Script != "wait 4" {
		t.Errorf("round 2 used %+v, want seat a's pre-restart submission", l.Players[1])
	}
	if l := roundLog(t, s2, id, 1); l.Players[0].Script != "wait 1" {
		t.Errorf("round 1 log lost across restart")
	}
}

func TestFullMatchOverHTTP(t *testing.T) {
	s := newTestServer(t)
	id, ta, v := create(t, s, "alice", idleScript)
	tb, v := join(t, s, id, "bob", strongScript)
	// Round 1: b wins. Alice revises to the strong script; bob keeps his.
	if v.Rounds[0].Winner == nil || *v.Rounds[0].Winner != "b" {
		t.Fatalf("round 1 %+v", v.Rounds[0])
	}
	submit(t, s, id, ta, strongScript)
	v = submit(t, s, id, tb, strongScript)
	// Round 2: mirror match draws (same script both sides).
	if v.Rounds[1].Winner != nil || v.Status != StatusAwaiting {
		t.Fatalf("round 2 %+v status %s", v.Rounds[1], v.Status)
	}
	// Round 3: alice goes idle again, bob wins his second round.
	submit(t, s, id, ta, idleScript)
	v = submit(t, s, id, tb, strongScript)
	if v.Status != StatusFinished || v.Winner == nil || *v.Winner != "b" || len(v.Rounds) != 3 || v.Seats.B.Wins != 2 || v.Seats.A.Wins != 0 {
		t.Fatalf("final %+v", v)
	}
	if got := getMatch(t, s, id); got.Status != StatusFinished || len(got.Rounds) != 3 {
		t.Errorf("GET after finish %+v", got)
	}
	for n := 1; n <= 3; n++ {
		roundLog(t, s, id, n)
	}
}
