package server

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestHostedMatchModel(t *testing.T) {
	m := newHostedMatch("0123456789abcdef", "cfg")
	if m.Status != StatusPlayers || m.A != nil || m.B != nil || m.freeSeat() != "a" {
		t.Fatalf("new hosted match %+v", m)
	}
	if seat, err := m.join("alice", strp("A1"), hashToken("ta")); err != nil || seat != "a" || m.Status != StatusWaiting || m.readyToPlay() {
		t.Fatalf("first join: seat %q err %v status %s", seat, err, m.Status)
	}
	if seat, err := m.join("bob", strp("B1"), hashToken("tb")); err != nil || seat != "b" || m.Status != StatusAwaiting || !m.readyToPlay() {
		t.Fatalf("second join: seat %q err %v status %s", seat, err, m.Status)
	}
	if _, err := m.join("carol", strp("C"), hashToken("tc")); err != errSeatTaken {
		t.Fatalf("third join: %v", err)
	}
	// A player-created match still joins as seat b.
	p := newMatch("fedcba9876543210", "cfg", "alice", "A1", hashToken("ta"))
	if seat, _ := p.join("bob", strp("B1"), hashToken("tb")); seat != "b" {
		t.Fatalf("player-created match joined as %q", seat)
	}
}

func TestHostAMatch(t *testing.T) {
	s := newTestServer(t)
	rec := do(t, s, "POST", "/matches", `{}`)
	if rec.Code != 201 {
		t.Fatalf("host: %d %s", rec.Code, rec.Body.String())
	}
	var raw map[string]json.RawMessage
	decodeJSON(t, rec, &raw)
	if _, ok := raw["seat"]; ok {
		t.Errorf("hosted create returned a seat: %s", rec.Body.String())
	}
	if _, ok := raw["token"]; ok {
		t.Errorf("hosted create returned a token: %s", rec.Body.String())
	}
	var r hostedResp
	decodeJSON(t, rec, &r)
	if len(r.MatchID) != 16 || r.Match.Status != StatusPlayers || r.Match.Seats.A != nil || r.Match.Seats.B != nil {
		t.Fatalf("hosted create %+v", r)
	}
	// The state is public, like any match.
	var v View
	decodeJSON(t, do(t, s, "GET", "/matches/"+r.MatchID, ""), &v)
	if v.Status != StatusPlayers || v.NextRound != 1 {
		t.Errorf("GET hosted match %+v", v)
	}
}

func TestPlayersJoinHostedMatch(t *testing.T) {
	s := newTestServer(t)
	var host hostedResp
	decodeJSON(t, do(t, s, "POST", "/matches", `{}`), &host)
	id := host.MatchID

	joinAs := func(name, script string) seatResp {
		t.Helper()
		rec := do(t, s, "POST", "/matches/"+id+"/join", seatBody(name, script))
		if rec.Code != 201 {
			t.Fatalf("join %s: %d %s", name, rec.Code, rec.Body.String())
		}
		var r seatResp
		decodeJSON(t, rec, &r)
		return r
	}
	a := joinAs("alice", strongScript)
	if a.Seat != "a" || len(a.Token) != 32 || a.Match.Status != StatusWaiting || len(a.Match.Rounds) != 0 {
		t.Fatalf("first join %+v", a)
	}
	b := joinAs("bob", idleScript)
	if b.Seat != "b" || len(b.Match.Rounds) != 1 || b.Match.Rounds[0].P1 != "a" || b.Match.Status != StatusAwaiting {
		t.Fatalf("second join %+v", b)
	}
	// Both seats play on with their own tokens.
	submit(t, s, id, a.Token, strongScript)
	if v := submit(t, s, id, b.Token, idleScript); len(v.Rounds) != 2 || v.Rounds[1].P1 != "b" {
		t.Fatalf("round 2 %+v", v.Rounds)
	}
	// A third player is turned away.
	expectError(t, do(t, s, "POST", "/matches/"+id+"/join", seatBody("carol", idleScript)), 409, "conflict")
}

func TestHalfCreateRequest(t *testing.T) {
	s := newTestServer(t)
	for _, body := range []string{`{"name": "alice"}`, `{"script": "kick"}`} {
		e := expectError(t, do(t, s, "POST", "/matches", body), 400, "bad_request")
		if !strings.Contains(e.Error.Message, "neither to host") {
			t.Errorf("%s: message %q", body, e.Error.Message)
		}
	}
	if entries, _ := os.ReadDir(s.store.root); len(entries) != 0 {
		t.Errorf("%d matches created by bad requests", len(entries))
	}
}

func TestJoinValidatesAsTheAssignedSeat(t *testing.T) {
	s := newTestServer(t)
	var host hostedResp
	decodeJSON(t, do(t, s, "POST", "/matches", `{}`), &host)
	e := expectError(t, do(t, s, "POST", "/matches/"+host.MatchID+"/join", seatBody("alice", "kik")), 422, "invalid_script")
	if len(e.Errors) == 0 {
		t.Fatalf("no validation details: %+v", e)
	}
	var v View
	decodeJSON(t, do(t, s, "GET", "/matches/"+host.MatchID, ""), &v)
	if v.Seats.A != nil || v.Status != StatusPlayers {
		t.Errorf("a failed join changed the match: %+v", v)
	}
}

func TestJoinWithoutScript(t *testing.T) {
	s := newTestServer(t)
	var host hostedResp
	decodeJSON(t, do(t, s, "POST", "/matches", `{}`), &host)
	id := host.MatchID
	joinName := func(name string) seatResp {
		t.Helper()
		rec := do(t, s, "POST", "/matches/"+id+"/join", `{"name": `+jsonStr(name)+`}`)
		if rec.Code != 201 {
			t.Fatalf("join %s: %d %s", name, rec.Code, rec.Body.String())
		}
		var r seatResp
		decodeJSON(t, rec, &r)
		return r
	}
	a := joinName("alice")
	b := joinName("bob")
	if a.Seat != "a" || b.Seat != "b" || b.Match.Status != StatusAwaiting || len(b.Match.Rounds) != 0 {
		t.Fatalf("name-only joins: a=%s b=%s status=%s rounds=%d", a.Seat, b.Seat, b.Match.Status, len(b.Match.Rounds))
	}
	if b.Match.Seats.A.Submitted || b.Match.Seats.B.Submitted {
		t.Fatalf("nothing should be submitted yet: %+v %+v", b.Match.Seats.A, b.Match.Seats.B)
	}
	// Round 1 runs only on the second submission.
	if v := submit(t, s, id, a.Token, strongScript); len(v.Rounds) != 0 || !v.Seats.A.Submitted {
		t.Fatalf("after one submission: %+v", v)
	}
	if v := submit(t, s, id, b.Token, idleScript); len(v.Rounds) != 1 || v.Rounds[0].P1 != "a" {
		t.Fatalf("after both submissions: rounds %+v", v.Rounds)
	}
	// A name is still required.
	expectError(t, do(t, s, "POST", "/matches/"+id+"/join", `{}`), 409, "conflict") // full now
	var h2 hostedResp
	decodeJSON(t, do(t, s, "POST", "/matches", `{}`), &h2)
	expectError(t, do(t, s, "POST", "/matches/"+h2.MatchID+"/join", `{}`), 400, "bad_request")
}

func TestJoinWithScriptStillCountsAsRoundOne(t *testing.T) {
	s := newTestServer(t)
	id, _, _ := create(t, s, "alice", strongScript)
	if _, v := join(t, s, id, "bob", idleScript); len(v.Rounds) != 1 {
		t.Fatalf("join with a script should play round 1: %+v", v.Rounds)
	}
}
