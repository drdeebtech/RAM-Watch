package main

import (
	"sort"
	"strings"
)

// ProcessGroup aggregates related processes under a named application bucket.
type ProcessGroup struct {
	Name          string
	TotalRSSBytes int64
	SafetyClass   SafetyClass // most permissive (lowest enum) across members
	Members       []Process
}

// knownGroups maps command/name patterns to human-readable app names.
// Checked in order; first match wins.
var knownGroups = []struct {
	name  string
	match func(name, cmd string) bool
}{
	{"Safari", func(n, c string) bool {
		return n == "Safari" ||
			strings.HasPrefix(n, "com.apple.WebKit") ||
			strings.HasPrefix(n, "com.apple.Safari") ||
			strings.Contains(c, "/Safari.app/")
	}},
	{"ChromaDB", func(n, c string) bool {
		return strings.Contains(c, "chroma-mcp") || strings.Contains(c, "chroma_mcp")
	}},
	{"Claude", func(n, c string) bool {
		return n == "claude" ||
			strings.Contains(c, "claude/versions") ||
			strings.Contains(c, "npm exec") ||
			strings.Contains(c, "@modelcontextprotocol")
	}},
	{"Cursor", func(n, c string) bool {
		return n == "Cursor" || n == "2.1.150" ||
			strings.Contains(c, "/Cursor.app/") || strings.Contains(c, "/Cursor.app ")
	}},
	{"Grammarly", func(n, c string) bool {
		return strings.HasPrefix(n, "Grammarly") || strings.Contains(c, "Grammarly")
	}},
	{"Google Chrome", func(n, c string) bool {
		return strings.Contains(c, "/Google Chrome.app/") || strings.Contains(c, "/Google/Chrome/")
	}},
	{"Google", func(n, c string) bool {
		return strings.HasPrefix(n, "Google")
	}},
	{"Bun", func(n, c string) bool { return n == "bun" }},
	{"MariaDB", func(n, c string) bool { return n == "mariadbd" }},
	{"Tailscale", func(n, c string) bool {
		return strings.Contains(n, "tailscale") || strings.Contains(c, "tailscale")
	}},
	{"Electron Apps", func(n, c string) bool { return n == "Electron" }},
}

// groupHint applies name/command heuristics and returns the assigned group, or
// "" if no rule matches.
func groupHint(p Process) string {
	for _, g := range knownGroups {
		if g.match(p.Name, p.Command) {
			return g.name
		}
	}
	return ""
}

// GroupProcesses folds a flat process list into named app groups.
// Strategy: heuristic name match first. PPID inheritance ONLY propagates a
// heuristic group (so we don't end up bucketing everything under "launchd").
// Anything still ungrouped falls back to its own name (or "macOS System" for
// com.apple.* bundle identifiers).
func GroupProcesses(procs []Process) []ProcessGroup {
	byPID := make(map[int]int, len(procs))
	for i, p := range procs {
		byPID[p.PID] = i
	}

	// Pass 1: direct heuristic matches
	heuristic := make(map[int]string, len(procs))
	for _, p := range procs {
		if hint := groupHint(p); hint != "" {
			heuristic[p.PID] = hint
		}
	}

	assignment := make(map[int]string, len(procs))

	// Pass 2: walk PPID chain, propagating only heuristic groups
	var resolveHeuristic func(idx, depth int) string
	resolveHeuristic = func(idx, depth int) string {
		p := procs[idx]
		if g, ok := assignment[p.PID]; ok {
			return g
		}
		if depth > 12 {
			return ""
		}
		if g, ok := heuristic[p.PID]; ok {
			assignment[p.PID] = g
			return g
		}
		if parentIdx, ok := byPID[p.PPID]; ok && p.PPID != p.PID {
			if g := resolveHeuristic(parentIdx, depth+1); g != "" {
				assignment[p.PID] = g
				return g
			}
		}
		return ""
	}

	for i := range procs {
		resolveHeuristic(i, 0)
	}

	// Pass 3: fallback for everything still unassigned
	for _, p := range procs {
		if _, ok := assignment[p.PID]; ok {
			continue
		}
		if strings.HasPrefix(p.Name, "com.apple.") {
			assignment[p.PID] = "macOS System"
		} else {
			assignment[p.PID] = p.Name
		}
	}

	groupMap := make(map[string]*ProcessGroup)
	for _, p := range procs {
		name := assignment[p.PID]
		if groupMap[name] == nil {
			groupMap[name] = &ProcessGroup{Name: name, SafetyClass: Critical}
		}
		g := groupMap[name]
		g.Members = append(g.Members, p)
		g.TotalRSSBytes += p.RSSBytes
		if p.SafetyClass < g.SafetyClass {
			g.SafetyClass = p.SafetyClass
		}
	}

	groups := make([]ProcessGroup, 0, len(groupMap))
	for _, g := range groupMap {
		sort.Slice(g.Members, func(i, j int) bool {
			return g.Members[i].RSSBytes > g.Members[j].RSSBytes
		})
		groups = append(groups, *g)
	}
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].TotalRSSBytes > groups[j].TotalRSSBytes
	})
	return groups
}
