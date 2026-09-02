//go:build linux

package tlaloque

import "syscall"

// peakRSSBytes returns the process's peak resident set size so far, in
// bytes. On Linux getrusage reports Maxrss in kilobytes. This is a
// process-global high-water mark — see SwarmAccounting.PeakRSSDeltaBytes.
func peakRSSBytes() int64 {
	var usage syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &usage); err != nil {
		return 0
	}
	return int64(usage.Maxrss) * 1024
}
