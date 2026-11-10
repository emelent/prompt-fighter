package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"

	"promptfighter/server"
)

var errMatchToolsDisabled = errors.New(`match tools are disabled: pf mcp was started without a pf serve URL. Start a server with "pf serve" and restart pf mcp with --server http://HOST:PORT`)

// apiClient talks to the pf serve HTTP API (docs/API.md).
type apiClient struct {
	base string
	http *http.Client
}

// apiError is a non-2xx response from pf serve.
type apiError struct {
	Status  int
	Code    string
	Message string
	Errors  []server.ScriptError
}

func (e *apiError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "pf serve rejected the request (%s): %s", e.Code, e.Message)
	for _, se := range e.Errors {
		b.WriteString("\n" + se.Text)
	}
	return b.String()
}

// do sends a request and returns the raw 2xx body.
func (c *apiClient) do(ctx context.Context, method, path, token string, body any) ([]byte, error) {
	if c == nil {
		return nil, errMatchToolsDisabled
	}
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, rd)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		var ne net.Error
		if errors.As(err, &ne) && ne.Timeout() {
			return nil, fmt.Errorf("pf serve at %s did not respond in time (%v); check that the server is healthy", c.base, err)
		}
		return nil, fmt.Errorf("cannot reach pf serve at %s (%v). Start it with \"pf serve\", or restart pf mcp with --server pointing at a running server", c.base, err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("reading the response from pf serve at %s: %v", c.base, err)
	}
	if len(data) > maxResponseBytes {
		return nil, fmt.Errorf("pf serve at %s sent a response larger than %d bytes", c.base, maxResponseBytes)
	}
	if resp.StatusCode/100 != 2 {
		var env struct {
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
			Errors []server.ScriptError `json:"errors"`
		}
		if json.Unmarshal(data, &env) != nil || env.Error.Code == "" {
			return nil, fmt.Errorf("pf serve at %s returned HTTP %d: %s", c.base, resp.StatusCode, strings.TrimSpace(string(data)))
		}
		return nil, &apiError{Status: resp.StatusCode, Code: env.Error.Code, Message: env.Error.Message, Errors: env.Errors}
	}
	return data, nil
}

func (c *apiClient) doJSON(ctx context.Context, method, path, token string, body, out any) error {
	data, err := c.do(ctx, method, path, token, body)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("unexpected response from pf serve: %v", err)
	}
	return nil
}
