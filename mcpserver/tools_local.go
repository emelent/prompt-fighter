package mcpserver

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"promptfighter/docs"
	"promptfighter/sim"
	"promptfighter/sim/dsl"
)

func ptr[T any](v T) *T { return &v }

// readOnly / mutating annotations (design D2).
func readOnly(title string, openWorld bool) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{Title: title, ReadOnlyHint: true, IdempotentHint: true, DestructiveHint: ptr(false), OpenWorldHint: ptr(openWorld)}
}

func mutating(title string) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{Title: title, DestructiveHint: ptr(false), OpenWorldHint: ptr(true)}
}

func textResult(s string) *mcp.CallToolResult {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: s}}}
}

// --- validate_script ---

// ValidateIn is the input of validate_script.
type ValidateIn struct {
	Script string `json:"script" jsonschema:"the .pf fight script source to validate"`
}

// ScriptErr is one validation error.
type ScriptErr struct {
	Line    int    `json:"line"`
	Col     int    `json:"col"`
	Message string `json:"message"`
	Text    string `json:"text" jsonschema:"line:col: message, as printed by pf validate"`
}

// ValidateOut is the output of validate_script.
type ValidateOut struct {
	Valid  bool        `json:"valid"`
	Errors []ScriptErr `json:"errors"`
}

func (h *handler) validate(_ context.Context, _ *mcp.CallToolRequest, in ValidateIn) (*mcp.CallToolResult, ValidateOut, error) {
	errs := dsl.Validate(in.Script, h.cfg.Limits)
	out := ValidateOut{Valid: len(errs) == 0, Errors: []ScriptErr{}}
	for _, e := range errs {
		out.Errors = append(out.Errors, ScriptErr{Line: e.Line, Col: e.Col, Message: e.Msg, Text: e.String()})
	}
	if out.Valid {
		return textResult("ok: the script is valid"), out, nil
	}
	return textResult(scriptErrorText("the script", errs)), out, nil
}

// --- simulate_round ---

// SimulateIn is the input of simulate_round.
type SimulateIn struct {
	Script         string  `json:"script" jsonschema:"your .pf fight script"`
	OpponentScript *string `json:"opponent_script,omitempty" jsonschema:"the opponent's .pf script; give this or opponent_bot, not both"`
	OpponentBot    string  `json:"opponent_bot,omitempty" jsonschema:"name of a sample bot to fight (see list_sample_bots); give this or opponent_script"`
	Side           string  `json:"side,omitempty" jsonschema:"which side you play: p1 (default, starts at x=30) or p2 (starts at x=70)"`
	Name           string  `json:"name,omitempty" jsonschema:"your fighter's name (default: you)"`
	OpponentName   string  `json:"opponent_name,omitempty" jsonschema:"opponent's name (default: the bot name or opponent)"`
	Frames         int     `json:"frames,omitempty" jsonschema:"round length in frames, 1-1200 (default 1200, a full round)"`
	IncludeChecks  bool    `json:"include_checks,omitempty" jsonschema:"also list every sensor read in the events (verbose)"`
}

// SimulateOut is the output of simulate_round: a replay summary plus which
// side you played.
type SimulateOut struct {
	You string `json:"you" jsonschema:"the side your script played: p1 or p2"`
	Summary
}

func (h *handler) simulateRound(ctx context.Context, _ *mcp.CallToolRequest, in SimulateIn) (*mcp.CallToolResult, SimulateOut, error) {
	var zero SimulateOut
	if (in.OpponentScript != nil) == (in.OpponentBot != "") {
		return nil, zero, errors.New("give exactly one of opponent_script or opponent_bot")
	}
	side := in.Side
	if side == "" {
		side = "p1"
	}
	if side != "p1" && side != "p2" {
		return nil, zero, fmt.Errorf("side must be p1 or p2 (got %q)", in.Side)
	}
	frames := h.cfg.FrameBudget
	if in.Frames != 0 {
		if in.Frames < 1 || in.Frames > h.cfg.FrameBudget {
			return nil, zero, fmt.Errorf("frames must be between 1 and %d (got %d)", h.cfg.FrameBudget, in.Frames)
		}
		frames = in.Frames
	}
	you := sim.Entrant{Name: in.Name, Source: in.Script}
	if you.Name == "" {
		you.Name = "you"
	}
	var opp sim.Entrant
	if in.OpponentBot != "" {
		b, ok := h.bot(in.OpponentBot)
		if !ok {
			return nil, zero, fmt.Errorf("unknown sample bot %q; available bots: %s", in.OpponentBot, strings.Join(h.botNames(), ", "))
		}
		opp = sim.Entrant{Name: b.Name, Source: b.Source}
	} else {
		opp = sim.Entrant{Name: "opponent", Source: *in.OpponentScript}
	}
	if in.OpponentName != "" {
		opp.Name = in.OpponentName
	}
	var msgs []string
	if errs := dsl.Validate(you.Source, h.cfg.Limits); len(errs) > 0 {
		msgs = append(msgs, scriptErrorText("your script", errs))
	}
	if errs := dsl.Validate(opp.Source, h.cfg.Limits); len(errs) > 0 {
		msgs = append(msgs, scriptErrorText("the opponent script", errs))
	}
	if len(msgs) > 0 {
		return nil, zero, errors.New(strings.Join(msgs, "\n"))
	}
	a, b := you, opp
	if side == "p2" {
		a, b = opp, you
	}
	l, err := h.simulate(ctx, a, b, frames)
	if err != nil {
		return nil, zero, err
	}
	out := SimulateOut{You: side, Summary: summarise(l, in.IncludeChecks)}
	label := func(id string) string {
		if id == side {
			return "(you, " + you.Name + ")"
		}
		return "(opponent, " + opp.Name + ")"
	}
	return textResult(out.Summary.text(label)), out, nil
}

// --- list_sample_bots / get_player_guide ---

// BotsOut is the output of list_sample_bots.
type BotsOut struct {
	Bots []Bot `json:"bots"`
}

func (h *handler) listBots(_ context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, BotsOut, error) {
	var b strings.Builder
	for _, bot := range h.bots {
		fmt.Fprintf(&b, "- %s: %s\n", bot.Name, bot.Summary)
	}
	b.WriteString("Use a name as opponent_bot in simulate_round. Full sources are in this result's structured content and in the promptfighter://bots/<name> resources.")
	return textResult(b.String()), BotsOut{Bots: h.bots}, nil
}

// GuideOut is the output of get_player_guide.
type GuideOut struct {
	Guide string `json:"guide" jsonschema:"the player guide in Markdown"`
}

func (h *handler) playerGuide(_ context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, GuideOut, error) {
	return textResult(docs.PlayerGuide), GuideOut{Guide: docs.PlayerGuide}, nil
}

func (h *handler) registerLocalTools(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "validate_script",
		Title:       "Validate a fight script",
		Description: "Check a .pf fight script for errors without running it. Returns valid=true, or a list of errors in line:col: message form with a suggested fix. Validate before simulating or submitting.",
		Annotations: readOnly("Validate a fight script", false),
	}, h.validate)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "simulate_round",
		Title:       "Simulate a round",
		Description: "Run one round of your script against a sample bot (opponent_bot) or another script (opponent_script) on this machine; no server needed. Returns the result, per-fighter stats and the event log (hits, blocks, whiffs, throws, knockdowns with frame numbers). Use it to test and iterate before playing a match. Set include_checks to also see every sensor read.",
		Annotations: readOnly("Simulate a round", false),
	}, h.simulateRound)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_sample_bots",
		Title:       "List sample bots",
		Description: "List the bundled sample bots (sparring partners) with their strategies and full source. Use a bot's name as opponent_bot in simulate_round, or read its source for ideas.",
		Annotations: readOnly("List sample bots", false),
	}, h.listBots)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_player_guide",
		Title:       "Get the player guide",
		Description: "Return the Prompt Fighter player guide (Markdown): the .pf language, every action and sensor with its frame cost, combat rules, limits, common errors and strategy tips. Read this before writing your first script.",
		Annotations: readOnly("Get the player guide", false),
	}, h.playerGuide)
}
