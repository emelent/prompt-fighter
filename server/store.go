package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
)

// storeVersion is the on-disk format version of match.json.
const storeVersion = 1

var errNoMatch = errors.New("match not found")

var matchIDRe = regexp.MustCompile(`^[0-9a-f]{16}$`)

// store keeps one directory per match:
//
//	<data>/matches/<id>/match.json      match state (store_version 1)
//	<data>/matches/<id>/rounds/<n>.json exact event log bytes of round n
type store struct{ root string }

func newStore(dataDir string) (*store, error) {
	root := filepath.Join(dataDir, "matches")
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("server: data directory: %w", err)
	}
	return &store{root: root}, nil
}

func validID(id string) bool { return matchIDRe.MatchString(id) }

func (st *store) dir(id string) string { return filepath.Join(st.root, id) }

func (st *store) exists(id string) bool {
	if !validID(id) {
		return false
	}
	_, err := os.Stat(filepath.Join(st.dir(id), "match.json"))
	return err == nil
}

func (st *store) load(id string) (*Match, error) {
	if !validID(id) {
		return nil, errNoMatch
	}
	b, err := os.ReadFile(filepath.Join(st.dir(id), "match.json"))
	if errors.Is(err, os.ErrNotExist) {
		return nil, errNoMatch
	}
	if err != nil {
		return nil, err
	}
	var m Match
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("match %s: corrupt match.json: %w", id, err)
	}
	if m.StoreVersion != storeVersion {
		return nil, fmt.Errorf("match %s: unsupported store_version %d (this server reads %d)", id, m.StoreVersion, storeVersion)
	}
	return &m, nil
}

func (st *store) save(m *Match) error {
	if !validID(m.ID) {
		return fmt.Errorf("invalid match id %q", m.ID)
	}
	if err := os.MkdirAll(filepath.Join(st.dir(m.ID), "rounds"), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(filepath.Join(st.dir(m.ID), "match.json"), append(b, '\n'))
}

func (st *store) roundPath(id string, n int) string {
	return filepath.Join(st.dir(id), "rounds", strconv.Itoa(n)+".json")
}

func (st *store) saveRound(id string, n int, log []byte) error {
	if !validID(id) {
		return fmt.Errorf("invalid match id %q", id)
	}
	if err := os.MkdirAll(filepath.Join(st.dir(id), "rounds"), 0o755); err != nil {
		return err
	}
	return writeAtomic(st.roundPath(id, n), log)
}

func (st *store) loadRound(id string, n int) ([]byte, error) {
	if !validID(id) {
		return nil, errNoMatch
	}
	b, err := os.ReadFile(st.roundPath(id, n))
	if errors.Is(err, os.ErrNotExist) {
		return nil, errNoMatch
	}
	return b, err
}

// writeAtomic replaces path with data so that readers and crashes only ever
// see the old or the new contents: temp file in the same directory, fsync,
// rename, then fsync the directory.
func writeAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	f, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	ok := false
	defer func() {
		if !ok {
			os.Remove(tmp)
		}
	}()
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	ok = true
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}
