package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"
)

// OSStatsProvider implements StatsProvider using real macOS system calls.
type OSStatsProvider struct{}

func (o *OSStatsProvider) MemStats() (MemStats, string, error) {
	totalOut, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
	if err != nil {
		return MemStats{}, "", fmt.Errorf("sysctl: %w", err)
	}
	vmOut, err := exec.Command("vm_stat").Output()
	if err != nil {
		return MemStats{}, "", fmt.Errorf("vm_stat: %w", err)
	}
	stats, err := ParseMemoryStats(string(vmOut), string(totalOut))
	if err != nil {
		return MemStats{}, "", err
	}
	pressure := "?"
	if out, err := exec.Command("memory_pressure").Output(); err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			if strings.Contains(line, "System-wide") {
				fields := strings.Fields(line)
				if len(fields) > 0 {
					pressure = fields[len(fields)-1]
				}
				break
			}
		}
	}
	return stats, pressure, nil
}

func (o *OSStatsProvider) Processes() ([]Process, error) {
	out, err := exec.Command("ps", "-axm", "-o", "rss,pid,ppid,etime,comm,args").Output()
	if err != nil {
		return nil, fmt.Errorf("ps: %w", err)
	}
	return ParseProcessList(string(out)), nil
}

// OSProcessKiller implements ProcessKiller using SIGTERM.
type OSProcessKiller struct{}

func (k *OSProcessKiller) Kill(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return proc.Signal(syscall.SIGTERM)
}

// OSProcessRestarter implements ProcessRestarter by relaunching the captured command.
type OSProcessRestarter struct{}

func (r *OSProcessRestarter) Restart(command string) error {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return fmt.Errorf("empty command")
	}
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Start()
}

func main() {
	srv := NewServer(&OSStatsProvider{}, &OSProcessKiller{}, &OSProcessRestarter{})

	mux := http.NewServeMux()
	mux.HandleFunc("/api/stream", srv.handleStream)
	mux.HandleFunc("/api/kill/", srv.handleKill)
	mux.HandleFunc("/api/restart/chroma", srv.handleRestartChroma)
	mux.HandleFunc("/", srv.handleIndex)

	fmt.Println("ram-watch running at http://localhost:7734")
	go func() {
		time.Sleep(300 * time.Millisecond)
		openBrowser("http://localhost:7734")
	}()

	if err := http.ListenAndServe(":7734", mux); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}

func openBrowser(url string) {
	switch runtime.GOOS {
	case "darwin":
		exec.Command("open", url).Start()
	case "linux":
		exec.Command("xdg-open", url).Start()
	}
}
