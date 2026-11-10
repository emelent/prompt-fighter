package mcpserver

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"promptfighter/server"
	"promptfighter/sim"
)

const (
	strong = "loop { move_forward x3 kick }"
	idle   = ""
)

func pfServe(t *testing.T) *httptest.Server {
	t.Helper()
	h, err := server.New(server.Options{DataDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(h)
	t.Cleanup(ts.Close)
	return ts
}

func TestMatchTwoClientsPlay(t *testing.T) {
	ts := pfServe(t)
	alice, bob := connect(t, Options{ServerURL: ts.URL}), connect(t, Options{ServerURL: ts.URL})

	var created SeatOut
	res := callTool(t, alice, "create_match", map[string]any{"name": "alice", "script": idle})
	structured(t, res, &created)
	if created.Seat != "a" || len(created.Token) != 32 || created.Match.Status != server.StatusWaiting {
		t.Fatalf("created %+v", created)
	}
	if txt := resultText(res); !strings.Contains(txt, created.Token) || !strings.Contains(txt, "share match_id "+created.MatchID) {
		t.Errorf("create text:\n%s", txt)
	}

	var joined SeatOut
	structured(t, callTool(t, bob, "join_match", map[string]any{"match_id": created.MatchID, "name": "bob", "script": strong}), &joined)
	if joined.Seat != "b" || len(joined.Match.Rounds) != 1 || *joined.Match.Rounds[0].Winner != "b" {
		t.Fatalf("joined %+v", joined.Match)
	}

	// Both clients see the same state.
	var ga, gb MatchOut
	structured(t, callTool(t, alice, "get_match", map[string]any{"match_id": created.MatchID}), &ga)
	structured(t, callTool(t, bob, "get_match", map[string]any{"match_id": created.MatchID}), &gb)
	ja, _ := json.Marshal(ga)
	jb, _ := json.Marshal(gb)
	if string(ja) != string(jb) || len(ga.Match.Rounds) != 1 {
		t.Errorf("states differ:\n%s\n%s", ja, jb)
	}

	// Round 2: alice keeps idling, bob keeps the strong script -> bob wins 2-0.
	res = callTool(t, alice, "submit_script", map[string]any{"match_id": created.MatchID, "token": created.Token, "script": idle})
	var sub MatchOut
	structured(t, res, &sub)
	if !strings.Contains(resultText(res), "waiting for your opponent to submit round 2") {
		t.Errorf("submit text:\n%s", resultText(res))
	}
	structured(t, callTool(t, bob, "submit_script", map[string]any{"match_id": created.MatchID, "token": joined.Token, "script": strong}), &sub)
	if sub.Match.Status != server.StatusFinished || sub.Match.Winner == nil || *sub.Match.Winner != "b" {
		t.Fatalf("final %+v", sub.Match)
	}
	if txt := resultText(callTool(t, alice, "get_match", map[string]any{"match_id": created.MatchID})); !strings.Contains(txt, "match finished: seat b (bob) won.") || !strings.Contains(txt, "round 2: seat b (bob) was p1") {
		t.Errorf("finished text:\n%s", txt)
	}

	// Replay of round 2: seat b played p1.
	var rp ReplayOut
	res = callTool(t, alice, "get_round_replay", map[string]any{"match_id": created.MatchID, "round": 2})
	structured(t, res, &rp)
	l, _ := sim.Simulate(sim.DefaultConfig(), sim.Entrant{Name: "bob", Source: strong}, sim.Entrant{Name: "alice", Source: idle})
	if rp.P1Seat != "b" || rp.Round != 2 || strings.Join(rp.Events, "\n") != strings.Join(l.EventLines(false), "\n") || rp.Players[0].Name != "bob" {
		t.Errorf("replay %+v", rp)
	}
	if txt := resultText(res); !strings.Contains(txt, "seat b (bob) played p1") {
		t.Errorf("replay text:\n%s", txt)
	}
	expectToolError(t, callTool(t, alice, "get_round_replay", map[string]any{"match_id": created.MatchID, "round": 3}), "round 3 has not been played")
}

func TestMatchErrors(t *testing.T) {
	ts := pfServe(t)
	cs := connect(t, Options{ServerURL: ts.URL})
	var created SeatOut
	structured(t, callTool(t, cs, "create_match", map[string]any{"name": "alice", "script": "kick"}), &created)
	expectToolError(t, callTool(t, cs, "submit_script", map[string]any{"match_id": created.MatchID, "token": "00000000000000000000000000000000", "script": "kick"}), "unauthorized")
	expectToolError(t, callTool(t, cs, "create_match", map[string]any{"name": "alice", "script": "kik"}), "invalid_script", `1:1: unknown action "kik" — did you mean "kick"?`)
	expectToolError(t, callTool(t, cs, "get_match", map[string]any{"match_id": "0000000000000000"}), "not_found")
	expectToolError(t, callTool(t, cs, "join_match", map[string]any{"match_id": created.MatchID, "name": "", "script": "kick"}), "bad_request")
	expectToolError(t, callTool(t, cs, "submit_script", map[string]any{"match_id": created.MatchID, "token": "", "script": "kick"}), "token is required")
}

func TestMatchToolsDisabled(t *testing.T) {
	cs := connect(t, Options{ServerURL: ""})
	for _, name := range []string{"create_match", "get_match"} {
		args := map[string]any{"match_id": "0123456789abcdef"}
		if name == "create_match" {
			args = map[string]any{"name": "a", "script": "kick"}
		}
		expectToolError(t, callTool(t, cs, name, args), "match tools are disabled", "--server")
	}
}

func TestMatchServerUnreachable(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	url := "http://" + ln.Addr().String()
	ln.Close() // nothing listens here now
	cs := connect(t, Options{ServerURL: url})
	expectToolError(t, callTool(t, cs, "get_match", map[string]any{"match_id": "0123456789abcdef"}), "cannot reach pf serve at "+url, `"pf serve"`, "--server")
}

func TestMatchServerSlow(t *testing.T) {
	block := make(chan struct{})
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-block }))
	defer ts.Close()
	defer close(block)
	cs := connect(t, Options{ServerURL: ts.URL, HTTPClient: &http.Client{Timeout: 100 * time.Millisecond}})
	start := time.Now()
	expectToolError(t, callTool(t, cs, "get_match", map[string]any{"match_id": "0123456789abcdef"}), "did not respond in time")
	if d := time.Since(start); d > 5*time.Second {
		t.Errorf("took %v", d)
	}
}
