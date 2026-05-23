package main

import "testing"

func TestGroupProcesses_HeuristicMatchesSafari(t *testing.T) {
	procs := []Process{
		{PID: 10, Name: "Safari", Command: "/Applications/Safari.app/Contents/MacOS/Safari", RSSBytes: 200 * 1024 * 1024},
		{PID: 11, Name: "com.apple.WebKit.WebContent", Command: "WebContent tab=1", PPID: 10, RSSBytes: 100 * 1024 * 1024},
		{PID: 12, Name: "com.apple.WebKit.GPU", Command: "GPU", PPID: 10, RSSBytes: 30 * 1024 * 1024},
	}
	groups := GroupProcesses(procs)
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d (%v)", len(groups), groupNames(groups))
	}
	if groups[0].Name != "Safari" {
		t.Errorf("expected group Safari, got %q", groups[0].Name)
	}
	if len(groups[0].Members) != 3 {
		t.Errorf("expected 3 members in Safari, got %d", len(groups[0].Members))
	}
}

func TestGroupProcesses_PPIDInheritance(t *testing.T) {
	// claude (PID 100) is heuristic-matched. exec (PID 200) has no heuristic
	// match, but its PPID = 100, so it should inherit "Claude".
	procs := []Process{
		{PID: 100, Name: "claude", Command: "/usr/local/bin/claude", RSSBytes: 500 * 1024 * 1024},
		{PID: 200, Name: "exec", Command: "/usr/local/bin/exec helper", PPID: 100, RSSBytes: 50 * 1024 * 1024},
	}
	groups := GroupProcesses(procs)
	if len(groups) != 1 {
		t.Fatalf("expected PPID inheritance to produce 1 group, got %d (%v)", len(groups), groupNames(groups))
	}
	if groups[0].Name != "Claude" {
		t.Errorf("expected Claude, got %q", groups[0].Name)
	}
}

func TestGroupProcesses_SortByTotalRAMDescending(t *testing.T) {
	procs := []Process{
		{PID: 1, Name: "small", RSSBytes: 10 * 1024 * 1024},
		{PID: 2, Name: "huge", RSSBytes: 1000 * 1024 * 1024},
		{PID: 3, Name: "medium", RSSBytes: 100 * 1024 * 1024},
	}
	groups := GroupProcesses(procs)
	if groups[0].Name != "huge" || groups[len(groups)-1].Name != "small" {
		t.Errorf("expected huge → medium → small ordering, got %v", groupNames(groups))
	}
}

func TestGroupProcesses_SafetyClassAggregation(t *testing.T) {
	// Group with a SafeToKill member should be marked SafeToKill (most permissive).
	procs := []Process{
		{PID: 1, Name: "claude", Command: "/usr/local/bin/claude", SafetyClass: System, RSSBytes: 500 * 1024 * 1024},
		{PID: 2, Name: "exec", Command: "/usr/local/bin/npm exec @mcp/server", SafetyClass: AutoRestart, PPID: 1, RSSBytes: 50 * 1024 * 1024},
	}
	groups := GroupProcesses(procs)
	if groups[0].SafetyClass != AutoRestart {
		t.Errorf("expected group SafetyClass=AutoRestart (most permissive), got %v", groups[0].SafetyClass)
	}
}

func TestGroupProcesses_AppleSystemBucket(t *testing.T) {
	// com.apple.* with no parent + no heuristic match → "macOS System"
	procs := []Process{
		{PID: 1, Name: "com.apple.geod", Command: "geod"},
		{PID: 2, Name: "com.apple.ThemeWidgetControlViewService", Command: "ThemeWidget"},
	}
	groups := GroupProcesses(procs)
	if len(groups) != 1 || groups[0].Name != "macOS System" {
		t.Errorf("expected single macOS System bucket, got %v", groupNames(groups))
	}
}

func groupNames(gs []ProcessGroup) []string {
	out := make([]string, len(gs))
	for i, g := range gs {
		out[i] = g.Name
	}
	return out
}
