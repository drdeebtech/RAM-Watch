package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const pageSize = 16384
const gbBytes = 1073741824.0

type MemStats struct {
	UsedGB  float64
	FreeGB  float64
	TotalGB float64
	// Raw segments for the gauge
	ActiveGB     float64
	WiredGB      float64
	CompressedGB float64
}

var vmStatPattern = regexp.MustCompile(`(?m)^([^:]+):\s+([\d]+)\.`)

func ParseMemoryStats(vmstatOutput, hwMemsize string) (MemStats, error) {
	totalBytes, err := strconv.ParseInt(strings.TrimSpace(hwMemsize), 10, 64)
	if err != nil {
		return MemStats{}, fmt.Errorf("invalid hw.memsize %q: %w", hwMemsize, err)
	}

	fields := map[string]int64{}
	for _, m := range vmStatPattern.FindAllStringSubmatch(vmstatOutput, -1) {
		key := strings.TrimSpace(m[1])
		val, _ := strconv.ParseInt(m[2], 10, 64)
		fields[key] = val
	}

	active := fields["Pages active"]
	wired := fields["Pages wired down"]
	compressed := fields["Pages occupied by compressor"]
	free := fields["Pages free"]

	usedBytes := float64((active + wired + compressed) * pageSize)
	totalGB := float64(totalBytes) / gbBytes
	usedGB := usedBytes / gbBytes
	_ = free // Pages free is a tiny subset; available = total − used

	return MemStats{
		UsedGB:       usedGB,
		FreeGB:       totalGB - usedGB,
		TotalGB:      totalGB,
		ActiveGB:     float64(active*pageSize) / gbBytes,
		WiredGB:      float64(wired*pageSize) / gbBytes,
		CompressedGB: float64(compressed*pageSize) / gbBytes,
	}, nil
}
