package blackboard

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Store struct{ Root string }

func DefaultRoot() string {
	if x := strings.TrimSpace(os.Getenv("XDG_STATE_HOME")); x != "" {
		return filepath.Join(x, "tlaloc", "blackboard")
	}
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return filepath.Join(".", ".tlaloc-blackboard")
	}
	return filepath.Join(home, ".local", "state", "tlaloc", "blackboard")
}

func New(root string) Store {
	if strings.TrimSpace(root) == "" {
		root = DefaultRoot()
	}
	return Store{Root: root}
}

func ContentID(e Entry) (string, error) {
	e.ID = ""
	e.RecordedAt = ""
	e.References = normalizeStrings(e.References)
	body, err := json.Marshal(e)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:]), nil
}

// Append is append-only and idempotent. A hard link publishes a fully synced
// temporary file atomically; a concurrent publisher of the same content is a
// successful no-op rather than a conflict.
func (s Store) Append(e Entry) (bool, Entry, error) {
	if e.Schema == "" {
		e.Schema = EntrySchema
	}
	e.References = normalizeStrings(e.References)
	id, err := ContentID(e)
	if err != nil {
		return false, e, err
	}
	if e.ID != "" && e.ID != id {
		return false, e, fmt.Errorf("blackboard id mismatch: %s != %s", e.ID, id)
	}
	e.ID = id
	if err := ValidateEntry(e); err != nil {
		return false, e, err
	}
	if e.RecordedAt == "" {
		e.RecordedAt = time.Now().UTC().Format(time.RFC3339Nano)
	}
	dir := filepath.Join(s.Root, "entries")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return false, e, err
	}
	final := filepath.Join(dir, id+".json")
	if _, err := os.Stat(final); err == nil {
		return false, e, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, e, err
	}
	body, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return false, e, err
	}
	body = append(body, '\n')
	tmp, err := os.CreateTemp(dir, ".entry-*.tmp")
	if err != nil {
		return false, e, err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return false, e, err
	}
	if _, err := tmp.Write(body); err != nil {
		tmp.Close()
		return false, e, err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return false, e, err
	}
	if err := tmp.Close(); err != nil {
		return false, e, err
	}
	if err := os.Link(tmpName, final); err != nil {
		if _, statErr := os.Stat(final); statErr == nil {
			return false, e, nil
		}
		return false, e, err
	}
	return true, e, nil
}

func (s Store) LoadAll() ([]Entry, error) {
	dir := filepath.Join(s.Root, "entries")
	files, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return []Entry{}, nil
	}
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(files))
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".json") {
			names = append(names, f.Name())
		}
	}
	sort.Strings(names)
	out := make([]Entry, 0, len(names))
	for _, name := range names {
		body, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		var e Entry
		dec := json.NewDecoder(strings.NewReader(string(body)))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&e); err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		if err := ValidateEntry(e); err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		id, err := ContentID(e)
		if err != nil {
			return nil, err
		}
		if e.ID != id {
			return nil, fmt.Errorf("%s: content id mismatch", name)
		}
		out = append(out, e)
	}
	return out, nil
}

func (s Store) Snapshot(runID string) (Snapshot, error) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return Snapshot{}, fmt.Errorf("run_id is required")
	}
	all, err := s.LoadAll()
	if err != nil {
		return Snapshot{}, err
	}
	entries := make([]Entry, 0)
	for _, e := range all {
		if e.RunID == runID {
			entries = append(entries, e)
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].ID < entries[j].ID })
	return Snapshot{Schema: SnapshotSchema, RunID: runID, Entries: entries}, nil
}
