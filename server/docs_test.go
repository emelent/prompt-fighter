package server

import (
	"os"
	"strings"
	"testing"
)

// TestDocsListEveryRoute keeps docs/API.md's endpoint table in sync with the
// routes the server registers.
func TestDocsListEveryRoute(t *testing.T) {
	b, err := os.ReadFile("../docs/API.md")
	if err != nil {
		t.Fatal(err)
	}
	doc := string(b)
	s := newTestServer(t)
	for _, r := range s.Routes() {
		method, path, _ := strings.Cut(r, " ")
		if row := "| " + method + " | `" + path + "` |"; !strings.Contains(doc, row) {
			t.Errorf("docs/API.md endpoint table is missing %q", row)
		}
	}
	for _, code := range []string{"bad_request", "unauthorized", "not_found", "method_not_allowed", "conflict", "payload_too_large", "invalid_script", "internal", "busy"} {
		if !strings.Contains(doc, "| `"+code+"` |") {
			t.Errorf("docs/API.md error table is missing %s", code)
		}
	}
}
