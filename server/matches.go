package server

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"promptfighter/sim"
)

// lockMatch serialises all mutations of one match within this process.
func (s *Server) lockMatch(id string) func() {
	s.mu.Lock()
	l, ok := s.locks[id]
	if !ok {
		l = &sync.Mutex{}
		s.locks[id] = l
	}
	s.mu.Unlock()
	l.Lock()
	return l.Unlock
}

func (s *Server) loadMatch(id string) (*Match, *apiError) {
	m, err := s.store.load(id)
	if errors.Is(err, errNoMatch) {
		return nil, errNotFound("no match with id " + strconv.Quote(id))
	}
	if err != nil {
		return nil, errInternal(err)
	}
	return m, nil
}

type seatReq struct {
	Name   string  `json:"name"`
	Script *string `json:"script"`
}

// check validates a create/join body.
func (s *Server) check(req seatReq, seat string) *apiError {
	if req.Script == nil {
		return errBadRequest("missing field \"script\"")
	}
	if len(req.Name) < 1 || len(req.Name) > maxNameBytes {
		return errBadRequest("name must be 1 to %d bytes", maxNameBytes)
	}
	if errs := s.validateScript(seat, *req.Script); len(errs) > 0 {
		return errInvalidScript(errs)
	}
	return nil
}

// playIfReady runs the next round when both seats have submitted. If no
// simulation slot is free the submissions stay pending and the round runs on
// a later request that finds the match ready.
func (s *Server) playIfReady(ctx context.Context, m *Match) *apiError {
	if !m.readyToPlay() {
		return nil
	}
	n := m.nextRound()
	p1, p2 := sides(n)
	a := sim.Entrant{Name: m.seat(p1).Name, Source: *m.seat(p1).Pending}
	b := sim.Entrant{Name: m.seat(p2).Name, Source: *m.seat(p2).Pending}
	l, e := s.simulate(ctx, a, b, s.cfg.FrameBudget)
	if e != nil {
		if e.code == "busy" {
			return nil
		}
		return e
	}
	// The round log is written before match.json, which is the source of truth.
	if err := s.store.saveRound(m.ID, n, l.JSON()); err != nil {
		return errInternal(err)
	}
	m.applyRound(roundResult{Winner: l.Result.Winner, Reason: l.Result.Reason, EndFrame: l.Result.EndFrame, HP: l.Result.HP, ConfigVersion: l.ConfigVersion})
	return nil
}

type seatResp struct {
	MatchID string `json:"match_id"`
	Seat    string `json:"seat"`
	Token   string `json:"token"`
	Match   View   `json:"match"`
}

func (s *Server) handleCreateMatch(w http.ResponseWriter, r *http.Request) *apiError {
	var req seatReq
	if e := s.decode(w, r, &req); e != nil {
		return e
	}
	if e := s.check(req, "a"); e != nil {
		return e
	}
	token, hash, err := newToken()
	if err != nil {
		return errInternal(err)
	}
	var id string
	for {
		if id, err = newMatchID(); err != nil {
			return errInternal(err)
		}
		if !s.store.exists(id) {
			break
		}
	}
	unlock := s.lockMatch(id)
	defer unlock()
	m := newMatch(id, s.cfg.Version, req.Name, *req.Script, hash)
	if err := s.store.save(m); err != nil {
		return errInternal(err)
	}
	writeJSON(w, http.StatusCreated, seatResp{MatchID: id, Seat: "a", Token: token, Match: m.view()})
	return nil
}

func (s *Server) handleJoin(w http.ResponseWriter, r *http.Request) *apiError {
	id := r.PathValue("id")
	var req seatReq
	if e := s.decode(w, r, &req); e != nil {
		return e
	}
	unlock := s.lockMatch(id)
	defer unlock()
	m, e := s.loadMatch(id)
	if e != nil {
		return e
	}
	if m.B != nil {
		return errConflict(errSeatTaken.Error())
	}
	if e := s.check(req, "b"); e != nil {
		return e
	}
	token, hash, err := newToken()
	if err != nil {
		return errInternal(err)
	}
	if err := m.join(req.Name, *req.Script, hash); err != nil {
		return errConflict(err.Error())
	}
	if e := s.playIfReady(r.Context(), m); e != nil {
		return e
	}
	if err := s.store.save(m); err != nil {
		return errInternal(err)
	}
	writeJSON(w, http.StatusCreated, seatResp{MatchID: id, Seat: "b", Token: token, Match: m.view()})
	return nil
}

func bearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if t, ok := strings.CutPrefix(h, "Bearer "); ok {
		return strings.TrimSpace(t)
	}
	return ""
}

func (s *Server) handleSubmit(w http.ResponseWriter, r *http.Request) *apiError {
	id := r.PathValue("id")
	token := bearer(r)
	if token == "" {
		return errUnauthorized("missing seat token; send \"Authorization: Bearer <token>\"")
	}
	var req struct {
		Script *string `json:"script"`
	}
	if e := s.decode(w, r, &req); e != nil {
		return e
	}
	if req.Script == nil {
		return errBadRequest("missing field \"script\"")
	}
	unlock := s.lockMatch(id)
	defer unlock()
	m, e := s.loadMatch(id)
	if e != nil {
		return e
	}
	seat := m.seatFor(hashToken(token))
	if seat == "" {
		return errUnauthorized("token does not belong to a seat in this match")
	}
	if m.Status == StatusFinished {
		return errConflict(errMatchFinished.Error())
	}
	if errs := s.validateScript(seat, *req.Script); len(errs) > 0 {
		return errInvalidScript(errs)
	}
	if err := m.submit(seat, *req.Script); err != nil {
		return errConflict(err.Error())
	}
	if e := s.playIfReady(r.Context(), m); e != nil {
		return e
	}
	if err := s.store.save(m); err != nil {
		return errInternal(err)
	}
	writeJSON(w, http.StatusOK, map[string]any{"match": m.view()})
	return nil
}

func (s *Server) handleGetMatch(w http.ResponseWriter, r *http.Request) *apiError {
	m, e := s.loadMatch(r.PathValue("id"))
	if e != nil {
		return e
	}
	writeJSON(w, http.StatusOK, m.view())
	return nil
}

func (s *Server) handleGetRound(w http.ResponseWriter, r *http.Request) *apiError {
	id := r.PathValue("id")
	m, e := s.loadMatch(id)
	if e != nil {
		return e
	}
	n, err := strconv.Atoi(r.PathValue("n"))
	if err != nil || n < 1 || n > len(m.Rounds) {
		return errNotFound("round " + r.PathValue("n") + " has not been played")
	}
	b, err := s.store.loadRound(id, n)
	if err != nil {
		return errInternal(err)
	}
	writeRaw(w, http.StatusOK, b)
	return nil
}
