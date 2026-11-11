package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"promptfighter/server"
	"promptfighter/sim"
)

const (
	defaultAddr    = "127.0.0.1:8080"
	defaultDataDir = "./data"
)

type serveFlags struct {
	addr, data  string
	frames, hp int
}

func parseServeFlags(args []string, stderr io.Writer) (serveFlags, error) {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var f serveFlags
	fs.StringVar(&f.addr, "addr", defaultAddr, "listen address HOST:PORT")
	fs.StringVar(&f.data, "data", defaultDataDir, "data directory for stored matches")
	dc := sim.DefaultConfig()
	fs.IntVar(&f.frames, "frames", dc.FrameBudget, "frames per match round (>= 1)")
	fs.IntVar(&f.hp, "hp", dc.MaxHP, "starting HP of each fighter (>= 1)")
	if err := fs.Parse(args); err != nil {
		return f, err
	}
	if fs.NArg() > 0 {
		return f, fmt.Errorf("unexpected arguments: %v", fs.Args())
	}
	if f.frames < 1 {
		return f, fmt.Errorf("--frames must be >= 1, got %d", f.frames)
	}
	if f.hp < 1 {
		return f, fmt.Errorf("--hp must be >= 1, got %d", f.hp)
	}
	return f, nil
}

// cmdServeSignals runs the server until SIGINT or SIGTERM.
func cmdServeSignals(args []string, stdout, stderr io.Writer) int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return cmdServe(ctx, args, stdout, stderr)
}

// cmdServe runs the HTTP API until ctx is cancelled, then shuts down gracefully.
func cmdServe(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	f, err := parseServeFlags(args, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "pf serve: %v\nusage: pf serve [--addr HOST:PORT] [--data DIR] [--frames N] [--hp N]\n", err)
		return 2
	}
	h, err := server.New(server.Options{DataDir: f.data, FrameBudget: f.frames, MaxHP: f.hp})
	if err != nil {
		fmt.Fprintf(stderr, "pf serve: %v\n", err)
		return 2
	}
	ln, err := net.Listen("tcp", f.addr)
	if err != nil {
		fmt.Fprintf(stderr, "pf serve: %v\n", err)
		return 2
	}
	srv := &http.Server{
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	fmt.Fprintf(stderr, "pf serve: listening on http://%s (data %s, %d frames, %d HP)\n", ln.Addr(), f.data, f.frames, f.hp)
	errc := make(chan error, 1)
	go func() { errc <- srv.Serve(ln) }()
	select {
	case err := <-errc:
		fmt.Fprintf(stderr, "pf serve: %v\n", err)
		return 1
	case <-ctx.Done():
	}
	shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		fmt.Fprintf(stderr, "pf serve: shutdown: %v\n", err)
		return 1
	}
	if err := <-errc; err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Fprintf(stderr, "pf serve: %v\n", err)
		return 1
	}
	fmt.Fprintln(stderr, "pf serve: stopped")
	return 0
}
