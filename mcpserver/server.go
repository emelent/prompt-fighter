// Package mcpserver implements "pf mcp": a Model Context Protocol server that
// lets a player's LLM client validate fight scripts, spar against the sample
// bots, play matches on a "pf serve" server and study replays.
package mcpserver

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"promptfighter/scripts"
	"promptfighter/sim"
	"promptfighter/sim/dsl"
)

// Version is the pf mcp server version reported in serverInfo.
const Version = "0.3.0"

// DefaultServerURL is the pf serve address the match tools use by default.
const DefaultServerURL = "http://127.0.0.1:8080"

const (
	defaultHTTPTimeout = 30 * time.Second
	maxResponseBytes   = 4 << 20
	listTTLMs          = 3_600_000 // the catalogue never changes while the process runs
)

// Options configures the MCP server.
type Options struct {
	// ServerURL is the base URL of the pf serve HTTP API used by the match
	// tools. Empty disables them.
	ServerURL string
	// HTTPClient overrides the client used for pf serve (tests). Defaults to
	// a client with a 30s timeout.
	HTTPClient *http.Client
	// MaxSims caps concurrent local simulations; default runtime.NumCPU().
	MaxSims int
}

const instructions = `Prompt Fighter is a fighting game where you write the fighter's AI as a small script (the .pf language) instead of pressing buttons.
Start with get_player_guide: it teaches the language, frame costs and combat rules.
Loop: write a script -> validate_script -> simulate_round against a sample bot (list_sample_bots) -> read the replay summary -> revise.
To play a real best-of-3 match against another player, use create_match / join_match, then submit_script before each round and get_match / get_round_replay to follow along. Keep the seat token you receive; it is shown only once.`

// Bot is a bundled sample script.
type Bot struct {
	Name    string `json:"name" jsonschema:"bot name, usable as opponent_bot"`
	Summary string `json:"summary" jsonschema:"the bot's strategy, from its header comments"`
	Source  string `json:"source" jsonschema:"full .pf source of the bot"`
}

type handler struct {
	cfg  sim.Config
	opts Options
	bots []Bot
	sims chan struct{}
	api  *apiClient
}

// New builds the MCP server with all tools and resources registered.
func New(opts Options) (*mcp.Server, error) {
	if opts.MaxSims <= 0 {
		opts.MaxSims = runtime.NumCPU()
	}
	if opts.HTTPClient == nil {
		opts.HTTPClient = &http.Client{Timeout: defaultHTTPTimeout}
	}
	h := &handler{cfg: sim.DefaultConfig(), opts: opts, sims: make(chan struct{}, opts.MaxSims)}
	bots, err := loadBots(h.cfg)
	if err != nil {
		return nil, err
	}
	h.bots = bots
	if opts.ServerURL != "" {
		h.api = &apiClient{base: strings.TrimRight(opts.ServerURL, "/"), http: opts.HTTPClient}
	}
	s := mcp.NewServer(&mcp.Implementation{Name: "prompt-fighter", Title: "Prompt Fighter", Version: Version}, &mcp.ServerOptions{
		Instructions: instructions,
		SetCacheable: func(_ context.Context, _ mcp.Request, c *mcp.Cacheable) {
			if c.TTLMs == 0 {
				c.TTLMs = listTTLMs
			}
			c.CacheScope = "public"
		},
	})
	h.registerLocalTools(s)
	h.registerMatchTools(s)
	h.registerResources(s)
	return s, nil
}

// loadBots reads the embedded sample bots, sorted by name, and validates them.
func loadBots(cfg sim.Config) ([]Bot, error) {
	names, err := fs.Glob(scripts.FS, "*.pf")
	if err != nil {
		return nil, err
	}
	sort.Strings(names)
	var bots []Bot
	for _, n := range names {
		b, err := fs.ReadFile(scripts.FS, n)
		if err != nil {
			return nil, err
		}
		src := string(b)
		if errs := dsl.Validate(src, cfg.Limits); len(errs) > 0 {
			return nil, fmt.Errorf("sample bot %s is invalid: %s", n, errs[0])
		}
		bots = append(bots, Bot{Name: strings.TrimSuffix(n, ".pf"), Summary: botSummary(src), Source: src})
	}
	return bots, nil
}

// botSummary joins the leading comment block, without the "#" markers.
func botSummary(src string) string {
	var parts []string
	for _, line := range strings.Split(src, "\n") {
		t := strings.TrimSpace(line)
		if !strings.HasPrefix(t, "#") {
			break
		}
		if t = strings.TrimSpace(strings.TrimLeft(t, "#")); t != "" {
			parts = append(parts, t)
		}
	}
	return strings.Join(parts, " ")
}

func (h *handler) bot(name string) (Bot, bool) {
	for _, b := range h.bots {
		if b.Name == name {
			return b, true
		}
	}
	return Bot{}, false
}

func (h *handler) botNames() []string {
	var out []string
	for _, b := range h.bots {
		out = append(out, b.Name)
	}
	return out
}

// simulate runs one round, holding a simulation slot.
func (h *handler) simulate(ctx context.Context, a, b sim.Entrant, frames int) (*sim.Log, error) {
	select {
	case h.sims <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	defer func() { <-h.sims }()
	cfg := h.cfg
	cfg.FrameBudget = frames
	return sim.Simulate(cfg, a, b)
}

// scriptErrorText renders validation errors one per line, like pf validate.
func scriptErrorText(label string, errs []dsl.Error) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s is invalid:", label)
	for _, e := range errs {
		b.WriteString("\n" + e.String())
	}
	return b.String()
}
