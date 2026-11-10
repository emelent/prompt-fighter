package server

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStoreRoundTrip(t *testing.T) {
	st, err := newStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	m := readyMatch(t)
	play(m, "p1")
	if err := st.save(m); err != nil {
		t.Fatal(err)
	}
	if err := st.saveRound(m.ID, 1, []byte(`{"log":1}`)); err != nil {
		t.Fatal(err)
	}
	got, err := st.load(m.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != m.ID || got.A.Name != "alice" || got.B.Wins != m.B.Wins || len(got.Rounds) != 1 || got.StoreVersion != 1 {
		t.Errorf("loaded %+v", got)
	}
	b, err := st.loadRound(m.ID, 1)
	if err != nil || !bytes.Equal(b, []byte(`{"log":1}`)) {
		t.Errorf("round %q %v", b, err)
	}
	if _, err := st.loadRound(m.ID, 2); err != errNoMatch {
		t.Errorf("missing round err %v", err)
	}
	if !st.exists(m.ID) || st.exists("fedcba9876543210") {
		t.Errorf("exists wrong")
	}
}

func TestStoreRejectsBadIDs(t *testing.T) {
	st, _ := newStore(t.TempDir())
	for _, id := range []string{"../../etc", "0123456789ABCDEF", "0123", "0123456789abcdef/../x", ""} {
		if _, err := st.load(id); err != errNoMatch {
			t.Errorf("load(%q) err %v, want errNoMatch", id, err)
		}
		if err := st.saveRound(id, 1, nil); err == nil {
			t.Errorf("saveRound(%q) accepted a bad id", id)
		}
	}
}

func TestStoreRejectsUnknownVersion(t *testing.T) {
	st, _ := newStore(t.TempDir())
	m := readyMatch(t)
	m.StoreVersion = 99
	if err := st.save(m); err != nil {
		t.Fatal(err)
	}
	if _, err := st.load(m.ID); err == nil || !strings.Contains(err.Error(), "store_version 99") {
		t.Errorf("err %v", err)
	}
}

func TestStoreIgnoresLeftoverTempFile(t *testing.T) {
	dir := t.TempDir()
	st, _ := newStore(dir)
	m := readyMatch(t)
	if err := st.save(m); err != nil {
		t.Fatal(err)
	}
	// Simulate a crash mid-write: a half-written temp file next to match.json.
	tmp := filepath.Join(st.dir(m.ID), ".tmp-12345")
	os.WriteFile(tmp, []byte(`{"store_version": 1, "match_id": "trunc`), 0o644)
	got, err := st.load(m.ID)
	if err != nil || got.A.Name != "alice" {
		t.Errorf("load after crash: %v %+v", err, got)
	}
	// The next save still succeeds and leaves no new temp files behind.
	if err := st.save(m); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(st.dir(m.ID))
	n := 0
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".tmp-") {
			n++
		}
	}
	if n != 1 {
		t.Errorf("temp files %d, want only the simulated leftover", n)
	}
}

func TestStoreAtomicReplace(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "f.json")
	if err := writeAtomic(p, []byte("one")); err != nil {
		t.Fatal(err)
	}
	if err := writeAtomic(p, []byte("two")); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(p); string(b) != "two" {
		t.Errorf("content %q", b)
	}
}
