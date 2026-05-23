package main

import (
	"math"
	"testing"
)

const sampleVmStat = `Mach Virtual Memory Statistics: (page size of 16384 bytes)
Pages free:                               12800.
Pages active:                             98304.
Pages inactive:                           32768.
Pages speculative:                         4096.
Pages throttled:                              0.
Pages wired down:                         49152.
Pages purgeable:                           1024.
"Translation faults":                   123456.
Pages copy-on-write:                       2048.
Pages zero filled:                        98765.
Pages reactivated:                          512.
Pages purged:                               256.
File-backed pages:                        24576.
Anonymous pages:                          73728.
Pages stored in compressor:               16384.
Pages occupied by compressor:             16384.
Decompressions:                            1024.
Compressions:                              2048.
Pageouts:                                     0.
Pageins:                                   4096.
Swapouts:                                     0.
Swapins:                                      0.
`

// hwMemsize is what sysctl hw.memsize returns (bytes as string)
const sampleHwMemsize = "17179869184" // 16 GB

func roundTo1(f float64) float64 {
	return math.Round(f*10) / 10
}

func TestParseMemoryStats_UsedGB(t *testing.T) {
	stats, err := ParseMemoryStats(sampleVmStat, sampleHwMemsize)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// active=98304, wired=49152, compressor=16384 pages × 16384 bytes
	// used = (98304+49152+16384) × 16384 = 163840 × 16384 = 2684354560 bytes = ~2.5 GB
	want := 2.5
	if roundTo1(stats.UsedGB) != want {
		t.Errorf("UsedGB: want %.1f, got %.1f", want, stats.UsedGB)
	}
}

func TestParseMemoryStats_TotalGB(t *testing.T) {
	stats, err := ParseMemoryStats(sampleVmStat, sampleHwMemsize)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := 16.0
	if roundTo1(stats.TotalGB) != want {
		t.Errorf("TotalGB: want %.1f, got %.1f", want, stats.TotalGB)
	}
}

func TestParseMemoryStats_FreeGB(t *testing.T) {
	stats, err := ParseMemoryStats(sampleVmStat, sampleHwMemsize)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Pages free=12800 × 16384 bytes = 209715200 bytes ≈ 0.2 GB
	// macOS "free" is truly-unused pages only, not total−used
	want := 0.2
	if roundTo1(stats.FreeGB) != want {
		t.Errorf("FreeGB: want %.1f, got %.1f", want, stats.FreeGB)
	}
}

func TestParseMemoryStats_InvalidHwMemsize(t *testing.T) {
	_, err := ParseMemoryStats(sampleVmStat, "not-a-number")
	if err == nil {
		t.Error("expected error for invalid hwMemsize, got nil")
	}
}
