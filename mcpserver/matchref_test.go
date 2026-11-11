package mcpserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"promptfighter/server"
)

func TestParseMatchRef(t *testing.T) {
	const id = "0123456789abcdef"
	ok := map[string]string{ // ref -> origin
		id:                   "",
		" 0123456789ABCDEF ": "",
		"http://127.0.0.1:8080/play/matches/" + id:                           "http://127.0.0.1:8080",
		"http://localhost:8080/watch/matches/" + id + "/rounds/2?autoplay=1": "http://localhost:8080",
		"https://pf.example.com/view/matches/" + id + "#f=3":                 "https://pf.example.com",
	}
	for ref, wantOrigin := range ok {
		if got, origin, err := parseMatchRef(ref); err != nil || got != id || origin != wantOrigin {
			t.Errorf("%q: got %q %q, %v", ref, got, origin, err)
		}
	}
	bad := map[string]string{
		"hello":                      "16-character match id",
		"http://127.0.0.1:8080/play": "no match id",
		"ftp://h/play/matches/" + id: "16-character match id",
	}
	for ref, want := range bad {
		if _, _, err := parseMatchRef(ref); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: error %v, want it to mention %q", ref, err, want)
		}
	}
}

func TestJoinHostedMatchByGameLink(t *testing.T) {
	ts := pfServe(t)
	// The host creates a hosted match over HTTP (as the web cover screen does).
	resp, err := http.Post(ts.URL+"/matches", "application/json", bytes.NewBufferString(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	var host struct {
		MatchID string      `json:"match_id"`
		Match   server.View `json:"match"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&host); err != nil || resp.StatusCode != 201 || host.Match.Status != server.StatusPlayers {
		t.Fatalf("host: %d %+v %v", resp.StatusCode, host, err)
	}
	link := ts.URL + "/play/matches/" + host.MatchID

	alice, bob := connect(t, Options{ServerURL: ts.URL}), connect(t, Options{ServerURL: ts.URL})
	var g MatchOut
	res := callTool(t, alice, "get_match", map[string]any{"match_id": link})
	structured(t, res, &g)
	if !strings.Contains(resultText(res), "waiting for players") {
		t.Errorf("get_match text for a hosted match:\n%s", resultText(res))
	}

	var a, b SeatOut
	res = callTool(t, alice, "join_match", map[string]any{"match_id": link, "name": "alice", "script": strong})
	structured(t, res, &a)
	if a.Seat != "a" || !strings.Contains(resultText(res), "You are seat a") {
		t.Fatalf("alice joined as %q:\n%s", a.Seat, resultText(res))
	}
	structured(t, callTool(t, bob, "join_match", map[string]any{"match_id": link, "name": "bob", "script": "loop { wait 10 }"}), &b)
	if b.Seat != "b" || len(b.Match.Rounds) != 1 {
		t.Fatalf("bob joined as %q, rounds %d", b.Seat, len(b.Match.Rounds))
	}
	// Submitting by link works too.
	var sub MatchOut
	structured(t, callTool(t, alice, "submit_script", map[string]any{"match_id": link, "token": a.Token, "script": strong}), &sub)
	if !sub.Match.Seats.A.Submitted {
		t.Errorf("submit by link: %+v", sub.Match.Seats.A)
	}
}

func TestGameLinkUsesItsOwnServer(t *testing.T) {
	configured, other := pfServe(t), pfServe(t)
	// The match lives on "other"; pf mcp was configured for "configured".
	resp, err := http.Post(other.URL+"/matches", "application/json", bytes.NewBufferString(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	var host struct {
		MatchID string `json:"match_id"`
	}
	json.NewDecoder(resp.Body).Decode(&host)
	link := other.URL + "/play/matches/" + host.MatchID
	c := connect(t, Options{ServerURL: configured.URL})
	var joined SeatOut
	structured(t, callTool(t, c, "join_match", map[string]any{"match_id": link, "name": "claude"}), &joined)
	if joined.Seat != "a" || joined.MatchID != host.MatchID {
		t.Fatalf("join via link on another server: %+v", joined)
	}
	// Later calls with the link go there too; the bare id is unknown to the configured server.
	var g MatchOut
	structured(t, callTool(t, c, "get_match", map[string]any{"match_id": link}), &g)
	if g.Match.Seats.A == nil || g.Match.Seats.A.Name != "claude" {
		t.Fatalf("get_match via link: %+v", g.Match)
	}
	expectToolError(t, callTool(t, c, "get_match", map[string]any{"match_id": host.MatchID}), "not_found")
	// Without any configured server, links still work and bare ids explain what to pass.
	none := connect(t, Options{ServerURL: ""})
	structured(t, callTool(t, none, "get_match", map[string]any{"match_id": link}), &g)
	expectToolError(t, callTool(t, none, "get_match", map[string]any{"match_id": host.MatchID}), "Pass the game link")
}

func TestJoinWithoutScriptOverMCP(t *testing.T) {
	ts := pfServe(t)
	resp, _ := http.Post(ts.URL+"/matches", "application/json", bytes.NewBufferString(`{}`))
	var host struct {
		MatchID string `json:"match_id"`
	}
	json.NewDecoder(resp.Body).Decode(&host)
	link := ts.URL + "/play/matches/" + host.MatchID
	a, b := connect(t, Options{}), connect(t, Options{})
	var sa, sb SeatOut
	res := callTool(t, a, "join_match", map[string]any{"match_id": link, "name": "alice"})
	structured(t, res, &sa)
	if !strings.Contains(resultText(res), "Nothing submitted yet") {
		t.Errorf("join text:\n%s", resultText(res))
	}
	structured(t, callTool(t, b, "join_match", map[string]any{"match_id": link, "name": "bob"}), &sb)
	if len(sb.Match.Rounds) != 0 {
		t.Fatalf("round 1 ran before any submission: %+v", sb.Match.Rounds)
	}
	callTool(t, a, "submit_script", map[string]any{"match_id": link, "token": sa.Token, "script": strong})
	var after MatchOut
	structured(t, callTool(t, b, "submit_script", map[string]any{"match_id": link, "token": sb.Token, "script": idle}), &after)
	if len(after.Match.Rounds) != 1 {
		t.Fatalf("round 1 after both submissions: %+v", after.Match.Rounds)
	}
}
