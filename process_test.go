package main

import "testing"

func TestAssignSafetyClass_BgSpareDaemon(t *testing.T) {
	cmd := "/Users/drdeeb/.nvm/versions/node/v22.0.0/bin/node /Users/drdeeb/.claude/versions/1.2.3/claude bg-spare"
	got := AssignSafetyClass("node", cmd)
	if got != SafeToKill {
		t.Errorf("expected SafeToKill, got %v", got)
	}
}

func TestAssignSafetyClass_MCPServer(t *testing.T) {
	cmd := "node /usr/local/bin/npm exec @modelcontextprotocol/server-filesystem"
	got := AssignSafetyClass("node", cmd)
	if got != AutoRestart {
		t.Errorf("expected AutoRestart, got %v", got)
	}
}

func TestAssignSafetyClass_ChromaDB(t *testing.T) {
	cmd := "/usr/local/bin/uvx chroma-mcp --host 0.0.0.0"
	got := AssignSafetyClass("uvx", cmd)
	if got != AutoRestart {
		t.Errorf("expected AutoRestart, got %v", got)
	}
}

func TestAssignSafetyClass_Critical_WindowServer(t *testing.T) {
	got := AssignSafetyClass("WindowServer", "/usr/sbin/WindowServer")
	if got != Critical {
		t.Errorf("WindowServer: expected Critical, got %v", got)
	}
}

func TestAssignSafetyClass_Critical_KernelTask(t *testing.T) {
	got := AssignSafetyClass("kernel_task", "kernel_task")
	if got != Critical {
		t.Errorf("kernel_task: expected Critical, got %v", got)
	}
}

func TestAssignSafetyClass_App_UserApp(t *testing.T) {
	got := AssignSafetyClass("Safari", "/Applications/Safari.app/Contents/MacOS/Safari")
	if got != App {
		t.Errorf("Safari: expected App, got %v", got)
	}
}

func TestAssignSafetyClass_App_UnknownFallback(t *testing.T) {
	// Unknown process with no rule and no com.apple.* prefix → App
	got := AssignSafetyClass("some-random-app", "/usr/local/bin/some-random-app")
	if got != App {
		t.Errorf("random app: expected App fallback, got %v", got)
	}
}

func TestAssignSafetyClass_Critical_AppleBundleFallback(t *testing.T) {
	// com.apple.* with no explicit rule → Critical
	got := AssignSafetyClass("com.apple.UnknownDaemon", "/System/Library/.../com.apple.UnknownDaemon")
	if got != Critical {
		t.Errorf("com.apple.* fallback: expected Critical, got %v", got)
	}
}
