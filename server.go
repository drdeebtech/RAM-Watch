package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type Server struct {
	stats         StatsProvider
	killer        ProcessKiller
	restarter     ProcessRestarter
	chromaCommand string
}

func NewServer(stats StatsProvider, killer ProcessKiller, restarter ProcessRestarter) *Server {
	return &Server{stats: stats, killer: killer, restarter: restarter}
}

type streamEvent struct {
	UsedGB       float64           `json:"used_gb"`
	FreeGB       float64           `json:"free_gb"`
	TotalGB      float64           `json:"total_gb"`
	ActiveGB     float64           `json:"active_gb"`
	WiredGB      float64           `json:"wired_gb"`
	CompressedGB float64           `json:"compressed_gb"`
	Pressure     string            `json:"pressure"`
	Groups       []processGroupDTO `json:"groups"`
}

type processDTO struct {
	PID         int     `json:"pid"`
	Name        string  `json:"name"`
	Command     string  `json:"command"`
	RAMMB       float64 `json:"ram_mb"`
	CPUPercent  float64 `json:"cpu_pct"`
	AgeSecs     int64   `json:"age_secs"`
	SafetyClass string  `json:"safety_class"`
	Description string  `json:"description"`
	ParentName  string  `json:"parent_name"`
}

type processGroupDTO struct {
	Name        string       `json:"name"`
	TotalMB     float64      `json:"total_mb"`
	SafetyClass string       `json:"safety_class"`
	Members     []processDTO `json:"members"`
}

func safetyLabel(sc SafetyClass) string {
	switch sc {
	case SafeToKill:
		return "safe-to-kill"
	case AutoRestart:
		return "auto-restart"
	case App:
		return "app"
	case Critical:
		return "critical"
	}
	return "critical"
}

func (s *Server) buildEvent() (*streamEvent, error) {
	mem, pressure, err := s.stats.MemStats()
	if err != nil {
		return nil, err
	}
	procs, err := s.stats.Processes()
	if err != nil {
		return nil, err
	}

	for _, p := range procs {
		if strings.Contains(p.Command, "chroma-mcp") && p.Command != "" {
			s.chromaCommand = p.Command
		}
	}

	// Index processes by PID so we can look up parents for descriptions.
	procByPID := make(map[int]*Process, len(procs))
	for i := range procs {
		procByPID[procs[i].PID] = &procs[i]
	}

	groups := GroupProcesses(procs)
	groupDTOs := make([]processGroupDTO, len(groups))
	for i, g := range groups {
		members := make([]processDTO, len(g.Members))
		for j, p := range g.Members {
			parent := procByPID[p.PPID]
			parentName := ""
			if parent != nil {
				parentName = parent.Name
			}
			members[j] = processDTO{
				PID:         p.PID,
				Name:        p.Name,
				Command:     p.Command,
				RAMMB:       float64(p.RSSBytes) / (1024 * 1024),
				CPUPercent:  p.CPUPercent,
				AgeSecs:     p.AgeSecs,
				SafetyClass: safetyLabel(p.SafetyClass),
				Description: DescribeProcess(p, parent),
				ParentName:  parentName,
			}
		}
		groupDTOs[i] = processGroupDTO{
			Name:        g.Name,
			TotalMB:     float64(g.TotalRSSBytes) / (1024 * 1024),
			SafetyClass: safetyLabel(g.SafetyClass),
			Members:     members,
		}
	}

	return &streamEvent{
		UsedGB:       mem.UsedGB,
		FreeGB:       mem.FreeGB,
		TotalGB:      mem.TotalGB,
		ActiveGB:     mem.ActiveGB,
		WiredGB:      mem.WiredGB,
		CompressedGB: mem.CompressedGB,
		Pressure:     pressure,
		Groups:       groupDTOs,
	}, nil
}

func (s *Server) handleKill(w http.ResponseWriter, r *http.Request) {
	pidStr := strings.TrimPrefix(r.URL.Path, "/api/kill/")
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		http.Error(w, "invalid pid", http.StatusBadRequest)
		return
	}

	procs, err := s.stats.Processes()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var target *Process
	for i := range procs {
		if procs[i].PID == pid {
			target = &procs[i]
			break
		}
	}
	if target == nil {
		http.Error(w, "process not found", http.StatusNotFound)
		return
	}
	if target.SafetyClass == Critical {
		http.Error(w, "cannot kill critical system process", http.StatusForbidden)
		return
	}

	if err := s.killer.Kill(pid); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleRestartChroma(w http.ResponseWriter, r *http.Request) {
	if s.chromaCommand == "" {
		http.Error(w, "no ChromaDB command cached — has ChromaDB run this session?", http.StatusServiceUnavailable)
		return
	}
	if err := s.restarter.Restart(s.chromaCommand); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleStream(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	event, err := s.buildEvent()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data, _ := json.Marshal(event)
	fmt.Fprintf(w, "data: %s\n\n", data)
}
