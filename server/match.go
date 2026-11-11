package server

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
)

// Match statuses.
const (
	StatusPlayers  = "waiting_for_players" // hosted match: no seat taken yet
	StatusWaiting  = "waiting_for_opponent"
	StatusAwaiting = "awaiting_scripts"
	StatusFinished = "finished"
)

const (
	winsNeeded = 2
	maxRounds  = 5
)

var (
	errSeatTaken     = errors.New("both seats are already taken")
	errMatchFinished = errors.New("the match is finished")
)

// Seat is one side of a match as stored on disk. Pending is the script
// submitted for the next round; it is never exposed through the API.
type Seat struct {
	Name      string  `json:"name"`
	TokenHash string  `json:"token_hash"`
	Pending   *string `json:"pending,omitempty"`
	Wins      int     `json:"wins"`
}

// SeatHP is a pair of per-seat values.
type SeatHP struct {
	A int `json:"a"`
	B int `json:"b"`
}

// Round is the record of one played round, in seat terms.
type Round struct {
	N             int     `json:"n"`
	P1            string  `json:"p1"`     // seat ("a"/"b") that played as p1
	Winner        *string `json:"winner"` // seat, or null for a drawn round
	Reason        string  `json:"reason"`
	EndFrame      int     `json:"end_frame"`
	HP            SeatHP  `json:"hp"`
	ConfigVersion string  `json:"config_version"`
}

// Match is the stored state of a best-of-3 match.
type Match struct {
	StoreVersion  int     `json:"store_version"`
	ID            string  `json:"match_id"`
	ConfigVersion string  `json:"config_version"` // engine config at creation
	Status        string  `json:"status"`
	A             *Seat   `json:"a"`
	B             *Seat   `json:"b"`
	Rounds        []Round `json:"rounds"`
	Winner        *string `json:"winner"`
}

func newMatch(id, configVersion, name, script, tokenHash string) *Match {
	return &Match{
		StoreVersion:  storeVersion,
		ID:            id,
		ConfigVersion: configVersion,
		Status:        StatusWaiting,
		A:             &Seat{Name: name, TokenHash: tokenHash, Pending: &script},
		Rounds:        []Round{},
	}
}

// newHostedMatch creates a match set up by a host who only watches: both
// seats are free, and players take them by joining.
func newHostedMatch(id, configVersion string) *Match {
	return &Match{
		StoreVersion:  storeVersion,
		ID:            id,
		ConfigVersion: configVersion,
		Status:        StatusPlayers,
		Rounds:        []Round{},
	}
}

// freeSeat returns the seat the next joiner takes ("a" before "b"), or "".
func (m *Match) freeSeat() string {
	switch {
	case m.A == nil:
		return "a"
	case m.B == nil:
		return "b"
	}
	return ""
}

func (m *Match) seat(id string) *Seat {
	if id == "a" {
		return m.A
	}
	return m.B
}

// join puts a player in the first free seat and returns it. Filling seat a of
// a hosted match leaves it waiting for an opponent; filling the last seat
// makes it ready for round 1.
//
// script may be nil: the seat then starts with nothing submitted, and the
// player submits round 1 like any other round.
func (m *Match) join(name string, script *string, tokenHash string) (string, error) {
	st := &Seat{Name: name, TokenHash: tokenHash, Pending: script}
	switch m.freeSeat() {
	case "a":
		m.A = st
		m.Status = StatusWaiting
		return "a", nil
	case "b":
		m.B = st
		if m.Status == StatusWaiting {
			m.Status = StatusAwaiting
		}
		return "b", nil
	}
	return "", errSeatTaken
}

// seatFor returns the seat id whose token hashes to hash, or "".
func (m *Match) seatFor(hash string) string {
	found := ""
	for _, id := range []string{"a", "b"} {
		if st := m.seat(id); st != nil && subtle.ConstantTimeCompare([]byte(st.TokenHash), []byte(hash)) == 1 {
			found = id
		}
	}
	return found
}

func (m *Match) submit(seat, script string) error {
	if m.Status == StatusFinished {
		return errMatchFinished
	}
	m.seat(seat).Pending = &script
	return nil
}

func (m *Match) nextRound() int { return len(m.Rounds) + 1 }

// readyToPlay reports whether both seats have a submission for the next round.
func (m *Match) readyToPlay() bool {
	return m.Status != StatusFinished && m.A != nil && m.B != nil && m.A.Pending != nil && m.B.Pending != nil
}

// sides returns the seats playing p1 and p2 in round n: seat a is p1 in odd rounds.
func sides(n int) (p1, p2 string) {
	if n%2 == 1 {
		return "a", "b"
	}
	return "b", "a"
}

// roundResult is an engine result in p1/p2 terms.
type roundResult struct {
	Winner        *string // "p1", "p2" or nil
	Reason        string
	EndFrame      int
	HP            [2]int // p1, p2
	ConfigVersion string
}

// applyRound records the result of the next round and updates wins, pending
// submissions and status.
func (m *Match) applyRound(res roundResult) Round {
	n := m.nextRound()
	p1, p2 := sides(n)
	r := Round{N: n, P1: p1, Reason: res.Reason, EndFrame: res.EndFrame, ConfigVersion: res.ConfigVersion}
	hp := map[string]int{p1: res.HP[0], p2: res.HP[1]}
	r.HP = SeatHP{A: hp["a"], B: hp["b"]}
	if res.Winner != nil {
		w := p1
		if *res.Winner == "p2" {
			w = p2
		}
		r.Winner = &w
		m.seat(w).Wins++
	}
	m.Rounds = append(m.Rounds, r)
	m.A.Pending, m.B.Pending = nil, nil
	switch {
	case m.A.Wins >= winsNeeded:
		m.finish("a")
	case m.B.Wins >= winsNeeded:
		m.finish("b")
	case n >= maxRounds:
		switch {
		case m.A.Wins > m.B.Wins:
			m.finish("a")
		case m.B.Wins > m.A.Wins:
			m.finish("b")
		default:
			m.finish("")
		}
	default:
		m.Status = StatusAwaiting
	}
	return r
}

func (m *Match) finish(winner string) {
	m.Status = StatusFinished
	if winner != "" {
		m.Winner = &winner
	}
}

// --- public view ---

// SeatView is the public state of a seat.
type SeatView struct {
	Name      string `json:"name"`
	Wins      int    `json:"wins"`
	Submitted bool   `json:"submitted"`
}

// RoundView is a played round plus the URL of its event log.
type RoundView struct {
	Round
	Log string `json:"log"`
}

// View is the public match state. It never includes pending scripts or tokens.
type View struct {
	MatchID       string `json:"match_id"`
	Status        string `json:"status"`
	ConfigVersion string `json:"config_version"`
	NextRound     int    `json:"next_round"`
	Seats         struct {
		A *SeatView `json:"a"`
		B *SeatView `json:"b"`
	} `json:"seats"`
	Rounds []RoundView `json:"rounds"`
	Winner *string     `json:"winner"`
}

func seatView(s *Seat) *SeatView {
	if s == nil {
		return nil
	}
	return &SeatView{Name: s.Name, Wins: s.Wins, Submitted: s.Pending != nil}
}

func (m *Match) view() View {
	v := View{MatchID: m.ID, Status: m.Status, ConfigVersion: m.ConfigVersion, NextRound: m.nextRound(), Winner: m.Winner}
	v.Seats.A, v.Seats.B = seatView(m.A), seatView(m.B)
	v.Rounds = make([]RoundView, 0, len(m.Rounds))
	for _, r := range m.Rounds {
		v.Rounds = append(v.Rounds, RoundView{Round: r, Log: fmt.Sprintf("/matches/%s/rounds/%d", m.ID, r.N)})
	}
	return v
}

// --- ids and tokens ---

func randomHex(nbytes int) (string, error) {
	b := make([]byte, nbytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func newMatchID() (string, error) { return randomHex(8) }

// newToken returns a fresh seat token and its stored hash.
func newToken() (token, hash string, err error) {
	token, err = randomHex(16)
	return token, hashToken(token), err
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
