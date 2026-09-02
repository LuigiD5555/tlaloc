package foldtest

import (
	"testing"
)

func TestParseCommand(t *testing.T) {
	tests := []struct {
		input       string
		wantOk      bool
		wantOp      string
		wantAddress string
		wantTerm    string
		wantDepth   int
	}{
		// UNFOLD cases
		{"UNFOLD page:5", true, "UNFOLD", "page:5", "", 0},
		{"UNFOLD block:doc/page-3/blocks/2", true, "UNFOLD", "block:doc/page-3/blocks/2", "", 0},
		{"UNFOLD page:10:low", true, "UNFOLD", "page:10", "", 0},
		{"UNFOLD page:10:high", true, "UNFOLD", "page:10", "", 0},

		// GROUP cases
		{"GROUP term:authentication depth:2", true, "GROUP", "", "authentication", 2},
		{"GROUP term:security depth:1", true, "GROUP", "", "security", 1},

		// Invalid cases
		{"UNFOLD", false, "", "", "", 0},
		{"GROUP", false, "", "", "", 0},
		{"UNKNOWN page:5", false, "", "", "", 0},
		{"GROUP term:auth", false, "", "", "", 0},
		{"GROUP depth:2", false, "", "", "", 0},
	}

	for _, tt := range tests {
		cmd, ok := ParseCommand(tt.input)
		if ok != tt.wantOk {
			t.Errorf("ParseCommand(%q) ok = %v, want %v", tt.input, ok, tt.wantOk)
			continue
		}

		if !ok {
			continue
		}

		if cmd.Op != tt.wantOp {
			t.Errorf("ParseCommand(%q) Op = %s, want %s", tt.input, cmd.Op, tt.wantOp)
		}

		if cmd.Address != tt.wantAddress {
			t.Errorf("ParseCommand(%q) Address = %s, want %s", tt.input, cmd.Address, tt.wantAddress)
		}

		if cmd.Term != tt.wantTerm {
			t.Errorf("ParseCommand(%q) Term = %s, want %s", tt.input, cmd.Term, tt.wantTerm)
		}

		if cmd.Depth != tt.wantDepth {
			t.Errorf("ParseCommand(%q) Depth = %d, want %d", tt.input, cmd.Depth, tt.wantDepth)
		}
	}
}
