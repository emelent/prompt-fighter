package mcpserver

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func readResource(t *testing.T, cs *mcp.ClientSession, uri string) (*mcp.ReadResourceResult, error) {
	t.Helper()
	return cs.ReadResource(context.Background(), &mcp.ReadResourceParams{URI: uri})
}

func TestResourceList(t *testing.T) {
	cs := connect(t, Options{})
	res, err := cs.ListResources(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	var uris []string
	for _, r := range res.Resources {
		uris = append(uris, r.URI)
		if r.MIMEType == "" || r.Description == "" {
			t.Errorf("%s: missing mime type or description", r.URI)
		}
	}
	sort.Strings(uris)
	want := "promptfighter://bots/ducker,promptfighter://bots/jumper,promptfighter://bots/lazy,promptfighter://bots/rushdown,promptfighter://bots/thrower,promptfighter://bots/turtle,promptfighter://docs/player-guide,promptfighter://docs/rules"
	if got := strings.Join(uris, ","); got != want {
		t.Errorf("resources %s", got)
	}
	if res.TTLMs <= 0 {
		t.Errorf("ttl %d", res.TTLMs)
	}
}

func TestResourceGuideMatchesTool(t *testing.T) {
	cs := connect(t, Options{})
	r, err := readResource(t, cs, guideURI)
	if err != nil {
		t.Fatal(err)
	}
	var out GuideOut
	structured(t, callTool(t, cs, "get_player_guide", map[string]any{}), &out)
	if len(r.Contents) != 1 || r.Contents[0].Text != out.Guide || r.Contents[0].MIMEType != "text/markdown" {
		t.Errorf("guide resource differs from get_player_guide")
	}
	rules, err := readResource(t, cs, rulesURI)
	want, _ := os.ReadFile(filepath.Join("..", "docs", "RULES.md"))
	if err != nil || rules.Contents[0].Text != string(want) {
		t.Errorf("rules resource: %v", err)
	}
}

func TestResourceBot(t *testing.T) {
	cs := connect(t, Options{})
	r, err := readResource(t, cs, "promptfighter://bots/turtle")
	want, _ := os.ReadFile(filepath.Join("..", "scripts", "turtle.pf"))
	if err != nil || len(r.Contents) != 1 || r.Contents[0].Text != string(want) || r.Contents[0].MIMEType != "text/plain" {
		t.Errorf("bot resource: %v", err)
	}
}

func TestResourceUnknown(t *testing.T) {
	cs := connect(t, Options{})
	if _, err := readResource(t, cs, "promptfighter://bots/ninja"); err == nil {
		t.Fatal("expected a resource-not-found error")
	} else if !strings.Contains(strings.ToLower(err.Error()), "not found") {
		t.Errorf("error %v", err)
	}
}
