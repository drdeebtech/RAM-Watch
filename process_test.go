package main

import "testing"

func TestAssignSafetyClass_BgSpareDaemon(t *testing.T) {
	cmd := "/Users/drdeeb/.nvm/versions/node/v22.0.0/bin/node /Users/drdeeb/.claude/versions/1.2.3/claude bg-spare"
	got := AssignSafetyClass(cmd)
	if got != SafeToKill {
		t.Errorf("expected SafeToKill, got %v", got)
	}
}

func TestAssignSafetyClass_MCPServer(t *testing.T) {
	cmd := "node /usr/local/bin/npm exec @modelcontextprotocol/server-filesystem"
	got := AssignSafetyClass(cmd)
	if got != AutoRestart {
		t.Errorf("expected AutoRestart, got %v", got)
	}
}

func TestAssignSafetyClass_ChromaDB(t *testing.T) {
	cmd := "/usr/local/bin/uvx chroma-mcp --host 0.0.0.0"
	got := AssignSafetyClass(cmd)
	if got != AutoRestart {
		t.Errorf("expected AutoRestart, got %v", got)
	}
}

func TestAssignSafetyClass_SystemProcess(t *testing.T) {
	for _, cmd := range []string{"kernel_task", "/usr/sbin/WindowServer", "Slack"} {
		got := AssignSafetyClass(cmd)
		if got != System {
			t.Errorf("cmd=%q: expected System, got %v", cmd, got)
		}
	}
}
