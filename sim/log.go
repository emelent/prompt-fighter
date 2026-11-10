package sim

import (
	"bytes"
	"encoding/json"
	"strconv"
)

// FormatVersion is the event-log schema version.
const FormatVersion = "1"

// Log is the complete record of one simulated round.
type Log struct {
	FormatVersion string    `json:"format_version"`
	ConfigVersion string    `json:"config_version"`
	FrameBudget   int       `json:"frame_budget"`
	Players       [2]Player `json:"players"`
	Result        Result    `json:"result"`
	Frames        []Frame   `json:"frames"`
	Events        []Event   `json:"events"`
}

// Player identifies a side and its script.
type Player struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Script string `json:"script"`
}

// Result is the round outcome.
type Result struct {
	Winner   *string `json:"winner"` // "p1", "p2" or null for a draw
	Reason   string  `json:"reason"` // ko, double_ko, timeout
	EndFrame int     `json:"end_frame"`
	HP       [2]int  `json:"hp"`
}

// Frame is the state of both fighters at the end of one tick.
type Frame struct {
	F  int          `json:"f"`
	P1 FighterFrame `json:"p1"`
	P2 FighterFrame `json:"p2"`
}

// FighterFrame is everything a renderer needs for one fighter on one frame.
type FighterFrame struct {
	X            int    `json:"x"`
	Y            int    `json:"y"`
	Facing       int    `json:"facing"`
	State        string `json:"state"`
	Anim         string `json:"anim"`
	AnimFrame    int    `json:"anim_frame"`
	Phase        string `json:"phase"`
	HP           int    `json:"hp"`
	Invulnerable bool   `json:"invulnerable"`
	Combo        int    `json:"combo"`
	ScriptLine   int    `json:"script_line"`
}

// Event is a discrete occurrence in the round.
type Event struct {
	Frame  int    `json:"frame"`
	Type   string `json:"type"`
	Actor  string `json:"actor"`
	Target string `json:"target,omitempty"`
	Detail Detail `json:"detail"`
}

// Detail carries type-specific event fields (see docs/RULES.md). Pointer and
// raw fields are used where zero/false values are meaningful.
type Detail struct {
	Attack  string          `json:"attack,omitempty"`
	Damage  *int            `json:"damage,omitempty"`
	Combo   int             `json:"combo,omitempty"`
	HP      *int            `json:"hp,omitempty"`
	Reason  string          `json:"reason,omitempty"`
	Outcome string          `json:"outcome,omitempty"`
	Sensor  string          `json:"sensor,omitempty"`
	Value   json.RawMessage `json:"value,omitempty"`
	Line    int             `json:"line,omitempty"`
	Cost    int             `json:"cost,omitempty"`
	Handler string          `json:"handler,omitempty"`
	Attacks []string        `json:"attacks,omitempty"`
	Damages []int           `json:"damages,omitempty"`
	Boost   int             `json:"boost,omitempty"`   // hit/throw: damage percent from the target's passivity, when above 100
	Repeats int             `json:"repeats,omitempty"` // stale: starts of this attack in a row
	Range   *int            `json:"range,omitempty"`   // stale: the reduced range
}

func intp(v int) *int { return &v }

func boolRaw(b bool) json.RawMessage {
	if b {
		return json.RawMessage("true")
	}
	return json.RawMessage("false")
}

func intRaw(v int) json.RawMessage { return json.RawMessage(strconv.Itoa(v)) }

func encode(v any, indent bool) []byte {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if indent {
		enc.SetIndent("", "  ")
	}
	if err := enc.Encode(v); err != nil {
		panic(err) // all types are plain data; cannot fail
	}
	return buf.Bytes()
}

// JSON returns the compact canonical encoding of the log (newline-terminated).
func (l *Log) JSON() []byte { return encode(l, false) }

// IndentJSON returns an indented encoding, used for golden files.
func (l *Log) IndentJSON() []byte { return encode(l, true) }
