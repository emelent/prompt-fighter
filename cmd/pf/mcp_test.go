package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func buildPF(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "pf")
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	return bin
}

// mcpProc is a pf mcp subprocess driven over its stdin/stdout.
type mcpProc struct {
	t      *testing.T
	cmd    *exec.Cmd
	in     io.WriteCloser
	out    *bufio.Scanner
	stderr *bytes.Buffer
	lines  []string
}

func startMcp(t *testing.T, bin string, args ...string) *mcpProc {
	t.Helper()
	cmd := exec.Command(bin, append([]string{"mcp"}, args...)...)
	in, _ := cmd.StdinPipe()
	out, _ := cmd.StdoutPipe()
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	sc := bufio.NewScanner(out)
	sc.Buffer(make([]byte, 1<<20), 8<<20)
	p := &mcpProc{t: t, cmd: cmd, in: in, out: sc, stderr: &stderr}
	t.Cleanup(func() { cmd.Process.Kill(); cmd.Wait() })
	return p
}

func (p *mcpProc) send(msg string) { p.in.Write([]byte(msg + "\n")) }

// recv reads stdout lines until the response with the given id.
func (p *mcpProc) recv(id int) map[string]any {
	p.t.Helper()
	for p.out.Scan() {
		line := p.out.Text()
		p.lines = append(p.lines, line)
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			p.t.Fatalf("stdout line is not JSON: %q", line)
		}
		if v, ok := m["id"].(float64); ok && int(v) == id {
			return m
		}
	}
	p.t.Fatalf("no response with id %d (stderr: %s)", id, p.stderr.String())
	return nil
}

// closeAndWait closes stdin and returns the exit code; remaining stdout lines
// must also be valid JSON.
func (p *mcpProc) closeAndWait() int {
	p.t.Helper()
	p.in.Close()
	for p.out.Scan() {
		line := p.out.Text()
		if !json.Valid([]byte(line)) {
			p.t.Errorf("stdout line is not JSON: %q", line)
		}
	}
	done := make(chan error, 1)
	go func() { done <- p.cmd.Wait() }()
	select {
	case err := <-done:
		if ee, ok := err.(*exec.ExitError); ok {
			return ee.ExitCode()
		}
		if err != nil {
			p.t.Fatal(err)
		}
		return 0
	case <-time.After(10 * time.Second):
		p.t.Fatal("pf mcp did not exit after stdin closed")
		return -1
	}
}

const meta2026 = `"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28","io.modelcontextprotocol/clientInfo":{"name":"wire-test","version":"1"},"io.modelcontextprotocol/clientCapabilities":{}}`

func result(t *testing.T, m map[string]any) map[string]any {
	t.Helper()
	r, ok := m["result"].(map[string]any)
	if !ok {
		t.Fatalf("no result: %v", m)
	}
	return r
}

func TestMcpWireDiscover2026(t *testing.T) {
	p := startMcp(t, buildPF(t), "--server", "")
	p.send(`{"jsonrpc":"2.0","id":1,"method":"server/discover","params":{` + meta2026 + `}}`)
	r := result(t, p.recv(1))
	caps, _ := r["capabilities"].(map[string]any)
	versions, _ := json.Marshal(r["supportedVersions"])
	if r["resultType"] != "complete" || !strings.Contains(string(versions), `"2026-07-28"`) || caps["tools"] == nil || caps["resources"] == nil || r["instructions"] == "" || r["instructions"] == nil {
		t.Errorf("discover result %v", r)
	}
	p.send(`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{` + meta2026 + `}}`)
	r = result(t, p.recv(2))
	if tools, _ := r["tools"].([]any); len(tools) != 9 || r["resultType"] != "complete" || r["ttlMs"] == float64(0) {
		t.Errorf("tools/list %v", r)
	}
	// A failing tool call (match tools disabled) must not pollute stdout.
	p.send(`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"get_match","arguments":{"match_id":"0123456789abcdef"},` + meta2026 + `}}`)
	if r = result(t, p.recv(3)); r["isError"] != true {
		t.Errorf("expected isError: %v", r)
	}
	if code := p.closeAndWait(); code != 0 {
		t.Errorf("exit %d", code)
	}
	if !strings.HasPrefix(p.stderr.String(), "pf mcp: serving MCP on stdio (pf serve: disabled)") {
		t.Errorf("stderr %q", p.stderr.String())
	}
}

func TestMcpWireLegacyInitialize(t *testing.T) {
	p := startMcp(t, buildPF(t))
	p.send(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"legacy","version":"1"}}}`)
	r := result(t, p.recv(1))
	info, _ := r["serverInfo"].(map[string]any)
	if r["protocolVersion"] != "2025-11-25" || info["name"] != "prompt-fighter" {
		t.Errorf("initialize %v", r)
	}
	p.send(`{"jsonrpc":"2.0","method":"notifications/initialized","params":{}}`)
	p.send(`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`)
	if tools, _ := result(t, p.recv(2))["tools"].([]any); len(tools) != 9 {
		t.Errorf("legacy tools/list returned %d tools", len(tools))
	}
	if code := p.closeAndWait(); code != 0 {
		t.Errorf("exit %d", code)
	}
}

func TestMcpFlags(t *testing.T) {
	if u, err := parseMcpFlags(nil, io.Discard); err != nil || u != "http://127.0.0.1:8080" {
		t.Errorf("default %q %v", u, err)
	}
	if u, err := parseMcpFlags([]string{"--server", ""}, io.Discard); err != nil || u != "" {
		t.Errorf("disabled %q %v", u, err)
	}
	for _, bad := range []string{"://bad", "ftp://x", "localhost:8080"} {
		var stderr bytes.Buffer
		if code := cmdMcp([]string{"--server", bad}, io.Discard, &stderr); code != 2 || stderr.Len() == 0 {
			t.Errorf("--server %q: code %d stderr %q", bad, code, stderr.String())
		}
	}
	if code := cmdMcp([]string{"--bogus"}, io.Discard, io.Discard); code != 2 {
		t.Errorf("bogus flag code %d", code)
	}
}
