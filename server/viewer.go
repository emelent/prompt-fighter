package server

import (
	"io/fs"
	"mime"
	"net/http"
	"path"

	"promptfighter/web"
)

// viewerCSP allows only same-origin scripts, styles and connections; no inline
// scripts, objects, framing or base-URI changes (replay-viewer spec).
const viewerCSP = "default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self' data: blob:; " +
	"connect-src 'self'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'"

func viewerHeaders(w http.ResponseWriter) {
	h := w.Header()
	h.Set("Content-Security-Policy", viewerCSP)
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Referrer-Policy", "no-referrer")
}

// handleViewerPage serves the viewer shell for /view, /view/matches/{id} and
// /view/matches/{id}/rounds/{n}; the page itself reports missing matches.
func (s *Server) handleViewerPage(w http.ResponseWriter, r *http.Request) *apiError {
	viewerHeaders(w)
	b, err := fs.ReadFile(web.FS, "index.html")
	if err != nil {
		return errInternal(err)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(b)
	return nil
}

// handleViewerAsset serves a built asset from the embedded dist directory.
func (s *Server) handleViewerAsset(w http.ResponseWriter, r *http.Request) *apiError {
	viewerHeaders(w)
	name := r.PathValue("file")
	if name == "" || name == "index.html" || path.Base(name) != name {
		return errNotFound("no such viewer asset: " + name)
	}
	b, err := fs.ReadFile(web.FS, name)
	if err != nil {
		return errNotFound("no such viewer asset: " + name)
	}
	ct := mime.TypeByExtension(path.Ext(name))
	if ct == "" {
		ct = "application/octet-stream"
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable") // names are content-hashed
	w.WriteHeader(http.StatusOK)
	w.Write(b)
	return nil
}
