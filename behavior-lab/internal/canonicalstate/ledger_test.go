package canonicalstate

import "testing"

func TestLedgerHeadIgnoresAuditTimestamps(t *testing.T) {
	var a, b Ledger
	a.Append("X", "agent", "i", "o", nil)
	b.Append("X", "agent", "i", "o", nil)
	if a.HeadHash != b.HeadHash {
		t.Fatalf("ledger head must be deterministic %s %s", a.HeadHash, b.HeadHash)
	}
}
