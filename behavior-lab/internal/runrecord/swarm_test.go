package runrecord

import "testing"

func TestCompareToBaseline(t *testing.T) {
	swarm := SwarmAccounting{WallMS: 40_000, TotalTokensIn: 1200, TotalTokensOut: 400}

	// Swarm matches quality, uses fewer tokens, finishes sooner -> wins.
	win := CompareToBaseline(0.82, 0.80, swarm, 5000, 60_000)
	if !win.SwarmWins {
		t.Fatalf("expected swarm to win: %+v", win)
	}

	// Same cost/speed advantage but lower quality -> does not win.
	lose := CompareToBaseline(0.70, 0.80, swarm, 5000, 60_000)
	if lose.SwarmWins {
		t.Error("swarm must not win when quality is below baseline")
	}
	if !lose.CheaperThanBaseline || !lose.FasterThanBaseline {
		t.Error("cheaper/faster flags should still be true independently of the quality gate")
	}

	// Cheaper and same quality but slower -> does not win (all three required).
	slow := CompareToBaseline(0.80, 0.80, SwarmAccounting{WallMS: 90_000, TotalTokensIn: 100, TotalTokensOut: 100}, 5000, 60_000)
	if slow.SwarmWins {
		t.Error("swarm must not win when it is slower than the baseline")
	}
}
