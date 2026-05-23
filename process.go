package main

import "strings"

type Process struct {
	PID         int
	PPID        int
	Name        string
	Command     string
	RSSBytes    int64
	CPUPercent  float64
	AgeSecs     int64
	SafetyClass SafetyClass
}

type StatsProvider interface {
	MemStats() (MemStats, string, error) // returns MemStats, pressure string, error
	Processes() ([]Process, error)
}

type ProcessKiller interface {
	Kill(pid int) error
}

type ProcessRestarter interface {
	Restart(command string) error
}

type SafetyClass int

// Ordered from MOST permissive (kill freely) to LEAST permissive (do not kill).
// Group aggregation picks the lowest value among members (most permissive).
const (
	SafeToKill  SafetyClass = iota // 0 - orphans, leftover daemons
	AutoRestart                    // 1 - respawns automatically (MCP servers, ChromaDB)
	App                            // 2 - user-facing app, killable but loses work
	Critical                       // 3 - killing crashes session / requires reboot
)

// AssignSafetyClass returns the classification for a process based on its
// name and full command. See descriptions.go for the rule set.
func AssignSafetyClass(name, cmd string) SafetyClass {
	if rule := matchProcessRule(name, cmd); rule != nil {
		return rule.Safety
	}
	// Fallback heuristics for unknown processes
	if strings.HasPrefix(name, "com.apple.") {
		return Critical
	}
	return App
}
