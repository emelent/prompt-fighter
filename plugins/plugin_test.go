// Package plugins holds the Claude Code plugins shipped with Prompt Fighter.
// This test keeps the pf plugin consistent with the marketplace manifest and
// with the MCP server it drives.
package plugins

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"promptfighter/mcpserver"
)

func readJSON(t *testing.T, path string, v any) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, v); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
}

func TestMarketplaceListsPlugin(t *testing.T) {
	var m struct {
		Name    string `json:"name"`
		Plugins []struct {
			Name, Source string
		} `json:"plugins"`
	}
	readJSON(t, "../.claude-plugin/marketplace.json", &m)
	if m.Name != "prompt-fighter" || len(m.Plugins) != 1 || m.Plugins[0].Name != "pf" || m.Plugins[0].Source != "./plugins/pf" {
		t.Fatalf("marketplace %+v", m)
	}
	var p struct{ Name string }
	readJSON(t, filepath.Join("..", m.Plugins[0].Source, ".claude-plugin", "plugin.json"), &p)
	if p.Name != "pf" {
		t.Fatalf("plugin name %q", p.Name)
	}
}

func TestPluginMCPServer(t *testing.T) {
	var m struct {
		MCPServers map[string]struct {
			Command string   `json:"command"`
			Args    []string `json:"args"`
		} `json:"mcpServers"`
	}
	readJSON(t, "pf/.mcp.json", &m)
	s, ok := m.MCPServers["prompt-fighter"]
	if !ok || s.Command != "pf" || strings.Join(s.Args, " ") != "mcp" {
		t.Fatalf("mcp config %+v", m)
	}
}

var toolRef = regexp.MustCompile(`mcp__plugin_pf_prompt-fighter__([a-z_]+)`)

func TestCommandsReferenceRealTools(t *testing.T) {
	known := map[string]bool{}
	for _, n := range mcpserver.ToolNames() {
		known[n] = true
	}
	for _, cmd := range []string{"join", "script", "submit"} {
		b, err := os.ReadFile(filepath.Join("pf", "commands", cmd+".md"))
		if err != nil {
			t.Fatal(err)
		}
		text := string(b)
		if !strings.HasPrefix(text, "---\n") || !strings.Contains(text, "\ndescription: ") {
			t.Errorf("%s.md: missing frontmatter description", cmd)
		}
		refs := toolRef.FindAllStringSubmatch(text, -1)
		if len(refs) == 0 {
			t.Errorf("%s.md: allows no MCP tools", cmd)
		}
		for _, r := range refs {
			if !known[r[1]] {
				t.Errorf("%s.md: unknown MCP tool %q", cmd, r[1])
			}
		}
		// Every tool named in backticks in the body must also be allowed.
		for _, name := range mcpserver.ToolNames() {
			if strings.Contains(text, "`"+name+"`") && !strings.Contains(text, "mcp__plugin_pf_prompt-fighter__"+name) {
				t.Errorf("%s.md: uses %s but does not allow it", cmd, name)
			}
		}
	}
}

// Only /pf:submit may submit: join and script must not even be allowed to.
func TestOnlySubmitSubmits(t *testing.T) {
	for _, cmd := range []string{"join", "script"} {
		b, err := os.ReadFile(filepath.Join("pf", "commands", cmd+".md"))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(b), "mcp__plugin_pf_prompt-fighter__submit_script") {
			t.Errorf("%s.md allows submit_script", cmd)
		}
	}
}

