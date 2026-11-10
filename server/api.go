// Package server implements the Prompt Fighter HTTP/JSON API: script
// validation, single-round simulation and best-of-3 matches stored on disk.
package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"promptfighter/sim"
	"promptfighter/sim/dsl"
)

// Options configures a Server. Zero values select the defaults.
type Options struct {
	DataDir string        // required: root of the on-disk match store
	MaxSims int           // concurrent simulations; default runtime.NumCPU()
	SimWait time.Duration // how long a request waits for a simulation slot; default 5s
	MaxBody int64         // request body limit in bytes; default 64 KiB
}

const (
	defaultMaxBody = 64 << 10
	defaultSimWait = 5 * time.Second
	maxNameBytes   = 40
	maxSimFrames   = 600
)

// Server serves the API. It implements http.Handler.
type Server struct {
	opts  Options
	cfg   sim.Config
	store *store
	sims  chan struct{}
	mux   *http.ServeMux

	mu     sync.Mutex
	locks  map[string]*sync.Mutex
	routes []string // "METHOD /pattern", for docs checks
}

// New creates a Server whose matches live under opts.DataDir.
func New(opts Options) (*Server, error) {
	if opts.DataDir == "" {
		return nil, errors.New("server: DataDir is required")
	}
	if opts.MaxSims <= 0 {
		opts.MaxSims = runtime.NumCPU()
	}
	if opts.SimWait <= 0 {
		opts.SimWait = defaultSimWait
	}
	if opts.MaxBody <= 0 {
		opts.MaxBody = defaultMaxBody
	}
	st, err := newStore(opts.DataDir)
	if err != nil {
		return nil, err
	}
	s := &Server{
		opts:  opts,
		cfg:   sim.DefaultConfig(),
		store: st,
		sims:  make(chan struct{}, opts.MaxSims),
		mux:   http.NewServeMux(),
		locks: map[string]*sync.Mutex{},
	}
	s.route("/healthz", s.handleHealth, http.MethodGet)
	s.route("/validate", s.handleValidate, http.MethodPost)
	s.route("/sim", s.handleSim, http.MethodPost)
	s.route("/matches", s.handleCreateMatch, http.MethodPost)
	s.route("/matches/{id}", s.handleGetMatch, http.MethodGet)
	s.route("/matches/{id}/join", s.handleJoin, http.MethodPost)
	s.route("/matches/{id}/scripts", s.handleSubmit, http.MethodPost)
	s.route("/matches/{id}/rounds/{n}", s.handleGetRound, http.MethodGet)
	s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, errNotFound("no such endpoint: "+r.URL.Path))
	})
	return s, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }

// Routes lists the registered routes as "METHOD /pattern", sorted.
func (s *Server) Routes() []string {
	out := append([]string(nil), s.routes...)
	sort.Strings(out)
	return out
}

type handlerFunc func(w http.ResponseWriter, r *http.Request) *apiError

// route registers a method-less pattern and enforces the allowed methods
// itself, so a wrong method gets a JSON 405 rather than the mux's plain text.
func (s *Server) route(pattern string, h handlerFunc, methods ...string) {
	allow := strings.Join(methods, ", ")
	for _, m := range methods {
		s.routes = append(s.routes, m+" "+pattern)
	}
	s.mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		ok := false
		for _, m := range methods {
			ok = ok || r.Method == m
		}
		if !ok {
			w.Header().Set("Allow", allow)
			writeError(w, &apiError{status: http.StatusMethodNotAllowed, code: "method_not_allowed",
				msg: fmt.Sprintf("%s is not allowed on %s; use %s", r.Method, pattern, allow)})
			return
		}
		if e := h(w, r); e != nil {
			writeError(w, e)
		}
	})
}

// --- errors ---

// ScriptError is one validation error in an invalid_script response.
type ScriptError struct {
	Player  string `json:"player,omitempty"`
	Line    int    `json:"line"`
	Col     int    `json:"col"`
	Message string `json:"message"`
	Text    string `json:"text"`
}

type apiError struct {
	status int
	code   string
	msg    string
	errs   []ScriptError
}

func (e *apiError) Error() string { return e.code + ": " + e.msg }

func errBadRequest(format string, a ...any) *apiError {
	return &apiError{status: http.StatusBadRequest, code: "bad_request", msg: fmt.Sprintf(format, a...)}
}
func errNotFound(msg string) *apiError {
	return &apiError{status: http.StatusNotFound, code: "not_found", msg: msg}
}
func errUnauthorized(msg string) *apiError {
	return &apiError{status: http.StatusUnauthorized, code: "unauthorized", msg: msg}
}
func errConflict(msg string) *apiError {
	return &apiError{status: http.StatusConflict, code: "conflict", msg: msg}
}
func errBusy() *apiError {
	return &apiError{status: http.StatusServiceUnavailable, code: "busy", msg: "all simulation slots are in use; retry shortly"}
}
func errInternal(err error) *apiError {
	return &apiError{status: http.StatusInternalServerError, code: "internal", msg: err.Error()}
}

func errInvalidScript(errs []ScriptError) *apiError {
	return &apiError{status: http.StatusUnprocessableEntity, code: "invalid_script", msg: "script is invalid", errs: errs}
}

func toScriptErrors(player string, errs []dsl.Error) []ScriptError {
	out := make([]ScriptError, 0, len(errs))
	for _, e := range errs {
		out = append(out, ScriptError{Player: player, Line: e.Line, Col: e.Col, Message: e.Msg, Text: e.String()})
	}
	return out
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	b, err := json.Marshal(v)
	if err != nil {
		status, b = http.StatusInternalServerError, []byte(`{"error":{"code":"internal","message":"encoding failed"}}`)
	}
	writeRaw(w, status, append(b, '\n'))
}

func writeRaw(w http.ResponseWriter, status int, b []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(b)
}

func writeError(w http.ResponseWriter, e *apiError) {
	body := struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
		Errors []ScriptError `json:"errors,omitempty"`
	}{}
	body.Error.Code, body.Error.Message, body.Errors = e.code, e.msg, e.errs
	writeJSON(w, e.status, body)
}

// --- request decoding ---

// decode reads exactly one JSON value into v, rejecting unknown fields,
// trailing data and bodies over the size limit.
func (s *Server) decode(w http.ResponseWriter, r *http.Request, v any) *apiError {
	r.Body = http.MaxBytesReader(w, r.Body, s.opts.MaxBody)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return decodeError(err, s.opts.MaxBody)
	}
	if _, err := dec.Token(); err != io.EOF {
		if err != nil {
			return decodeError(err, s.opts.MaxBody)
		}
		return errBadRequest("unexpected data after the JSON body")
	}
	return nil
}

func decodeError(err error, limit int64) *apiError {
	var mbe *http.MaxBytesError
	if errors.As(err, &mbe) {
		return &apiError{status: http.StatusRequestEntityTooLarge, code: "payload_too_large",
			msg: fmt.Sprintf("request body exceeds %d bytes", limit)}
	}
	if errors.Is(err, io.EOF) {
		return errBadRequest("request body is empty; expected a JSON object")
	}
	return errBadRequest("invalid JSON body: %v", err)
}

// --- simulation slots ---

func (s *Server) acquire(ctx context.Context) bool {
	t := time.NewTimer(s.opts.SimWait)
	defer t.Stop()
	select {
	case s.sims <- struct{}{}:
		return true
	case <-t.C:
		return false
	case <-ctx.Done():
		return false
	}
}

func (s *Server) release() { <-s.sims }

// simulate runs one round under the server config, holding a slot.
func (s *Server) simulate(ctx context.Context, a, b sim.Entrant, frames int) (*sim.Log, *apiError) {
	if !s.acquire(ctx) {
		return nil, errBusy()
	}
	defer s.release()
	cfg := s.cfg
	cfg.FrameBudget = frames
	l, err := sim.Simulate(cfg, a, b)
	if err != nil {
		var se *sim.ScriptError
		if errors.As(err, &se) {
			return nil, errInvalidScript(toScriptErrors(se.Player, se.Errors))
		}
		return nil, errInternal(err)
	}
	return l, nil
}

// --- stateless endpoints ---

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) *apiError {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":         "ok",
		"config_version": s.cfg.Version,
		"format_version": sim.FormatVersion,
		"store_version":  storeVersion,
	})
	return nil
}

func (s *Server) validateScript(player, src string) []ScriptError {
	return toScriptErrors(player, dsl.Validate(src, s.cfg.Limits))
}

func (s *Server) handleValidate(w http.ResponseWriter, r *http.Request) *apiError {
	var req struct {
		Script *string `json:"script"`
	}
	if e := s.decode(w, r, &req); e != nil {
		return e
	}
	if req.Script == nil {
		return errBadRequest("missing field \"script\"")
	}
	errs := s.validateScript("", *req.Script)
	writeJSON(w, http.StatusOK, map[string]any{"valid": len(errs) == 0, "errors": errs})
	return nil
}

type entrantReq struct {
	Name   string  `json:"name"`
	Script *string `json:"script"`
}

func (s *Server) handleSim(w http.ResponseWriter, r *http.Request) *apiError {
	var req struct {
		P1     *entrantReq `json:"p1"`
		P2     *entrantReq `json:"p2"`
		Frames *int        `json:"frames"`
	}
	if e := s.decode(w, r, &req); e != nil {
		return e
	}
	frames := s.cfg.FrameBudget
	if req.Frames != nil {
		if *req.Frames < 1 || *req.Frames > maxSimFrames {
			return errBadRequest("frames must be between 1 and %d (got %d)", maxSimFrames, *req.Frames)
		}
		frames = *req.Frames
	}
	var ents [2]sim.Entrant
	var errs []ScriptError
	for i, p := range []*entrantReq{req.P1, req.P2} {
		id := fmt.Sprintf("p%d", i+1)
		if p == nil || p.Script == nil {
			return errBadRequest("missing field \"%s.script\"", id)
		}
		if len(p.Name) > maxNameBytes {
			return errBadRequest("%s.name must be at most %d bytes", id, maxNameBytes)
		}
		name := p.Name
		if name == "" {
			name = id
		}
		ents[i] = sim.Entrant{Name: name, Source: *p.Script}
		errs = append(errs, s.validateScript(id, *p.Script)...)
	}
	if len(errs) > 0 {
		return errInvalidScript(errs)
	}
	l, e := s.simulate(r.Context(), ents[0], ents[1], frames)
	if e != nil {
		return e
	}
	writeRaw(w, http.StatusOK, l.JSON())
	return nil
}
