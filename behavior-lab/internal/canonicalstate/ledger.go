package canonicalstate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"
)

func (l *Ledger) Append(operation, actor, inputHash, outputHash string, evidence []string) {
	if l.Schema == "" {
		l.Schema = LedgerSchema
	}
	t := Transition{Index: len(l.Transitions), Operation: operation, Actor: actor, InputHash: inputHash, OutputHash: outputHash, Evidence: append([]string(nil), evidence...), TimestampUTC: time.Now().UTC()}
	l.Transitions = append(l.Transitions, t)
	l.HeadHash = ledgerHash(l.Transitions)
}

func ledgerHash(rows []Transition) string {
	stable := append([]Transition(nil), rows...)
	for i := range stable {
		stable[i].TimestampUTC = time.Time{}
	}
	b, _ := json.Marshal(stable)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
