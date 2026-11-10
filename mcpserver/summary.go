package mcpserver

import (
	"fmt"
	"strings"

	"promptfighter/sim"
)

// maxEventLines caps the event lines in a replay summary.
const maxEventLines = 200

// PlayerRef names one side of a round.
type PlayerRef struct {
	ID   string `json:"id" jsonschema:"p1 or p2"`
	Name string `json:"name"`
}

// ResultInfo is a round result.
type ResultInfo struct {
	Winner   *string `json:"winner" jsonschema:"p1, p2, or null for a draw"`
	Reason   string  `json:"reason" jsonschema:"ko, double_ko or timeout"`
	EndFrame int     `json:"end_frame"`
	HP       [2]int  `json:"hp" jsonschema:"final HP of p1 and p2"`
}

// Stats are per-fighter totals over a round.
type Stats struct {
	Hits        int `json:"hits" jsonschema:"attacks that hit the opponent"`
	DamageDealt int `json:"damage_dealt" jsonschema:"damage dealt by hits and throws"`
	Blocks      int `json:"blocks" jsonschema:"opponent attacks this fighter blocked"`
	Whiffs      int `json:"whiffs" jsonschema:"own attacks that missed"`
	Throws      int `json:"throws" jsonschema:"successful throws"`
	Knockdowns  int `json:"knockdowns" jsonschema:"times this fighter was knocked down"`
	Checks      int `json:"checks" jsonschema:"sensor reads performed"`
	MaxCombo    int `json:"max_combo" jsonschema:"longest combo landed (hits)"`
}

// PlayerStats holds stats for both sides.
type PlayerStats struct {
	P1 Stats `json:"p1"`
	P2 Stats `json:"p2"`
}

// Summary is an LLM-sized digest of a round's event log.
type Summary struct {
	Players         []PlayerRef `json:"players"`
	Result          ResultInfo  `json:"result"`
	Stats           PlayerStats `json:"stats"`
	Events          []string    `json:"events" jsonschema:"event lines exactly as pf sim prints them"`
	EventsTruncated bool        `json:"events_truncated" jsonschema:"true when only the first 200 event lines are included"`
	EventsTotal     int         `json:"events_total" jsonschema:"total number of event lines before truncation"`
	resultLine      string
}

func summarise(l *sim.Log, includeChecks bool) Summary {
	s := Summary{
		Players: []PlayerRef{{ID: "p1", Name: l.Players[0].Name}, {ID: "p2", Name: l.Players[1].Name}},
		Result:  ResultInfo{Winner: l.Result.Winner, Reason: l.Result.Reason, EndFrame: l.Result.EndFrame, HP: l.Result.HP},
	}
	stat := func(id string) *Stats {
		if id == "p1" {
			return &s.Stats.P1
		}
		return &s.Stats.P2
	}
	for _, e := range l.Events {
		d := e.Detail
		switch e.Type {
		case "hit":
			st := stat(e.Actor)
			st.Hits++
			if d.Damage != nil {
				st.DamageDealt += *d.Damage
			}
			st.MaxCombo = max(st.MaxCombo, d.Combo)
		case "block":
			stat(e.Target).Blocks++
		case "whiff":
			stat(e.Actor).Whiffs++
		case "throw":
			if d.Outcome == "success" {
				st := stat(e.Actor)
				st.Throws++
				if d.Damage != nil {
					st.DamageDealt += *d.Damage
				}
			}
		case "knockdown":
			stat(e.Actor).Knockdowns++
		case "check":
			stat(e.Actor).Checks++
		}
	}
	lines := l.EventLines(includeChecks)
	s.EventsTotal = len(lines)
	if len(lines) > maxEventLines {
		lines, s.EventsTruncated = lines[:maxEventLines], true
	}
	s.Events = lines
	s.resultLine = l.ResultLine()
	return s
}

func (st Stats) line() string {
	return fmt.Sprintf("hits %d, damage dealt %d, blocks %d, whiffs %d, throws %d, knocked down %d, checks %d, max combo %d",
		st.Hits, st.DamageDealt, st.Blocks, st.Whiffs, st.Throws, st.Knockdowns, st.Checks, st.MaxCombo)
}

// text renders the summary for a model to read. label names each side,
// e.g. "you" or "seat b (bob)".
func (s Summary) text(label func(id string) string) string {
	var b strings.Builder
	b.WriteString(s.resultLine + "\n")
	for _, p := range s.Players {
		st := s.Stats.P1
		if p.ID == "p2" {
			st = s.Stats.P2
		}
		fmt.Fprintf(&b, "%s %s: %s\n", p.ID, label(p.ID), st.line())
	}
	if s.EventsTruncated {
		fmt.Fprintf(&b, "events (first %d of %d):\n", len(s.Events), s.EventsTotal)
	} else {
		fmt.Fprintf(&b, "events (%d):\n", len(s.Events))
	}
	for _, e := range s.Events {
		b.WriteString(e + "\n")
	}
	return b.String()
}
