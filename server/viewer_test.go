package server

import (
	"encoding/json"
	"io/fs"
	"regexp"
	"strings"
	"testing"

	"promptfighter/web"
)

func checkViewerHeaders(t *testing.T, path string, h map[string][]string) {
	t.Helper()
	get := func(k string) string {
		if v := h[k]; len(v) > 0 {
			return v[0]
		}
		return ""
	}
	csp := get("Content-Security-Policy")
	for _, d := range []string{"default-src 'none'", "script-src 'self'", "style-src 'self'", "media-src 'self'", "connect-src 'self'", "base-uri 'none'", "frame-ancestors 'none'"} {
		if !strings.Contains(csp, d) {
			t.Errorf("%s: CSP %q lacks %q", path, csp, d)
		}
	}
	if strings.Contains(csp, "unsafe-inline") || strings.Contains(csp, "unsafe-eval") {
		t.Errorf("%s: CSP allows unsafe sources: %q", path, csp)
	}
	if get("X-Content-Type-Options") != "nosniff" || get("Referrer-Policy") != "no-referrer" {
		t.Errorf("%s: headers %v", path, h)
	}
}

func TestViewPagesAreHTML(t *testing.T) {
	s := newTestServer(t)
	for _, p := range []string{"/view", "/view/matches/0123456789abcdef", "/view/matches/0123456789abcdef/rounds/2", "/watch/matches/0123456789abcdef", "/watch/matches/0123456789abcdef/rounds/2"} {
		rec := do(t, s, "GET", p, "")
		if rec.Code != 200 || rec.Header().Get("Content-Type") != "text/html; charset=utf-8" {
			t.Errorf("%s: %d %q", p, rec.Code, rec.Header().Get("Content-Type"))
		}
		if json.Valid(rec.Body.Bytes()) || !strings.Contains(rec.Body.String(), "<!doctype html>") {
			t.Errorf("%s: body is not the viewer HTML", p)
		}
		checkViewerHeaders(t, p, rec.Header())
	}
}

var srcRef = regexp.MustCompile(`(?:src|href)="([^"]+)"`)

func TestViewAssetsAndSameOrigin(t *testing.T) {
	s := newTestServer(t)
	page := do(t, s, "GET", "/view/matches/0123456789abcdef/rounds/2", "").Body.String()
	var js, css int
	for _, m := range srcRef.FindAllStringSubmatch(page, -1) {
		ref := m[1]
		if ref == "data:," {
			continue // empty inline favicon: no request is made
		}
		if !strings.HasPrefix(ref, "/") || strings.HasPrefix(ref, "//") {
			t.Errorf("page references a non-same-origin path %q", ref)
			continue
		}
		if !strings.HasPrefix(ref, "/view/static/") {
			continue
		}
		rec := do(t, s, "GET", ref, "")
		ct := rec.Header().Get("Content-Type")
		switch {
		case strings.HasSuffix(ref, ".js"):
			js++
			if ct != "text/javascript; charset=utf-8" {
				t.Errorf("%s content type %q", ref, ct)
			}
		case strings.HasSuffix(ref, ".css"):
			css++
			if ct != "text/css; charset=utf-8" {
				t.Errorf("%s content type %q", ref, ct)
			}
		}
		if rec.Code != 200 {
			t.Errorf("%s: %d", ref, rec.Code)
		}
		checkViewerHeaders(t, ref, rec.Header())
	}
	if js == 0 || css == 0 {
		t.Errorf("page references %d js and %d css assets", js, css)
	}
	entries, _ := fs.ReadDir(web.FS, ".")
	if len(entries) < 3 {
		t.Errorf("embedded dist has %d entries", len(entries))
	}
}

func TestViewUnknownAsset(t *testing.T) {
	s := newTestServer(t)
	rec := do(t, s, "GET", "/view/static/nope.js", "")
	expectError(t, rec, 404, "not_found")
	checkViewerHeaders(t, "/view/static/nope.js", rec.Header())
	expectError(t, do(t, s, "GET", "/view/static/index.html", ""), 404, "not_found")
	expectError(t, do(t, s, "GET", "/view/static/..%2fembed.go", ""), 404, "not_found")
	expectError(t, do(t, s, "POST", "/view", ""), 405, "method_not_allowed")
}

func TestPlayPagesAreHTML(t *testing.T) {
	s := newTestServer(t)
	for _, p := range []string{"/play", "/play/matches/0123456789abcdef"} {
		rec := do(t, s, "GET", p, "")
		if rec.Code != 200 || rec.Header().Get("Content-Type") != "text/html; charset=utf-8" || !strings.Contains(rec.Body.String(), "<!doctype html>") {
			t.Errorf("%s: %d %q", p, rec.Code, rec.Header().Get("Content-Type"))
		}
		checkViewerHeaders(t, p, rec.Header())
	}
}

func TestRootRedirectsToLobby(t *testing.T) {
	s := newTestServer(t)
	rec := do(t, s, "GET", "/", "")
	if rec.Code != 302 || rec.Header().Get("Location") != "/play" {
		t.Errorf("GET /: %d Location %q", rec.Code, rec.Header().Get("Location"))
	}
	checkViewerHeaders(t, "/", rec.Header())
	expectError(t, do(t, s, "POST", "/", ""), 405, "method_not_allowed")
	expectError(t, do(t, s, "GET", "/nope", ""), 404, "not_found")
	expectError(t, do(t, s, "GET", "/play/nope", ""), 404, "not_found")
}

func TestViewAudioAssets(t *testing.T) {
	s := newTestServer(t)
	entries, _ := fs.ReadDir(web.FS, ".")
	n := 0
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".mp3") {
			continue
		}
		n++
		rec := do(t, s, "GET", "/view/static/"+e.Name(), "")
		if rec.Code != 200 || rec.Header().Get("Content-Type") != "audio/mpeg" {
			t.Errorf("%s: %d %q, want 200 audio/mpeg", e.Name(), rec.Code, rec.Header().Get("Content-Type"))
		}
		checkViewerHeaders(t, e.Name(), rec.Header())
	}
	if n < 7 {
		t.Errorf("embedded %d mp3 files, want at least 7 (six effects and the fight music)", n)
	}
}
