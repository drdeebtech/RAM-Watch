package main

import (
	"bufio"
	"path/filepath"
	"strconv"
	"strings"
)

// ParseProcessList parses output from: ps -axm -o rss,pid,ppid,etime,comm,args
func ParseProcessList(psOutput string) []Process {
	var procs []Process
	scanner := bufio.NewScanner(strings.NewReader(psOutput))
	first := true
	for scanner.Scan() {
		line := scanner.Text()
		if first {
			first = false
			continue // skip header
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		rssKB, err := strconv.ParseInt(fields[0], 10, 64)
		if err != nil {
			continue
		}
		pid, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}
		ppid, _ := strconv.Atoi(fields[2])
		elapsed := fields[3]
		command := ""
		if len(fields) >= 6 {
			command = strings.Join(fields[5:], " ")
		}
		// use basename of the first args token for the name (comm truncates at 16 chars)
		name := filepath.Base(fields[4])
		if len(fields) >= 6 {
			name = filepath.Base(fields[5])
		}

		procs = append(procs, Process{
			PID:         pid,
			PPID:        ppid,
			Name:        name,
			Command:     command,
			RSSBytes:    rssKB * 1024,
			AgeSecs:     parseElapsed(elapsed),
			SafetyClass: AssignSafetyClass(name, command),
		})
	}
	return procs
}

// parseElapsed converts ps elapsed format [[DD-]HH:]MM:SS to seconds.
func parseElapsed(s string) int64 {
	var days, hours, mins, secs int64
	parts := strings.SplitN(s, "-", 2)
	if len(parts) == 2 {
		days, _ = strconv.ParseInt(parts[0], 10, 64)
		s = parts[1]
	}
	timeParts := strings.Split(s, ":")
	switch len(timeParts) {
	case 3:
		hours, _ = strconv.ParseInt(timeParts[0], 10, 64)
		mins, _ = strconv.ParseInt(timeParts[1], 10, 64)
		secs, _ = strconv.ParseInt(timeParts[2], 10, 64)
	case 2:
		mins, _ = strconv.ParseInt(timeParts[0], 10, 64)
		secs, _ = strconv.ParseInt(timeParts[1], 10, 64)
	}
	return days*86400 + hours*3600 + mins*60 + secs
}
