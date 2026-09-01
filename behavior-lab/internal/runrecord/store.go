package runrecord

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func Store(root string, record Record) (string, error) {
	if err := Validate(record); err != nil {
		return "", err
	}
	observedAt, err := time.Parse(time.RFC3339, record.Model.ObservedAt)
	if err != nil {
		return "", fmt.Errorf("parse model observed_at: %w", err)
	}
	monthDirectory := filepath.Join(root, observedAt.UTC().Format("2006-01"))
	if err := os.MkdirAll(monthDirectory, 0o755); err != nil {
		return "", err
	}
	filename := strings.ReplaceAll(record.RunID, ":", "-") + ".json"
	recordPath := filepath.Join(monthDirectory, filename)
	encoded, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return "", err
	}
	encoded = append(encoded, '\n')
	recordFile, err := os.OpenFile(recordPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return "", fmt.Errorf("immutable run record already exists: %s", recordPath)
		}
		return "", err
	}
	if _, err := recordFile.Write(encoded); err != nil {
		recordFile.Close()
		return "", err
	}
	if err := recordFile.Close(); err != nil {
		return "", err
	}
	indexPath := filepath.Join(root, "index.jsonl")
	indexFile, err := os.OpenFile(indexPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		return "", err
	}
	indexEntry := struct {
		RunID   string `json:"run_id"`
		EnvHash string `json:"env_hash"`
		Path    string `json:"path"`
		Verdict string `json:"verdict"`
	}{
		RunID: record.RunID, EnvHash: record.EnvHash,
		Path:    filepath.ToSlash(filepath.Join(observedAt.UTC().Format("2006-01"), filename)),
		Verdict: record.Outcome.Verdict,
	}
	indexBody, err := json.Marshal(indexEntry)
	if err == nil {
		_, err = indexFile.Write(append(indexBody, '\n'))
	}
	closeErr := indexFile.Close()
	if err != nil {
		return "", err
	}
	if closeErr != nil {
		return "", closeErr
	}
	return recordPath, nil
}

func Load(path string) (Record, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return Record{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	var record Record
	if err := decoder.Decode(&record); err != nil {
		return Record{}, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return Record{}, errors.New("run record contains trailing JSON")
	}
	if err := Validate(record); err != nil {
		return Record{}, err
	}
	return record, nil
}

func LoadIndex(path string) ([]string, error) {
	indexFile, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer indexFile.Close()
	var entries []string
	scanner := bufio.NewScanner(indexFile)
	for scanner.Scan() {
		entries = append(entries, scanner.Text())
	}
	return entries, scanner.Err()
}
