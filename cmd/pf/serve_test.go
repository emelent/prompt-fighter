package main

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"
)

// syncBuffer is a goroutine-safe bytes.Buffer.
type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

func TestServeDefaults(t *testing.T) {
	f, err := parseServeFlags(nil, io.Discard)
	if err != nil || f.addr != "127.0.0.1:8080" || f.data != "./data" {
		t.Errorf("defaults %+v %v", f, err)
	}
}

func TestServeBadAddr(t *testing.T) {
	var stderr bytes.Buffer
	code := cmdServe(context.Background(), []string{"--addr", "nonsense", "--data", t.TempDir()}, io.Discard, &stderr)
	if code != 2 || stderr.Len() == 0 {
		t.Errorf("code %d stderr %q", code, stderr.String())
	}
}

func TestServeUsageError(t *testing.T) {
	if code := cmdServe(context.Background(), []string{"--bogus"}, io.Discard, io.Discard); code != 2 {
		t.Errorf("code %d", code)
	}
}

func TestServeRunsAndShutsDown(t *testing.T) {
	data := filepath.Join(t.TempDir(), "nested", "data")
	ctx, cancel := context.WithCancel(context.Background())
	stderr := &syncBuffer{}
	done := make(chan int, 1)
	go func() { done <- cmdServe(ctx, []string{"--addr", "127.0.0.1:0", "--data", data}, io.Discard, stderr) }()

	re := regexp.MustCompile(`listening on (http://[^ ]+) \(data `)
	var url string
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(10 * time.Millisecond) {
		if m := re.FindStringSubmatch(stderr.String()); m != nil {
			url = m[1]
			break
		}
	}
	if url == "" {
		cancel()
		t.Fatalf("server did not start: %q", stderr.String())
	}
	if n := strings.Count(stderr.String(), "\n"); n != 1 {
		t.Errorf("startup should log one line, got %q", stderr.String())
	}
	resp, err := http.Get(url + "/healthz")
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("healthz: %v %v", err, resp)
	}
	resp.Body.Close()
	cancel()
	select {
	case code := <-done:
		if code != 0 {
			t.Errorf("exit %d, stderr %q", code, stderr.String())
		}
	case <-time.After(15 * time.Second):
		t.Fatal("server did not shut down")
	}
}
