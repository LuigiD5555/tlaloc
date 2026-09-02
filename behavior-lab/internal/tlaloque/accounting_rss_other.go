//go:build !linux

package tlaloque

// peakRSSBytes is only implemented on Linux; elsewhere the run-level RSS
// ceiling is simply not reported.
func peakRSSBytes() int64 { return 0 }
