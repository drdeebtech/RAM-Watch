package main

import "strings"

type Process struct {
	PID         int
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

const (
	SafeToKill  SafetyClass = iota
	AutoRestart SafetyClass = iota
	System      SafetyClass = iota
)

func AssignSafetyClass(cmd string) SafetyClass {
	switch {
	case strings.Contains(cmd, "claude/versions"):
		return SafeToKill
	case strings.Contains(cmd, "npm exec"):
		return AutoRestart
	case strings.Contains(cmd, "chroma-mcp"):
		return AutoRestart
	default:
		return System
	}
}
