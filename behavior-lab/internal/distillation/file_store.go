package distillation

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type FileStore struct {
	Root string
}

func (store FileStore) Save(ctx context.Context, artifact Artifact) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	root := strings.TrimSpace(store.Root)
	if root == "" {
		return fmt.Errorf("model store root is required")
	}
	if artifact.Schema != ArtifactSchemaR0 || strings.TrimSpace(artifact.WorkerID) == "" {
		return fmt.Errorf("invalid specialist artifact")
	}
	workerID := filepath.Base(artifact.WorkerID)
	if workerID != artifact.WorkerID || workerID == "." {
		return fmt.Errorf("worker_id %q is not a safe artifact name", artifact.WorkerID)
	}
	directory := filepath.Join(root, strings.ToLower(artifact.Capability), workerID)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(artifact, "", "  ")
	if err != nil {
		return err
	}
	temporary, err := os.CreateTemp(directory, ".metadata-*.json")
	if err != nil {
		return err
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if _, err := temporary.Write(append(payload, '\n')); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryName, filepath.Join(directory, "metadata.json"))
}
