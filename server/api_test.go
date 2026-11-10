package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestEnvelopeMalformedJSON(t *testing.T) {
	s := newTestServer(t)
	expectError(t, do(t, s, "POST", "/validate", `{"script": `), 400, "bad_request")
	expectError(t, do(t, s, "POST", "/validate", ``), 400, "bad_request")
	expectError(t, do(t, s, "POST", "/validate", `{"script": 5}`), 400, "bad_request")
}

func TestDecodeUnknownField(t *testing.T) {
	s := newTestServer(t)
	b := expectError(t, do(t, s, "POST", "/validate", `{"script": "kick", "scirpt": "x"}`), 400, "bad_request")
	if !strings.Contains(b.Error.Message, "scirpt") {
		t.Errorf("message %q should name the unknown field", b.Error.Message)
	}
}

func TestDecodeTrailingData(t *testing.T) {
	s := newTestServer(t)
	expectError(t, do(t, s, "POST", "/validate", `{"script": "kick"} {"x": 1}`), 400, "bad_request")
}

func TestDecodeMissingField(t *testing.T) {
	s := newTestServer(t)
	expectError(t, do(t, s, "POST", "/validate", `{}`), 400, "bad_request")
}

func TestRouteUnknown(t *testing.T) {
	s := newTestServer(t)
	expectError(t, do(t, s, "GET", "/nope", ""), 404, "not_found")
	expectError(t, do(t, s, "GET", "/matches/abc/def/ghi/jkl", ""), 404, "not_found")
}

func TestRouteWrongMethod(t *testing.T) {
	s := newTestServer(t)
	rec := do(t, s, "GET", "/sim", "")
	expectError(t, rec, 405, "method_not_allowed")
	if a := rec.Header().Get("Allow"); a != "POST" {
		t.Errorf("Allow %q", a)
	}
	expectError(t, do(t, s, "DELETE", "/matches/0123456789abcdef", ""), 405, "method_not_allowed")
}

func TestDecodeOversizedBody(t *testing.T) {
	s := newTestServer(t)
	big := `{"p1": {"script": "` + strings.Repeat("#", 65<<10) + `"}}`
	expectError(t, do(t, s, "POST", "/sim", big), http.StatusRequestEntityTooLarge, "payload_too_large")
}

func TestRoutesList(t *testing.T) {
	s := newTestServer(t)
	want := "GET /healthz,GET /matches/{id},GET /matches/{id}/rounds/{n},GET /view,GET /view/matches/{id},GET /view/matches/{id}/rounds/{n},GET /view/static/{file},POST /matches,POST /matches/{id}/join,POST /matches/{id}/scripts,POST /sim,POST /validate"
	if got := strings.Join(s.Routes(), ","); got != want {
		t.Errorf("routes %s", got)
	}
}
