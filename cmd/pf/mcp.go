package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"promptfighter/mcpserver"
)

// parseMcpFlags parses pf mcp flags. An empty --server disables match tools.
func parseMcpFlags(args []string, stderr io.Writer) (string, error) {
	fs := flag.NewFlagSet("mcp", flag.ContinueOnError)
	fs.SetOutput(stderr)
	serverURL := fs.String("server", mcpserver.DefaultServerURL, `pf serve base URL for the match tools ("" disables them)`)
	if err := fs.Parse(args); err != nil {
		return "", err
	}
	if fs.NArg() > 0 {
		return "", fmt.Errorf("unexpected arguments: %v", fs.Args())
	}
	if *serverURL == "" {
		return "", nil
	}
	u, err := url.Parse(*serverURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", fmt.Errorf("--server must be an http(s) URL such as %s (got %q)", mcpserver.DefaultServerURL, *serverURL)
	}
	return *serverURL, nil
}

// cmdMcp runs the MCP server on stdin/stdout until stdin closes. Nothing but
// MCP messages may be written to stdout.
func cmdMcp(args []string, _ io.Writer, stderr io.Writer) int {
	serverURL, err := parseMcpFlags(args, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "pf mcp: %v\nusage: pf mcp [--server URL]\n", err)
		return 2
	}
	s, err := mcpserver.New(mcpserver.Options{ServerURL: serverURL})
	if err != nil {
		fmt.Fprintf(stderr, "pf mcp: %v\n", err)
		return 1
	}
	target := serverURL
	if target == "" {
		target = "disabled"
	}
	fmt.Fprintf(stderr, "pf mcp: serving MCP on stdio (pf serve: %s)\n", target)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := s.Run(ctx, &mcp.StdioTransport{}); err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, context.Canceled) && !errors.Is(err, mcp.ErrConnectionClosed) {
		fmt.Fprintf(stderr, "pf mcp: %v\n", err)
		return 1
	}
	return 0
}
