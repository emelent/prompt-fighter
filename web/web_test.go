package web

import (
	"io/fs"
	"os"
	"os/exec"
	"regexp"
	"testing"
)

var assetRef = regexp.MustCompile(`(?:src|href)="([^"]+)"`)

// TestDistSelfConsistent checks the committed build: index.html exists and
// every asset it references is a flat /view/static/ path that is embedded.
func TestDistSelfConsistent(t *testing.T) {
	b, err := fs.ReadFile(FS, "index.html")
	if err != nil {
		t.Fatalf("dist/index.html missing (run `npm run build` in web/): %v", err)
	}
	refs := assetRef.FindAllStringSubmatch(string(b), -1)
	if len(refs) == 0 {
		t.Fatal("index.html references no assets")
	}
	for _, m := range refs {
		ref := m[1]
		if ref == "/view" || ref[0] == '#' || ref == "data:," {
			continue // in-app links and the empty inline favicon
		}
		name, ok := cutPrefix(ref, "/view/static/")
		if !ok {
			t.Errorf("index.html references %q, which is not a /view/static/ asset", ref)
			continue
		}
		if _, err := fs.Stat(FS, name); err != nil {
			t.Errorf("index.html references %q, which is not embedded", ref)
		}
	}
}

func cutPrefix(s, p string) (string, bool) {
	if len(s) >= len(p) && s[:len(p)] == p {
		return s[len(p):], true
	}
	return "", false
}

// TestFrontend runs the TypeScript checks and Vitest suites when the npm
// dependencies are installed (cd web && npm ci); otherwise it is skipped.
func TestFrontend(t *testing.T) {
	if testing.Short() {
		t.Skip("frontend tests skipped in -short mode")
	}
	if _, err := os.Stat("node_modules"); err != nil {
		t.Skip("web/node_modules not installed; run `npm ci` in web/ to enable the frontend tests")
	}
	if _, err := exec.LookPath("npm"); err != nil {
		t.Skip("npm not on PATH")
	}
	cmd := exec.Command("npm", "test", "--silent")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("npm test failed: %v\n%s", err, out)
	}
}
