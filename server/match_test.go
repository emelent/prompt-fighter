package server

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"
)

func strp(s string) *string { return &s }

// readyMatch returns a match with both seats joined and submitted.
func readyMatch(t *testing.T) *Match {
	t.Helper()
	m := newMatch("0123456789abcdef", "v", "alice", "A1", hashToken("ta"))
	if _, err := m.join("bob", strp("B1"), hashToken("tb")); err != nil {
		t.Fatal(err)
	}
	return m
}

// play submits both seats (if needed) and applies an engine result in p1/p2 terms.
func play(m *Match, engineWinner string) Round {
	if m.A.Pending == nil {
		m.submit("a", "A")
	}
	if m.B.Pending == nil {
		m.submit("b", "B")
	}
	var w *string
	if engineWinner != "" {
		w = strp(engineWinner)
	}
	return m.applyRound(roundResult{Winner: w, Reason: "ko", HP: [2]int{10, 20}})
}

func TestMatchCreateAndJoin(t *testing.T) {
	m := newMatch("0123456789abcdef", "v", "alice", "A1", hashToken("ta"))
	if m.Status != StatusWaiting || m.B != nil || m.readyToPlay() {
		t.Fatalf("new match %+v", m)
	}
	if _, err := m.join("bob", strp("B1"), hashToken("tb")); err != nil || m.Status != StatusAwaiting || !m.readyToPlay() {
		t.Fatalf("after join: %v %+v", err, m)
	}
	if _, err := m.join("carol", strp("C"), hashToken("tc")); err != errSeatTaken {
		t.Errorf("second join: %v, want errSeatTaken", err)
	}
}

func TestMatchSidesAlternate(t *testing.T) {
	m := readyMatch(t)
	var got []string
	for i := 0; i < 3; i++ {
		got = append(got, play(m, "").P1)
	}
	if strings.Join(got, "") != "aba" {
		t.Errorf("p1 seats %v, want a b a", got)
	}
}

func TestMatchRoundResultMapsToSeats(t *testing.T) {
	m := readyMatch(t)
	play(m, "")        // round 1 draw
	r := play(m, "p1") // round 2: seat b is p1
	if r.P1 != "b" || r.Winner == nil || *r.Winner != "b" || r.HP != (SeatHP{A: 20, B: 10}) {
		t.Errorf("round 2 %+v", r)
	}
	if m.B.Wins != 1 || m.A.Wins != 0 {
		t.Errorf("wins a=%d b=%d", m.A.Wins, m.B.Wins)
	}
}

func TestMatchTwoStraightWins(t *testing.T) {
	m := readyMatch(t)
	play(m, "p1") // round 1: a is p1
	play(m, "p2") // round 2: a is p2
	if m.Status != StatusFinished || m.Winner == nil || *m.Winner != "a" || len(m.Rounds) != 2 {
		t.Errorf("status %s winner %v rounds %d", m.Status, m.Winner, len(m.Rounds))
	}
}

func TestMatchDrawsExtend(t *testing.T) {
	m := readyMatch(t)
	play(m, "")   // 1: draw
	play(m, "p1") // 2: b is p1 -> b wins
	play(m, "p1") // 3: a is p1 -> a wins
	if m.Status != StatusAwaiting || m.A.Wins != 1 || m.B.Wins != 1 || m.nextRound() != 4 {
		t.Errorf("status %s wins %d-%d next %d", m.Status, m.A.Wins, m.B.Wins, m.nextRound())
	}
	if m.A.Pending != nil || m.B.Pending != nil {
		t.Errorf("pending submissions must clear after a round")
	}
}

func TestMatchRoundCapDraw(t *testing.T) {
	m := readyMatch(t)
	play(m, "")   // 1 draw
	play(m, "p2") // 2: a is p2 -> a
	play(m, "")   // 3 draw
	play(m, "p1") // 4: b is p1 -> b
	play(m, "")   // 5 draw
	if m.Status != StatusFinished || m.Winner != nil || len(m.Rounds) != 5 {
		t.Errorf("status %s winner %v rounds %d", m.Status, m.Winner, len(m.Rounds))
	}
}

func TestMatchRoundCapMoreWins(t *testing.T) {
	m := readyMatch(t)
	play(m, "p1") // 1: a
	for i := 0; i < 4; i++ {
		play(m, "")
	}
	if m.Status != StatusFinished || m.Winner == nil || *m.Winner != "a" {
		t.Errorf("status %s winner %v", m.Status, m.Winner)
	}
}

func TestMatchReplaceBeforeRound(t *testing.T) {
	m := newMatch("0123456789abcdef", "v", "alice", "A1", hashToken("ta"))
	m.submit("a", "Y") // while waiting for an opponent
	m.join("bob", strp("B1"), hashToken("tb"))
	if *m.A.Pending != "Y" {
		t.Errorf("pending %q, want Y", *m.A.Pending)
	}
	play(m, "")
	m.submit("a", "X")
	m.submit("a", "Z")
	if *m.A.Pending != "Z" || m.readyToPlay() {
		t.Errorf("pending %q ready %v", *m.A.Pending, m.readyToPlay())
	}
}

func TestMatchSubmitAfterFinish(t *testing.T) {
	m := readyMatch(t)
	play(m, "p1")
	play(m, "p2")
	if err := m.submit("a", "X"); err != errMatchFinished {
		t.Errorf("err %v", err)
	}
	if m.readyToPlay() {
		t.Errorf("finished match must not be ready")
	}
}

func TestMatchViewHidesSecrets(t *testing.T) {
	m := readyMatch(t)
	play(m, "")
	m.submit("a", "SECRET_REVISION")
	b, _ := json.Marshal(m.view())
	s := string(b)
	if strings.Contains(s, "SECRET_REVISION") || strings.Contains(s, hashToken("ta")) || strings.Contains(s, "token") {
		t.Errorf("view leaks secrets: %s", s)
	}
	v := m.view()
	if !v.Seats.A.Submitted || v.Seats.B.Submitted || v.NextRound != 2 || v.Rounds[0].Log != "/matches/0123456789abcdef/rounds/1" {
		t.Errorf("view %+v", v)
	}
}

func TestMatchSeatForToken(t *testing.T) {
	m := readyMatch(t)
	if m.seatFor(hashToken("ta")) != "a" || m.seatFor(hashToken("tb")) != "b" || m.seatFor(hashToken("nope")) != "" {
		t.Errorf("seatFor wrong")
	}
}

func TestMatchIDsAndTokens(t *testing.T) {
	id, err := newMatchID()
	tok, hash, err2 := newToken()
	if err != nil || err2 != nil {
		t.Fatal(err, err2)
	}
	if !regexp.MustCompile(`^[0-9a-f]{16}$`).MatchString(id) || !regexp.MustCompile(`^[0-9a-f]{32}$`).MatchString(tok) {
		t.Errorf("id %q token %q", id, tok)
	}
	if hash == tok || hash != hashToken(tok) || len(hash) != 64 {
		t.Errorf("hash %q", hash)
	}
	id2, _ := newMatchID()
	if id == id2 {
		t.Errorf("ids repeat")
	}
}
