package server

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"promptfighter/sim"
)

const (
	strongScript = "loop { move_forward x3 kick }" // beats an idle script from either side
	idleScript   = ""
)

func seatBody(name, script string) string {
	return `{"name": ` + jsonStr(name) + `, "script": ` + jsonStr(script) + `}`
}

func create(t *testing.T, s *Server, name, script string) (id, token string, v View) {
	t.Helper()
	rec := do(t, s, "POST", "/matches", seatBody(name, script))
	if rec.Code != 201 {
		t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
	}
	var r seatResp
	decodeJSON(t, rec, &r)
	return r.MatchID, r.Token, r.Match
}

func join(t *testing.T, s *Server, id, name, script string) (token string, v View) {
	t.Helper()
	rec := do(t, s, "POST", "/matches/"+id+"/join", seatBody(name, script))
	if rec.Code != 201 {
		t.Fatalf("join: %d %s", rec.Code, rec.Body.String())
	}
	var r seatResp
	decodeJSON(t, rec, &r)
	if r.Seat != "b" || r.MatchID != id {
		t.Fatalf("join resp %+v", r)
	}
	return r.Token, r.Match
}

func submit(t *testing.T, s *Server, id, token, script string) View {
	t.Helper()
	rec := do(t, s, "POST", "/matches/"+id+"/scripts", `{"script": `+jsonStr(script)+`}`, "Authorization", "Bearer "+token)
	if rec.Code != 200 {
		t.Fatalf("submit: %d %s", rec.Code, rec.Body.String())
	}
	var r struct{ Match View }
	decodeJSON(t, rec, &r)
	return r.Match
}

func getMatch(t *testing.T, s *Server, id string) View {
	t.Helper()
	rec := do(t, s, "GET", "/matches/"+id, "")
	if rec.Code != 200 {
		t.Fatalf("get: %d %s", rec.Code, rec.Body.String())
	}
	var v View
	decodeJSON(t, rec, &v)
	return v
}

func roundLog(t *testing.T, s *Server, id string, n int) *sim.Log {
	t.Helper()
	rec := do(t, s, "GET", "/matches/"+id+"/rounds/"+strconv.Itoa(n), "")
	if rec.Code != 200 {
		t.Fatalf("round %d: %d %s", n, rec.Code, rec.Body.String())
	}
	var l sim.Log
	decodeJSON(t, rec, &l)
	return &l
}

func TestMatchesCreate(t *testing.T) {
	s := newTestServer(t)
	id, token, v := create(t, s, "alice", strongScript)
	if !regexp.MustCompile(`^[0-9a-f]{16}$`).MatchString(id) || !regexp.MustCompile(`^[0-9a-f]{32}$`).MatchString(token) {
		t.Errorf("id %q token %q", id, token)
	}
	if v.Status != StatusWaiting || v.Seats.B != nil || v.Seats.A.Name != "alice" || !v.Seats.A.Submitted || v.NextRound != 1 || v.ConfigVersion != "2026.11-3" {
		t.Errorf("view %+v", v)
	}
}

func TestMatchesJoinRunsRoundOne(t *testing.T) {
	s := newTestServer(t)
	id, _, _ := create(t, s, "alice", strongScript)
	_, v := join(t, s, id, "bob", idleScript)
	if len(v.Rounds) != 1 || v.Rounds[0].P1 != "a" || v.Rounds[0].Winner == nil || *v.Rounds[0].Winner != "a" || v.Status != StatusAwaiting {
		t.Fatalf("view %+v", v)
	}
	if v.Seats.A.Submitted || v.Seats.B.Submitted || v.NextRound != 2 {
		t.Errorf("submissions should clear after round 1: %+v", v.Seats)
	}
}

func TestMatchesSeatTaken(t *testing.T) {
	s := newTestServer(t)
	id, _, _ := create(t, s, "alice", strongScript)
	join(t, s, id, "bob", idleScript)
	expectError(t, do(t, s, "POST", "/matches/"+id+"/join", seatBody("carol", idleScript)), 409, "conflict")
}

func TestMatchesInvalidScriptOnCreate(t *testing.T) {
	s := newTestServer(t)
	b := expectError(t, do(t, s, "POST", "/matches", seatBody("alice", "kik")), 422, "invalid_script")
	if len(b.Errors) != 1 || b.Errors[0].Player != "a" {
		t.Errorf("errors %+v", b.Errors)
	}
	entries, _ := os.ReadDir(filepath.Join(s.opts.DataDir, "matches"))
	if len(entries) != 0 {
		t.Errorf("a match was created: %v", entries)
	}
	expectError(t, do(t, s, "POST", "/matches", seatBody("", "kick")), 400, "bad_request")
}

func TestMatchesTokenRequired(t *testing.T) {
	s := newTestServer(t)
	id, _, _ := create(t, s, "alice", strongScript)
	expectError(t, do(t, s, "POST", "/matches/"+id+"/scripts", `{"script": "kick"}`), 401, "unauthorized")
	expectError(t, do(t, s, "POST", "/matches/"+id+"/scripts", `{"script": "kick"}`, "Authorization", "Bearer 00000000000000000000000000000000"), 401, "unauthorized")
}

func TestMatchesTokenNotStoredInPlainText(t *testing.T) {
	s := newTestServer(t)
	id, ta, _ := create(t, s, "alice", strongScript)
	tb, _ := join(t, s, id, "bob", idleScript)
	filepath.WalkDir(s.opts.DataDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, _ := os.ReadFile(p)
		if bytes.Contains(b, []byte(ta)) || bytes.Contains(b, []byte(tb)) {
			t.Errorf("%s contains a seat token", p)
		}
		return nil
	})
}

func TestMatchesReviseBetweenRounds(t *testing.T) {
	s := newTestServer(t)
	id, ta, _ := create(t, s, "alice", "wait 1")
	tb, _ := join(t, s, id, "bob", "wait 2")
	submit(t, s, id, ta, "wait 3") // revised
	v := submit(t, s, id, tb, "wait 2")
	if len(v.Rounds) != 2 {
		t.Fatalf("rounds %d", len(v.Rounds))
	}
	l := roundLog(t, s, id, 2) // round 2: b is p1
	if l.Players[0].Script != "wait 2" || l.Players[0].Name != "bob" || l.Players[1].Script != "wait 3" || l.Players[1].Name != "alice" {
		t.Errorf("round 2 players %+v", l.Players)
	}
}

func TestMatchesReplaceBeforeRound(t *testing.T) {
	s := newTestServer(t)
	id, ta, _ := create(t, s, "alice", "wait 1")
	tb, _ := join(t, s, id, "bob", "wait 1")
	submit(t, s, id, ta, "wait 5")
	v := submit(t, s, id, ta, "wait 6")
	if len(v.Rounds) != 1 || !v.Seats.A.Submitted {
		t.Fatalf("view %+v", v)
	}
	submit(t, s, id, tb, "wait 1")
	if l := roundLog(t, s, id, 2); l.Players[1].Script != "wait 6" {
		t.Errorf("seat a played %q, want the latest submission", l.Players[1].Script)
	}
}

func TestMatchesSubmitAfterFinish(t *testing.T) {
	s := newTestServer(t)
	id, ta, _ := create(t, s, "alice", strongScript)
	tb, _ := join(t, s, id, "bob", idleScript)
	submit(t, s, id, ta, strongScript)
	v := submit(t, s, id, tb, idleScript)
	if v.Status != StatusFinished || v.Winner == nil || *v.Winner != "a" || len(v.Rounds) != 2 {
		t.Fatalf("view %+v", v)
	}
	expectError(t, do(t, s, "POST", "/matches/"+id+"/scripts", `{"script": "kick"}`, "Authorization", "Bearer "+ta), 409, "conflict")
}

func TestMatchesInvalidSubmissionKeepsEarlier(t *testing.T) {
	s := newTestServer(t)
	id, ta, _ := create(t, s, "alice", "wait 1")
	tb, _ := join(t, s, id, "bob", "wait 1")
	submit(t, s, id, ta, "wait 7")
	expectError(t, do(t, s, "POST", "/matches/"+id+"/scripts", `{"script": "kik"}`, "Authorization", "Bearer "+ta), 422, "invalid_script")
	submit(t, s, id, tb, "wait 1")
	if l := roundLog(t, s, id, 2); l.Players[1].Script != "wait 7" {
		t.Errorf("seat a played %q", l.Players[1].Script)
	}
}

func TestMatchesPendingScriptHidden(t *testing.T) {
	s := newTestServer(t)
	id, ta, _ := create(t, s, "alice", "wait 1")
	join(t, s, id, "bob", "wait 1")
	submit(t, s, id, ta, "# SECRET_PLAN\nwait 9")
	rec := do(t, s, "GET", "/matches/"+id, "")
	if strings.Contains(rec.Body.String(), "SECRET_PLAN") {
		t.Errorf("pending script leaked: %s", rec.Body.String())
	}
	v := getMatch(t, s, id)
	if !v.Seats.A.Submitted || v.Seats.B.Submitted {
		t.Errorf("seats %+v %+v", v.Seats.A, v.Seats.B)
	}
}

func TestMatchesUnknown(t *testing.T) {
	s := newTestServer(t)
	expectError(t, do(t, s, "GET", "/matches/0000000000000000", ""), 404, "not_found")
	expectError(t, do(t, s, "GET", "/matches/..%2f..%2fetc", ""), 404, "not_found")
	expectError(t, do(t, s, "POST", "/matches/0000000000000000/join", seatBody("bob", "kick")), 404, "not_found")
}

func TestMatchesRoundLogEqualsEngine(t *testing.T) {
	s := newTestServer(t)
	id, _, _ := create(t, s, "alice", strongScript)
	join(t, s, id, "bob", idleScript)
	rec := do(t, s, "GET", "/matches/"+id+"/rounds/1", "")
	l, _ := sim.Simulate(sim.DefaultConfig(), sim.Entrant{Name: "alice", Source: strongScript}, sim.Entrant{Name: "bob", Source: idleScript})
	if rec.Code != 200 || !bytes.Equal(rec.Body.Bytes(), l.JSON()) {
		t.Errorf("round 1 log differs from the engine (%d)", rec.Code)
	}
}

func TestMatchesUnplayedRound(t *testing.T) {
	s := newTestServer(t)
	id, _, _ := create(t, s, "alice", strongScript)
	join(t, s, id, "bob", idleScript)
	for _, n := range []string{"3", "2", "0", "x"} {
		expectError(t, do(t, s, "GET", "/matches/"+id+"/rounds/"+n, ""), 404, "not_found")
	}
}

func TestMatchesConcurrentFinalSubmissions(t *testing.T) {
	s := newTestServer(t)
	for i := 0; i < 10; i++ {
		id, ta, _ := create(t, s, "alice", "wait 1")
		tb, _ := join(t, s, id, "bob", "wait 1")
		var wg sync.WaitGroup
		for _, tok := range []string{ta, tb} {
			wg.Add(1)
			go func(tok string) {
				defer wg.Done()
				rec := do(t, s, "POST", "/matches/"+id+"/scripts", `{"script": "wait 2"}`, "Authorization", "Bearer "+tok)
				if rec.Code != 200 {
					t.Errorf("submit %d", rec.Code)
				}
			}(tok)
		}
		wg.Wait()
		if v := getMatch(t, s, id); len(v.Rounds) != 2 || v.Rounds[1].N != 2 {
			t.Fatalf("iteration %d: rounds %+v", i, v.Rounds)
		}
	}
}

func TestMatchesBusyDefersRound(t *testing.T) {
	s := newTestServer(t, func(o *Options) { o.MaxSims, o.SimWait = 1, 20*time.Millisecond })
	id, ta, _ := create(t, s, "alice", "wait 1")
	s.sims <- struct{}{} // simulator saturated
	_, v := join(t, s, id, "bob", "wait 1")
	if len(v.Rounds) != 0 || !v.Seats.A.Submitted || !v.Seats.B.Submitted {
		t.Fatalf("round should stay pending while busy: %+v", v)
	}
	s.release()
	v = submit(t, s, id, ta, "wait 1") // the next request finds the match ready
	if len(v.Rounds) != 1 {
		t.Errorf("rounds %d, want the deferred round to run", len(v.Rounds))
	}
}
