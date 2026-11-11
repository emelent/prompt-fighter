package mcpserver

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"
)

var (
	bareMatchID = regexp.MustCompile(`^[0-9a-f]{16}$`)
	linkMatchID = regexp.MustCompile(`/(?:play|watch|view)/matches/([0-9a-f]{16})(?:[/?#]|$)`)
)

// parseMatchRef resolves a match reference that a model was given: a bare
// 16-character match id, or a game link such as
// http://host:8080/play/matches/<id>. For a link it also returns the link's
// origin, the pf serve server the match lives on; for a bare id the origin is
// "" and the caller uses its configured server.
func parseMatchRef(ref string) (id, origin string, err error) {
	ref = strings.TrimSpace(ref)
	if bareMatchID.MatchString(strings.ToLower(ref)) {
		return strings.ToLower(ref), "", nil
	}
	u, perr := url.Parse(ref)
	if perr != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return "", "", fmt.Errorf("match_id must be a 16-character match id or a game link like http://host:8080/play/matches/<id> (got %q)", ref)
	}
	m := linkMatchID.FindStringSubmatch(strings.ToLower(u.Path))
	if m == nil {
		return "", "", fmt.Errorf("this link has no match id in it; a game link looks like http://host:8080/play/matches/<16-character id> (got %q)", ref)
	}
	return m[1], u.Scheme + "://" + u.Host, nil
}

// sameOrigin compares scheme, host and port, treating the loopback names
// localhost, 127.0.0.1 and ::1 as the same host.
func sameOrigin(a, b *url.URL) bool {
	if !strings.EqualFold(a.Scheme, b.Scheme) {
		return false
	}
	ha, pa := hostPort(a)
	hb, pb := hostPort(b)
	return pa == pb && (strings.EqualFold(ha, hb) || (loopback(ha) && loopback(hb)))
}

func hostPort(u *url.URL) (host, port string) {
	host, port = u.Hostname(), u.Port()
	if port == "" {
		port = map[string]string{"http": "80", "https": "443"}[strings.ToLower(u.Scheme)]
	}
	return host, port
}

func loopback(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
