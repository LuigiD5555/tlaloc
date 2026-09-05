package episode

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Store writes one Episode under root, sharded by month (UTC, from
// observedAt) and named by EpisodeID, mirroring runrecord.Store's
// immutable-write convention: it refuses to overwrite an existing episode
// file rather than silently replacing prior experience data.
func Store(root string, ep Episode, observedAt time.Time) (string, error) {
	if ep.EpisodeID == "" {
		return "", errors.New("episode: Store: EpisodeID is empty")
	}
	if ep.Schema != Schema {
		return "", fmt.Errorf("episode: Store: Schema = %q, want %q", ep.Schema, Schema)
	}
	if observedAt.IsZero() {
		return "", errors.New("episode: Store: observedAt is zero")
	}

	monthDirectory := filepath.Join(root, observedAt.UTC().Format("2006-01"))
	if err := os.MkdirAll(monthDirectory, 0o755); err != nil {
		return "", err
	}

	filename := strings.NewReplacer("/", "-", "\\", "-", ":", "-").Replace(ep.EpisodeID) + ".json"
	episodePath := filepath.Join(monthDirectory, filename)

	encoded, err := json.MarshalIndent(ep, "", "  ")
	if err != nil {
		return "", err
	}
	encoded = append(encoded, '\n')

	file, err := os.OpenFile(episodePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return "", fmt.Errorf("episode: Store: immutable episode already exists: %s", episodePath)
		}
		return "", err
	}
	defer file.Close()

	if _, err := file.Write(encoded); err != nil {
		return "", err
	}
	return episodePath, nil
}

// StoreAll writes each Episode with Store, stopping at the first error. On
// error it returns the paths successfully written before the failure so the
// caller can see partial progress.
func StoreAll(root string, episodes []Episode, observedAt time.Time) ([]string, error) {
	written := make([]string, 0, len(episodes))
	for _, ep := range episodes {
		path, err := Store(root, ep, observedAt)
		if err != nil {
			return written, err
		}
		written = append(written, path)
	}
	return written, nil
}

// Load reads one Episode back from disk.
func Load(path string) (Episode, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Episode{}, err
	}
	var ep Episode
	if err := json.Unmarshal(data, &ep); err != nil {
		return Episode{}, err
	}
	return ep, nil
}
