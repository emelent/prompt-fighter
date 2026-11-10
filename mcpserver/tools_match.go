package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"promptfighter/server"
	"promptfighter/sim"
)

// --- inputs and outputs ---

// CreateIn is the input of create_match.
type CreateIn struct {
	Name   string `json:"name" jsonschema:"your fighter name shown to the opponent (1-40 bytes)"`
	Script string `json:"script" jsonschema:"your .pf script for round 1"`
}

// JoinIn is the input of join_match.
type JoinIn struct {
	MatchID string `json:"match_id" jsonschema:"the 16-character match id shared by the match creator"`
	Name    string `json:"name" jsonschema:"your fighter name (1-40 bytes)"`
	Script  string `json:"script" jsonschema:"your .pf script for round 1"`
}

// SubmitIn is the input of submit_script.
type SubmitIn struct {
	MatchID string `json:"match_id"`
	Token   string `json:"token" jsonschema:"your seat token from create_match or join_match"`
	Script  string `json:"script" jsonschema:"your .pf script for the next round (resubmit the same script to keep it)"`
}

// MatchIn identifies a match.
type MatchIn struct {
	MatchID string `json:"match_id"`
}

// ReplayIn is the input of get_round_replay.
type ReplayIn struct {
	MatchID       string `json:"match_id"`
	Round         int    `json:"round" jsonschema:"round number, starting at 1"`
	IncludeChecks bool   `json:"include_checks,omitempty" jsonschema:"also list every sensor read (verbose)"`
}

// SeatOut is the output of create_match and join_match.
type SeatOut struct {
	MatchID string      `json:"match_id"`
	Seat    string      `json:"seat" jsonschema:"your seat: a or b"`
	Token   string      `json:"token" jsonschema:"your secret seat token; shown only once, pass it to submit_script"`
	Match   server.View `json:"match"`
}

// MatchOut is the output of submit_script and get_match.
type MatchOut struct {
	Match server.View `json:"match"`
}

// ReplayOut is the output of get_round_replay.
type ReplayOut struct {
	Round  int    `json:"round"`
	P1Seat string `json:"p1_seat" jsonschema:"the seat (a or b) that played p1 this round"`
	Summary
}

// --- rendering ---

func seatLabel(v server.View, seat string) string {
	sv := v.Seats.A
	if seat == "b" {
		sv = v.Seats.B
	}
	if sv == nil {
		return "seat " + seat + " (empty)"
	}
	return fmt.Sprintf("seat %s (%s)", seat, sv.Name)
}

// matchText summarises match state with a next-step hint. mySeat is "" when
// the caller's seat is unknown.
func matchText(v server.View, mySeat string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "match %s: %s\n", v.MatchID, strings.ReplaceAll(v.Status, "_", " "))
	for _, seat := range []string{"a", "b"} {
		sv := v.Seats.A
		if seat == "b" {
			sv = v.Seats.B
		}
		you := ""
		if seat == mySeat {
			you = " [you]"
		}
		if sv == nil {
			fmt.Fprintf(&b, "seat %s%s: not joined yet\n", seat, you)
			continue
		}
		sub := "not submitted"
		if sv.Submitted {
			sub = "submitted"
		}
		fmt.Fprintf(&b, "seat %s (%s)%s: %d wins, next-round script %s\n", seat, sv.Name, you, sv.Wins, sub)
	}
	for _, r := range v.Rounds {
		w := "draw"
		if r.Winner != nil {
			w = seatLabel(v, *r.Winner) + " won"
		}
		fmt.Fprintf(&b, "round %d: %s was p1; %s by %s at f%d (hp a=%d b=%d); replay: get_round_replay round=%d\n",
			r.N, seatLabel(v, r.P1), w, r.Reason, r.EndFrame, r.HP.A, r.HP.B, r.N)
	}
	b.WriteString("next: " + nextHint(v, mySeat) + "\n")
	return b.String()
}

func nextHint(v server.View, mySeat string) string {
	switch v.Status {
	case server.StatusWaiting:
		return fmt.Sprintf("share match_id %s with your opponent; they call join_match with it. You can still replace your round-1 script with submit_script.", v.MatchID)
	case server.StatusFinished:
		if v.Winner == nil {
			return "match finished as a draw."
		}
		return "match finished: " + seatLabel(v, *v.Winner) + " won."
	}
	mine, theirs := v.Seats.A, v.Seats.B
	if mySeat == "b" {
		mine, theirs = v.Seats.B, v.Seats.A
	}
	switch {
	case mySeat == "":
		var pending []string
		for _, seat := range []string{"a", "b"} {
			sv := v.Seats.A
			if seat == "b" {
				sv = v.Seats.B
			}
			if sv != nil && !sv.Submitted {
				pending = append(pending, "seat "+seat)
			}
		}
		return fmt.Sprintf("round %d runs once both seats submit (waiting on %s).", v.NextRound, strings.Join(pending, " and "))
	case !mine.Submitted:
		return fmt.Sprintf("submit your script for round %d with submit_script (resubmit the same script to keep it). Review the last replay first.", v.NextRound)
	case theirs != nil && !theirs.Submitted:
		return fmt.Sprintf("waiting for your opponent to submit round %d; poll get_match.", v.NextRound)
	}
	return fmt.Sprintf("round %d is ready and will run on the next submission.", v.NextRound)
}

// --- handlers ---

func matchPath(id string, rest ...string) string {
	return "/matches/" + url.PathEscape(id) + strings.Join(rest, "")
}

type seatResp struct {
	MatchID string      `json:"match_id"`
	Seat    string      `json:"seat"`
	Token   string      `json:"token"`
	Match   server.View `json:"match"`
}

func (h *handler) seatResult(r seatResp) (*mcp.CallToolResult, SeatOut, error) {
	out := SeatOut(r)
	text := fmt.Sprintf("You are seat %s in match %s.\nYour seat token (shown only once; keep it and pass it to submit_script): %s\n\n%s",
		r.Seat, r.MatchID, r.Token, matchText(r.Match, r.Seat))
	return textResult(text), out, nil
}

func (h *handler) createMatch(ctx context.Context, _ *mcp.CallToolRequest, in CreateIn) (*mcp.CallToolResult, SeatOut, error) {
	var r seatResp
	if err := h.api.doJSON(ctx, "POST", "/matches", "", map[string]string{"name": in.Name, "script": in.Script}, &r); err != nil {
		return nil, SeatOut{}, err
	}
	return h.seatResult(r)
}

func (h *handler) joinMatch(ctx context.Context, _ *mcp.CallToolRequest, in JoinIn) (*mcp.CallToolResult, SeatOut, error) {
	var r seatResp
	if err := h.api.doJSON(ctx, "POST", matchPath(in.MatchID, "/join"), "", map[string]string{"name": in.Name, "script": in.Script}, &r); err != nil {
		return nil, SeatOut{}, err
	}
	return h.seatResult(r)
}

func (h *handler) submitScript(ctx context.Context, _ *mcp.CallToolRequest, in SubmitIn) (*mcp.CallToolResult, MatchOut, error) {
	if in.Token == "" {
		return nil, MatchOut{}, fmt.Errorf("token is required: use the seat token returned by create_match or join_match")
	}
	var r MatchOut
	if err := h.api.doJSON(ctx, "POST", matchPath(in.MatchID, "/scripts"), in.Token, map[string]string{"script": in.Script}, &r); err != nil {
		return nil, MatchOut{}, err
	}
	// The API does not say which seat the token belongs to; infer it from
	// the submitted flag when only one seat has a pending script.
	seat := ""
	if a, b := r.Match.Seats.A, r.Match.Seats.B; a != nil && b != nil && a.Submitted != b.Submitted {
		seat = map[bool]string{true: "a", false: "b"}[a.Submitted]
	}
	return textResult("Script accepted.\n\n" + matchText(r.Match, seat)), r, nil
}

func (h *handler) getMatch(ctx context.Context, _ *mcp.CallToolRequest, in MatchIn) (*mcp.CallToolResult, MatchOut, error) {
	var r MatchOut
	if err := h.api.doJSON(ctx, "GET", matchPath(in.MatchID), "", nil, &r.Match); err != nil {
		return nil, MatchOut{}, err
	}
	return textResult(matchText(r.Match, "")), r, nil
}

func (h *handler) roundReplay(ctx context.Context, _ *mcp.CallToolRequest, in ReplayIn) (*mcp.CallToolResult, ReplayOut, error) {
	var v server.View
	if err := h.api.doJSON(ctx, "GET", matchPath(in.MatchID), "", nil, &v); err != nil {
		return nil, ReplayOut{}, err
	}
	if in.Round < 1 || in.Round > len(v.Rounds) {
		return nil, ReplayOut{}, fmt.Errorf("round %d has not been played; match %s has %d played round(s)", in.Round, in.MatchID, len(v.Rounds))
	}
	data, err := h.api.do(ctx, "GET", matchPath(in.MatchID, "/rounds/", strconv.Itoa(in.Round)), "", nil)
	if err != nil {
		return nil, ReplayOut{}, err
	}
	var l sim.Log
	if err := json.Unmarshal(data, &l); err != nil {
		return nil, ReplayOut{}, fmt.Errorf("unexpected round log from pf serve: %v", err)
	}
	p1 := v.Rounds[in.Round-1].P1
	p2 := map[string]string{"a": "b", "b": "a"}[p1]
	out := ReplayOut{Round: in.Round, P1Seat: p1, Summary: summarise(&l, in.IncludeChecks)}
	label := func(id string) string {
		if id == "p1" {
			return "(" + seatLabel(v, p1) + ")"
		}
		return "(" + seatLabel(v, p2) + ")"
	}
	text := fmt.Sprintf("match %s round %d: %s played p1, %s played p2\n%s", in.MatchID, in.Round, seatLabel(v, p1), seatLabel(v, p2), out.Summary.text(label))
	return textResult(text), out, nil
}

func (h *handler) registerMatchTools(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:  "create_match",
		Title: "Create a match",
		Description: "Create a best-of-3 match on the pf serve server and take seat a. Your script is your round-1 entry. " +
			"Returns a match_id to share with your opponent and your secret seat token: it is shown only once, keep it and pass it to submit_script. " +
			"Matches do not expire. Rules: first to 2 round wins; draws don't count; at most 5 rounds; sides alternate (seat a is p1 in odd rounds).",
		Annotations: mutating("Create a match"),
	}, h.createMatch)
	mcp.AddTool(s, &mcp.Tool{
		Name:  "join_match",
		Title: "Join a match",
		Description: "Join an existing match (by match_id) as seat b with your round-1 script. Round 1 is played immediately. " +
			"Returns your secret seat token: it is shown only once, keep it and pass it to submit_script.",
		Annotations: mutating("Join a match"),
	}, h.joinMatch)
	mcp.AddTool(s, &mcp.Tool{
		Name:  "submit_script",
		Title: "Submit a script for the next round",
		Description: "Submit your script for the next round of a match, using your seat token. Resubmitting the same script keeps it; you may resubmit until the round runs. " +
			"The round runs as soon as both seats have submitted. Your opponent never sees your script before the round is played.",
		Annotations: mutating("Submit a script"),
	}, h.submitScript)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_match",
		Title:       "Get match state",
		Description: "Show a match's status, wins, which seats have submitted for the next round, and every played round's result. Poll this to see when your opponent has submitted or a round has run.",
		Annotations: readOnly("Get match state", true),
	}, h.getMatch)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_round_replay",
		Title:       "Get a round replay",
		Description: "Summarise a played round of a match: which seat played p1, the result, per-fighter stats and the event log with frame numbers. Study it before revising your script.",
		Annotations: readOnly("Get a round replay", true),
	}, h.roundReplay)
}
