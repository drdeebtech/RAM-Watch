package main

import "testing"

// Sample output from: ps -axm -o rss,pid,ppid,etime,comm,args
const samplePS = `  RSS   PID  PPID     ELAPSED COMM            ARGS
104448   123     1    01:23:45 claude          /home/.claude/versions/1.0/claude bg-spare
 51200   456   123    00:05:00 npm             node /usr/bin/npm exec @mcp/server
204800   789     0  3-02:15:30 kernel_task     kernel_task
 81920   101     1    00:30:00 uvx             /usr/local/bin/uvx chroma-mcp --host 0.0.0.0
`

func TestParseProcessList_Count(t *testing.T) {
	procs := ParseProcessList(samplePS)
	if len(procs) != 4 {
		t.Errorf("want 4 processes, got %d", len(procs))
	}
}

func TestParseProcessList_RSSConvertedToBytes(t *testing.T) {
	procs := ParseProcessList(samplePS)
	// claude row: 104448 KB → 106954752 bytes
	want := int64(104448 * 1024)
	if procs[0].RSSBytes != want {
		t.Errorf("RSSBytes: want %d, got %d", want, procs[0].RSSBytes)
	}
}

func TestParseProcessList_PIDParsed(t *testing.T) {
	procs := ParseProcessList(samplePS)
	if procs[0].PID != 123 {
		t.Errorf("PID: want 123, got %d", procs[0].PID)
	}
}

func TestParseProcessList_PPIDParsed(t *testing.T) {
	procs := ParseProcessList(samplePS)
	// row 1 (npm) has PPID 123 → child of claude
	if procs[1].PPID != 123 {
		t.Errorf("PPID: want 123, got %d", procs[1].PPID)
	}
	if procs[0].PPID != 1 {
		t.Errorf("PPID for row 0: want 1, got %d", procs[0].PPID)
	}
}

func TestParseProcessList_SafetyClassAssigned(t *testing.T) {
	procs := ParseProcessList(samplePS)
	cases := []struct {
		idx  int
		want SafetyClass
	}{
		{0, SafeToKill},  // claude bg-spare
		{1, AutoRestart}, // npm exec
		{2, System},      // kernel_task
		{3, AutoRestart}, // chroma-mcp
	}
	for _, c := range cases {
		if procs[c.idx].SafetyClass != c.want {
			t.Errorf("procs[%d].SafetyClass: want %v, got %v", c.idx, c.want, procs[c.idx].SafetyClass)
		}
	}
}

func TestParseProcessList_ElapsedParsed(t *testing.T) {
	procs := ParseProcessList(samplePS)
	cases := []struct {
		idx     int
		elapsed string
		want    int64
	}{
		{0, "01:23:45", 1*3600 + 23*60 + 45},    // HH:MM:SS
		{1, "00:05:00", 5 * 60},                   // HH:MM:SS zeros
		{2, "3-02:15:30", 3*86400 + 2*3600 + 15*60 + 30}, // DD-HH:MM:SS
	}
	for _, c := range cases {
		if procs[c.idx].AgeSecs != c.want {
			t.Errorf("procs[%d] elapsed %q: want %d, got %d", c.idx, c.elapsed, c.want, procs[c.idx].AgeSecs)
		}
	}
}
