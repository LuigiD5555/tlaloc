package tlaloque

import (
	"encoding/json"
	"sort"
	"strings"
)

type EnsembleMemberInput struct {
	WorkerID string          `json:"worker_id"`
	Output   json.RawMessage `json:"output"`
}

// EnsembleMemberInputs extracts the successful member outputs visible to a
// fusion worker at dispatch time. With ANY/QUORUM this can intentionally be a
// strict subset of all configured members.
func EnsembleMemberInputs(context map[string]json.RawMessage, prefix string) []EnsembleMemberInput {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		prefix = DefaultEnsembleContextPrefix
	}
	rows := make([]EnsembleMemberInput, 0)
	for key, output := range context {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		workerID := strings.TrimSpace(strings.TrimPrefix(key, prefix))
		if workerID == "" {
			continue
		}
		rows = append(rows, EnsembleMemberInput{
			WorkerID: workerID,
			Output:   append(json.RawMessage(nil), output...),
		})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].WorkerID < rows[j].WorkerID })
	return rows
}
