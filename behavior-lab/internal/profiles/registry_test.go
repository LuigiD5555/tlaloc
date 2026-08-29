package profiles

import (
	"crypto/sha256"
	"fmt"
	"testing"
)

func TestRegistryHasSingleActivePathPerProfile(t *testing.T) {
	registry := Builtin()
	ids := registry.IDs()
	if len(ids) != 2 {
		t.Fatalf("got %v", ids)
	}
	for _, id := range ids {
		profile, err := registry.Lookup(id, "0.1.0")
		if err != nil || profile.ID != id {
			t.Fatalf("lookup %s: %v", id, err)
		}
	}
}
func TestRelationalFixtureMatchesUpstreamRelease(t *testing.T) {
	const expectedSHA = "363904ae72ed7b9db7d4b371c91ec259e86f3e98bb74e18d2a2b882219c0a5c1"
	actual := fmt.Sprintf("%x", sha256.Sum256(relationalFixtureData))
	if actual != expectedSHA {
		t.Fatalf("upstream fixture drift: got %s", actual)
	}
}
func TestRegistryRejectsUnknownAndVersionDrift(t *testing.T) {
	registry := Builtin()
	if _, err := registry.Lookup("missing", "0.1.0"); err == nil {
		t.Fatal("expected unsupported profile")
	}
	if _, err := registry.Lookup(CoherentID, "9.0"); err == nil {
		t.Fatal("expected unsupported version")
	}
}
func TestRelationalComparatorIsStrict(t *testing.T) {
	profile, _ := Builtin().Lookup(RelationalID, "0.1.0")
	expected := profile.Cases[0].ExpectedRaw
	if result := profile.Compare(expected, expected); !result.Pass {
		t.Fatalf("unexpected %#v", result.Findings)
	}
	if result := profile.Compare(expected, `{"contract":"wrong","outcome":"FIXED_POINT","state":{},"steps":1}`); result.Pass {
		t.Fatal("expected mismatch")
	}
}
