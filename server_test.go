package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeStats is a StatsProvider that returns fixed data for tests
type fakeStats struct {
	mem      MemStats
	pressure string
	procs    []Process
}

func (f *fakeStats) MemStats() (MemStats, string, error) {
	return f.mem, f.pressure, nil
}

func (f *fakeStats) Processes() ([]Process, error) {
	return f.procs, nil
}

// fakeKiller records which PIDs were killed
type fakeKiller struct {
	killed []int
	err    error
}

func (f *fakeKiller) Kill(pid int) error {
	if f.err != nil {
		return f.err
	}
	f.killed = append(f.killed, pid)
	return nil
}

// fakeRestarter records restart commands
type fakeRestarter struct {
	restarted []string
	err       error
}

func (f *fakeRestarter) Restart(cmd string) error {
	if f.err != nil {
		return f.err
	}
	f.restarted = append(f.restarted, cmd)
	return nil
}

const chromaCmd = "/usr/local/bin/uvx chroma-mcp --host 0.0.0.0"

var defaultStats = &fakeStats{
	mem:      MemStats{UsedGB: 8.0, FreeGB: 2.0, TotalGB: 16.0},
	pressure: "Normal",
	procs: []Process{
		{PID: 100, Name: "claude", Command: "/home/.claude/versions/1.0/claude bg-spare", RSSBytes: 100 * 1024 * 1024, SafetyClass: SafeToKill},
		{PID: 200, Name: "npm", Command: "node /usr/bin/npm exec @mcp/server", RSSBytes: 50 * 1024 * 1024, SafetyClass: AutoRestart},
		{PID: 300, Name: "kernel_task", Command: "kernel_task", RSSBytes: 200 * 1024 * 1024, SafetyClass: Critical},
		{PID: 400, Name: "chroma-mcp", Command: chromaCmd, RSSBytes: 80 * 1024 * 1024, SafetyClass: AutoRestart},
	},
}

func TestStreamEndpoint_EmitsValidJSON(t *testing.T) {
	srv := NewServer(defaultStats, &fakeKiller{}, &fakeRestarter{})
	req := httptest.NewRequest(http.MethodGet, "/api/stream", nil)
	w := httptest.NewRecorder()

	srv.handleStream(w, req)

	body := w.Body.String()
	// SSE format: "data: {...}\n\n"
	if !strings.HasPrefix(body, "data: ") {
		t.Fatalf("body does not start with 'data: ', got: %q", body)
	}
	jsonPart := strings.TrimPrefix(strings.SplitN(body, "\n", 2)[0], "data: ")

	var event map[string]interface{}
	if err := json.Unmarshal([]byte(jsonPart), &event); err != nil {
		t.Fatalf("invalid JSON in SSE event: %v\nbody: %q", err, body)
	}
	for _, key := range []string{"used_gb", "free_gb", "total_gb", "pressure", "groups"} {
		if _, ok := event[key]; !ok {
			t.Errorf("SSE event missing field %q", key)
		}
	}
	groups, ok := event["groups"].([]interface{})
	if !ok || len(groups) == 0 {
		t.Errorf("expected non-empty groups, got %v", event["groups"])
	}
	// Total members across all groups should equal input process count
	totalMembers := 0
	for _, g := range groups {
		members := g.(map[string]interface{})["members"].([]interface{})
		totalMembers += len(members)
	}
	if totalMembers != 4 {
		t.Errorf("expected 4 members across all groups, got %d", totalMembers)
	}
}

func TestKillEndpoint_ValidPID_Returns200(t *testing.T) {
	killer := &fakeKiller{}
	srv := NewServer(defaultStats, killer, &fakeRestarter{})
	req := httptest.NewRequest(http.MethodPost, "/api/kill/100", nil)
	w := httptest.NewRecorder()

	srv.handleKill(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status: want 200, got %d", w.Code)
	}
	if len(killer.killed) != 1 || killer.killed[0] != 100 {
		t.Errorf("expected PID 100 killed, got %v", killer.killed)
	}
}

func TestKillEndpoint_UnknownPID_Returns404(t *testing.T) {
	srv := NewServer(defaultStats, &fakeKiller{}, &fakeRestarter{})
	req := httptest.NewRequest(http.MethodPost, "/api/kill/9999", nil)
	w := httptest.NewRecorder()

	srv.handleKill(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status: want 404, got %d", w.Code)
	}
}

func TestKillEndpoint_SystemProcess_Returns403(t *testing.T) {
	killer := &fakeKiller{}
	srv := NewServer(defaultStats, killer, &fakeRestarter{})
	req := httptest.NewRequest(http.MethodPost, "/api/kill/300", nil) // PID 300 = kernel_task (System)
	w := httptest.NewRecorder()

	srv.handleKill(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status: want 403, got %d", w.Code)
	}
	if len(killer.killed) != 0 {
		t.Errorf("system process should not have been killed, got kills: %v", killer.killed)
	}
}

func TestSSETick_CachesChromaCommand(t *testing.T) {
	srv := NewServer(defaultStats, &fakeKiller{}, &fakeRestarter{})
	req := httptest.NewRequest(http.MethodGet, "/api/stream", nil)
	w := httptest.NewRecorder()

	srv.handleStream(w, req)

	if srv.chromaCommand != chromaCmd {
		t.Errorf("chromaCommand: want %q, got %q", chromaCmd, srv.chromaCommand)
	}
}

func TestRestartChroma_WithCachedCommand_Returns200(t *testing.T) {
	restarter := &fakeRestarter{}
	srv := NewServer(defaultStats, &fakeKiller{}, restarter)
	// seed the cache via an SSE tick
	srv.handleStream(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/stream", nil))

	req := httptest.NewRequest(http.MethodPost, "/api/restart/chroma", nil)
	w := httptest.NewRecorder()
	srv.handleRestartChroma(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status: want 200, got %d", w.Code)
	}
	if len(restarter.restarted) != 1 || restarter.restarted[0] != chromaCmd {
		t.Errorf("Restart called with wrong command: %v", restarter.restarted)
	}
}

func TestRestartChroma_NoCachedCommand_Returns503(t *testing.T) {
	restarter := &fakeRestarter{}
	srv := NewServer(defaultStats, &fakeKiller{}, restarter)
	// no SSE tick — chromaCommand is empty

	req := httptest.NewRequest(http.MethodPost, "/api/restart/chroma", nil)
	w := httptest.NewRecorder()
	srv.handleRestartChroma(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status: want 503, got %d", w.Code)
	}
	if len(restarter.restarted) != 0 {
		t.Errorf("Restart should not have been called, got: %v", restarter.restarted)
	}
}

func TestRestartChroma_WorksAfterChromaKilled(t *testing.T) {
	restarter := &fakeRestarter{}
	statsWithoutChroma := &fakeStats{
		mem:      defaultStats.mem,
		pressure: defaultStats.pressure,
		procs:    defaultStats.procs[:3], // ChromaDB not in list (killed)
	}
	srv := NewServer(defaultStats, &fakeKiller{}, restarter)
	// cache command while ChromaDB was alive
	srv.handleStream(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/stream", nil))
	// now simulate ChromaDB being gone
	srv.stats = statsWithoutChroma

	req := httptest.NewRequest(http.MethodPost, "/api/restart/chroma", nil)
	w := httptest.NewRecorder()
	srv.handleRestartChroma(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status: want 200, got %d (cached command should survive kill)", w.Code)
	}
}

func TestSSEEvent_GroupsSortedByRAMDescending(t *testing.T) {
	unsorted := &fakeStats{
		mem:      MemStats{UsedGB: 8.0, FreeGB: 2.0, TotalGB: 16.0},
		pressure: "Normal",
		procs: []Process{
			{PID: 1, Name: "small", RSSBytes: 10 * 1024 * 1024, SafetyClass: Critical},
			{PID: 2, Name: "large", RSSBytes: 500 * 1024 * 1024, SafetyClass: Critical},
			{PID: 3, Name: "medium", RSSBytes: 100 * 1024 * 1024, SafetyClass: Critical},
		},
	}
	srv := NewServer(unsorted, &fakeKiller{}, &fakeRestarter{})
	req := httptest.NewRequest(http.MethodGet, "/api/stream", nil)
	w := httptest.NewRecorder()

	srv.handleStream(w, req)

	jsonPart := strings.TrimPrefix(strings.SplitN(w.Body.String(), "\n", 2)[0], "data: ")
	var event map[string]interface{}
	json.Unmarshal([]byte(jsonPart), &event)

	groups := event["groups"].([]interface{})
	first := groups[0].(map[string]interface{})["name"].(string)
	last := groups[len(groups)-1].(map[string]interface{})["name"].(string)

	if first != "large" || last != "small" {
		t.Errorf("want group order [large, medium, small], got first=%q last=%q", first, last)
	}
}

func TestStreamEndpoint_ContentType(t *testing.T) {
	srv := NewServer(defaultStats, &fakeKiller{}, &fakeRestarter{})
	req := httptest.NewRequest(http.MethodGet, "/api/stream", nil)
	w := httptest.NewRecorder()

	srv.handleStream(w, req)

	ct := w.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "text/event-stream") {
		t.Errorf("Content-Type: want text/event-stream, got %q", ct)
	}
}
