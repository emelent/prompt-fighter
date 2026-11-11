package mcpserver

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"promptfighter/sim"
)

var allTools = ToolNames()

func TestToolsCatalogue(t *testing.T) {
	cs := connect(t, Options{ServerURL: DefaultServerURL})
	var orders [2]string
	for i := range orders {
		res, err := cs.ListTools(context.Background(), nil)
		if err != nil {
			t.Fatal(err)
		}
		if res.TTLMs <= 0 || res.CacheScope != "public" {
			t.Errorf("cacheable ttl %d scope %q", res.TTLMs, res.CacheScope)
		}
		var names []string
		for _, tool := range res.Tools {
			names = append(names, tool.Name)
			if tool.Title == "" || len(tool.Description) < 40 || tool.InputSchema == nil || tool.OutputSchema == nil || tool.Annotations == nil {
				t.Errorf("%s: incomplete definition", tool.Name)
				continue
			}
			a := tool.Annotations
			if a.DestructiveHint == nil || *a.DestructiveHint {
				t.Errorf("%s: destructiveHint must be false", tool.Name)
			}
			wantRO := map[string]bool{"validate_script": true, "simulate_round": true, "list_sample_bots": true, "get_player_guide": true, "get_match": true, "get_round_replay": true}[tool.Name]
			wantOW := map[string]bool{"create_match": true, "join_match": true, "submit_script": true, "get_match": true, "get_round_replay": true}[tool.Name]
			if a.ReadOnlyHint != wantRO || a.OpenWorldHint == nil || *a.OpenWorldHint != wantOW {
				t.Errorf("%s: readOnly %v openWorld %v", tool.Name, a.ReadOnlyHint, *a.OpenWorldHint)
			}
			b, _ := json.Marshal(tool.InputSchema)
			if !strings.Contains(string(b), `"additionalProperties":false`) {
				t.Errorf("%s: input schema does not reject unknown properties: %s", tool.Name, b)
			}
		}
		orders[i] = strings.Join(names, ",")
	}
	if orders[0] != strings.Join(allTools, ",") || orders[0] != orders[1] {
		t.Errorf("tools %q / %q", orders[0], orders[1])
	}
}

func TestToolsUnknownArgumentRejected(t *testing.T) {
	cs := connect(t, Options{})
	res := callTool(t, cs, "validate_script", map[string]any{"script": "kick", "scirpt": "x"})
	if !res.IsError || res.StructuredContent != nil {
		t.Errorf("unknown argument must fail: %s", resultText(res))
	}
}

func TestLocalValidate(t *testing.T) {
	cs := connect(t, Options{})
	var out ValidateOut
	structured(t, callTool(t, cs, "validate_script", map[string]any{"script": "loop { kick }"}), &out)
	if !out.Valid || out.Errors == nil || len(out.Errors) != 0 {
		t.Errorf("valid: %+v", out)
	}
	res := callTool(t, cs, "validate_script", map[string]any{"script": "punch\n\n    kik"})
	structured(t, res, &out)
	if out.Valid || len(out.Errors) != 1 || out.Errors[0].Text != `3:5: unknown action "kik" — did you mean "kick"?` {
		t.Errorf("invalid: %+v", out)
	}
	if !strings.Contains(resultText(res), `3:5: unknown action "kik" — did you mean "kick"?`) {
		t.Errorf("text %q", resultText(res))
	}
}

func TestLocalListBots(t *testing.T) {
	cs := connect(t, Options{})
	var out BotsOut
	res := callTool(t, cs, "list_sample_bots", map[string]any{})
	structured(t, res, &out)
	var names []string
	for _, b := range out.Bots {
		names = append(names, b.Name)
		src, _ := os.ReadFile(filepath.Join("..", "scripts", b.Name+".pf"))
		if b.Source != string(src) || b.Summary == "" {
			t.Errorf("bot %s source/summary mismatch", b.Name)
		}
	}
	if strings.Join(names, ",") != "ducker,jumper,lazy,rushdown,thrower,turtle" {
		t.Errorf("bots %v", names)
	}
	if !strings.Contains(resultText(res), "- turtle: ") {
		t.Errorf("text %q", resultText(res))
	}
}

func TestLocalSimulateAgainstBot(t *testing.T) {
	cs := connect(t, Options{})
	mine := "loop { move_forward x3 kick }"
	var out SimulateOut
	res := callTool(t, cs, "simulate_round", map[string]any{"script": mine, "opponent_bot": "turtle"})
	structured(t, res, &out)
	turtle, _ := os.ReadFile(filepath.Join("..", "scripts", "turtle.pf"))
	l, _ := sim.Simulate(sim.DefaultConfig(), sim.Entrant{Name: "you", Source: mine}, sim.Entrant{Name: "turtle", Source: string(turtle)})
	if out.You != "p1" || out.Players[0].Name != "you" || out.Players[1].Name != "turtle" {
		t.Errorf("players %+v you %s", out.Players, out.You)
	}
	if out.Result.Reason != l.Result.Reason || out.Result.HP != l.Result.HP || out.Result.EndFrame != l.Result.EndFrame {
		t.Errorf("result %+v vs engine %+v", out.Result, l.Result)
	}
	if strings.Join(out.Events, "\n") != strings.Join(l.EventLines(false), "\n") {
		t.Errorf("events differ from pf sim lines")
	}
	if txt := resultText(res); !strings.HasPrefix(txt, l.ResultLine()) || !strings.Contains(txt, "p1 (you, you):") || !strings.Contains(txt, "p2 (opponent, turtle):") {
		t.Errorf("text:\n%s", txt)
	}
}

func TestLocalSimulateSideP2(t *testing.T) {
	cs := connect(t, Options{})
	var out SimulateOut
	structured(t, callTool(t, cs, "simulate_round", map[string]any{"script": "wait 1", "opponent_script": "", "side": "p2", "name": "me", "frames": 50, "include_checks": true}), &out)
	if out.You != "p2" || out.Players[1].Name != "me" || out.Players[0].Name != "opponent" || out.Result.EndFrame != 49 {
		t.Errorf("out %+v", out)
	}
}

func TestLocalSimulateErrors(t *testing.T) {
	cs := connect(t, Options{})
	expectToolError(t, callTool(t, cs, "simulate_round", map[string]any{"script": "kick"}), "exactly one of opponent_script or opponent_bot")
	expectToolError(t, callTool(t, cs, "simulate_round", map[string]any{"script": "kick", "opponent_script": "kick", "opponent_bot": "turtle"}), "exactly one")
	expectToolError(t, callTool(t, cs, "simulate_round", map[string]any{"script": "kick", "opponent_bot": "ninja"}), `unknown sample bot "ninja"`, "ducker, jumper, lazy, rushdown, thrower, turtle")
	expectToolError(t, callTool(t, cs, "simulate_round", map[string]any{"script": "punch\n\n    kik", "opponent_bot": "lazy"}), `3:5: unknown action "kik" — did you mean "kick"?`, "your script")
	expectToolError(t, callTool(t, cs, "simulate_round", map[string]any{"script": "kick", "opponent_script": "kik"}), "the opponent script")
	expectToolError(t, callTool(t, cs, "simulate_round", map[string]any{"script": "kick", "opponent_bot": "lazy", "side": "left"}), "side must be p1 or p2")
	expectToolError(t, callTool(t, cs, "simulate_round", map[string]any{"script": "kick", "opponent_bot": "lazy", "frames": 601}), "frames must be between 1 and 600")
}

func TestLocalPlayerGuide(t *testing.T) {
	cs := connect(t, Options{})
	var out GuideOut
	res := callTool(t, cs, "get_player_guide", map[string]any{})
	structured(t, res, &out)
	guide, _ := os.ReadFile(filepath.Join("..", "docs", "PLAYER_GUIDE.md"))
	if out.Guide != string(guide) || resultText(res) != string(guide) {
		t.Errorf("guide mismatch")
	}
}

func TestToolsDiscover(t *testing.T) {
	cs := connect(t, Options{})
	ir := cs.InitializeResult()
	if ir == nil || ir.ServerInfo == nil || ir.ServerInfo.Name != "prompt-fighter" || ir.Instructions == "" || ir.Capabilities.Tools == nil || ir.Capabilities.Resources == nil {
		t.Fatalf("server info %+v", ir)
	}
	if ir.ProtocolVersion != "2026-07-28" {
		t.Errorf("negotiated %q, want 2026-07-28", ir.ProtocolVersion)
	}
}
