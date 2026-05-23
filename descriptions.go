package main

import (
	"fmt"
	"strings"
)

// processRule pairs a matcher with the resulting safety class and a
// human-readable description.
type processRule struct {
	Match       func(name, cmd string) bool
	Safety      SafetyClass
	Description string
}

// processRules — checked in order; first match wins.
// Command-pattern rules go FIRST so they beat bare-name rules
// (e.g., `node` running `npm exec` is auto-restart, not a bare Node app).
var processRules = []processRule{
	// ── Command-pattern rules (specific behaviors) ────────────────────────────
	{
		Match:       func(n, c string) bool { return strings.Contains(c, "bg-spare") && strings.Contains(c, "claude") },
		Safety:      SafeToKill,
		Description: "Claude background spare daemon — orphan from an old session. Safe to clean up.",
	},
	{
		Match:       func(n, c string) bool { return strings.Contains(c, "npm exec") || strings.Contains(c, "@modelcontextprotocol") },
		Safety:      AutoRestart,
		Description: "Claude MCP server (Model Context Protocol). Spawned by your Claude session; respawns on demand.",
	},
	{
		Match:       func(n, c string) bool { return strings.Contains(c, "chroma-mcp") || strings.Contains(c, "chroma_mcp") },
		Safety:      AutoRestart,
		Description: "ChromaDB vector store. Used by the claude-mem plugin; will be re-launched on next use.",
	},
	{
		Match:       func(n, c string) bool { return strings.Contains(c, "claude/versions") },
		Safety:      App,
		Description: "Claude Code CLI session. Killing this ends your active conversation.",
	},

	// ── CRITICAL: system daemons & kernel ─────────────────────────────────────
	{Match: nameIs("WindowServer"), Safety: Critical, Description: "macOS display server. Manages every window on screen and the GPU compositor. Killing it logs you out."},
	{Match: nameIs("launchd"), Safety: Critical, Description: "macOS init process (PID 1). Parent of all user-space processes. Cannot be killed."},
	{Match: nameIs("kernel_task"), Safety: Critical, Description: "macOS kernel scheduler. Surfaces kernel-mode CPU/IO. Cannot be killed."},
	{Match: nameIs("loginwindow"), Safety: Critical, Description: "Login session manager. Killing it logs you out immediately."},
	{Match: nameIs("Dock"), Safety: Critical, Description: "macOS Dock. Killing it removes the dock until restart."},
	{Match: nameIs("SystemUIServer"), Safety: Critical, Description: "Menu bar / status item host. Killing it drops your menu extras."},
	{Match: nameIs("ControlCenter"), Safety: Critical, Description: "macOS Control Center (menu bar shortcuts: WiFi, Bluetooth, volume)."},
	{Match: nameIs("WindowManager"), Safety: Critical, Description: "Stage Manager and window tiling system."},
	{Match: nameIs("NotificationCenter"), Safety: Critical, Description: "macOS notification system. Killing it stops delivery of all notifications."},
	{Match: nameIs("UIKitSystem"), Safety: Critical, Description: "UIKit support process for iPad apps on macOS (Catalyst)."},
	{Match: nameIs("WallpaperAgent"), Safety: Critical, Description: "Desktop wallpaper renderer. Killing it leaves a blank desktop."},
	{Match: nameIs("coreaudiod"), Safety: Critical, Description: "Core Audio daemon. Killing it silences all audio system-wide."},
	{Match: nameIs("audioaccessoryd"), Safety: Critical, Description: "Manages audio accessories (AirPods, USB audio)."},
	{Match: nameIs("mds"), Safety: Critical, Description: "Spotlight metadata server (master). Coordinates all file indexing."},
	{Match: nameIs("mds_stores"), Safety: Critical, Description: "Spotlight index storage. Holds the searchable file database."},
	{Match: nameIs("mdworker_shared"), Safety: Critical, Description: "Spotlight worker. Crawls and indexes files in the background."},
	{Match: nameIs("corespotlightd"), Safety: Critical, Description: "CoreSpotlight indexer (app content search). Required for in-app search."},
	{Match: nameIs("securityd"), Safety: Critical, Description: "Security framework daemon. Handles keychain, code signing, sandbox."},
	{Match: nameIs("syspolicyd"), Safety: Critical, Description: "System policy / Gatekeeper enforcement."},
	{Match: nameIs("tccd"), Safety: Critical, Description: "Transparency, Consent, and Control daemon. Manages privacy permissions."},
	{Match: nameIs("apsd"), Safety: Critical, Description: "Apple Push Notification Service daemon."},
	{Match: nameIs("configd"), Safety: Critical, Description: "System configuration daemon. Manages network and host configuration."},
	{Match: nameIs("opendirectoryd"), Safety: Critical, Description: "Directory services daemon. User accounts and authentication."},
	{Match: nameIs("bluetoothd"), Safety: Critical, Description: "Bluetooth daemon. Killing it disconnects all Bluetooth devices."},
	{Match: nameIs("locationd"), Safety: Critical, Description: "Location services. Required by Find My, weather, time zone."},
	{Match: nameIs("airportd"), Safety: Critical, Description: "Wi-Fi (AirPort) management daemon."},
	{Match: nameIs("mDNSResponder"), Safety: Critical, Description: "Bonjour / mDNS daemon. Resolves *.local domains, enables AirPlay."},
	{Match: nameIs("runningboardd"), Safety: Critical, Description: "App lifecycle manager. Decides which processes get suspended."},
	{Match: nameIs("launchservicesd"), Safety: Critical, Description: "Maps file types to applications and handles 'open' commands."},
	{Match: nameIs("powerd"), Safety: Critical, Description: "Power management daemon. Sleep, wake, battery, thermal."},
	{Match: nameIs("thermalmonitord"), Safety: Critical, Description: "Thermal management. Throttles CPU/GPU when hot."},
	{Match: nameIs("watchdogd"), Safety: Critical, Description: "System watchdog. Detects and recovers from process hangs."},
	{Match: nameIs("corebrightnessd"), Safety: Critical, Description: "Display brightness control daemon."},
	{Match: nameIs("fseventsd"), Safety: Critical, Description: "Filesystem events daemon. Powers folder-change notifications across apps."},
	{Match: nameIs("diskarbitrationd"), Safety: Critical, Description: "Disk mount/unmount manager."},
	{Match: nameIs("backupd"), Safety: Critical, Description: "Time Machine backup daemon."},
	{Match: nameIs("logd"), Safety: Critical, Description: "Unified logging daemon. Aggregates all system logs."},
	{Match: nameIs("distnoted"), Safety: Critical, Description: "Distributed notification center. Inter-process messaging."},
	{Match: nameIs("notifyd"), Safety: Critical, Description: "Posix notification daemon. Low-level system signals."},

	// ── APP: user-facing applications & dev tooling ───────────────────────────
	{Match: nameIs("Safari"), Safety: App, Description: "Safari web browser (main process). Killing it closes Safari and loses unsaved form input."},
	{Match: prefixedName("com.apple.WebKit.WebContent"), Safety: App, Description: "Safari/WebKit browser tab. One process per tab. Killing it closes that tab."},
	{Match: prefixedName("com.apple.WebKit.GPU"), Safety: App, Description: "Safari/WebKit GPU process. Killing it forces Safari to re-render."},
	{Match: prefixedName("com.apple.WebKit.Networking"), Safety: App, Description: "Safari/WebKit network process. Handles HTTP for all tabs."},
	{Match: prefixedName("com.apple.WebKit"), Safety: App, Description: "WebKit helper process (Safari or web-using app)."},
	{Match: nameIs("Finder"), Safety: App, Description: "macOS file manager. Killing it just relaunches; no harm."},
	{Match: nameIs("Terminal"), Safety: App, Description: "macOS Terminal app. Killing it terminates all open shells inside."},
	{Match: nameIs("Cursor"), Safety: App, Description: "Cursor IDE (Electron app)."},
	{Match: func(n, c string) bool { return strings.Contains(c, "/Cursor.app/") }, Safety: App, Description: "Cursor IDE helper process (Electron renderer or GPU)."},
	{Match: nameIs("Electron"), Safety: App, Description: "Electron-based application process."},
	{Match: nameIs("Notes"), Safety: App, Description: "Apple Notes app."},
	{Match: nameIs("Messages"), Safety: App, Description: "Apple Messages app. Killing it disconnects iMessage."},
	{Match: nameIs("Mail"), Safety: App, Description: "Apple Mail app."},
	{Match: nameIs("Slack"), Safety: App, Description: "Slack desktop app."},
	{Match: nameIs("Grammarly"), Safety: App, Description: "Grammarly desktop helper / browser extension host."},
	{Match: prefixedName("Grammarly"), Safety: App, Description: "Grammarly helper process."},
	{Match: nameIs("Google Chrome"), Safety: App, Description: "Google Chrome web browser."},
	{Match: func(n, c string) bool { return strings.Contains(c, "Google Chrome") }, Safety: App, Description: "Google Chrome helper process (renderer, GPU, network)."},
	{Match: nameIs("claude"), Safety: App, Description: "Claude Code CLI session. Killing it ends your active conversation."},
	{Match: nameIs("bun"), Safety: App, Description: "Bun JavaScript runtime. Likely a long-running dev tool or script."},
	{Match: nameIs("node"), Safety: App, Description: "Node.js runtime. Likely a dev server, build tool, or script."},
	{Match: nameIs("python"), Safety: App, Description: "Python runtime. Likely a long-running script, server, or dev tool."},
	{Match: nameIs("uvx"), Safety: App, Description: "UV-managed Python tool runner."},
	{Match: nameIs("uv"), Safety: App, Description: "UV Python package manager / runner."},
	{Match: nameIs("mariadbd"), Safety: App, Description: "MariaDB database server. Killing it drops active connections."},
	{Match: nameIs("Spotlight"), Safety: App, Description: "Spotlight search UI. Killing it just closes the search panel."},
}

// matchProcessRule returns the first rule that matches, or nil if none does.
func matchProcessRule(name, cmd string) *processRule {
	for i := range processRules {
		if processRules[i].Match(name, cmd) {
			return &processRules[i]
		}
	}
	return nil
}

// nameIs returns a matcher that compares process name exactly.
func nameIs(target string) func(name, cmd string) bool {
	return func(name, cmd string) bool { return name == target }
}

// prefixedName returns a matcher that checks for a process name prefix.
func prefixedName(prefix string) func(name, cmd string) bool {
	return func(name, cmd string) bool { return strings.HasPrefix(name, prefix) }
}

// DescribeProcess returns a human-readable description of a process.
// If a known rule matches, returns its description. Otherwise generates a
// fallback from the parent context, age, and command.
func DescribeProcess(p Process, parent *Process) string {
	if rule := matchProcessRule(p.Name, p.Command); rule != nil {
		return rule.Description
	}
	parentName := "system (launchd)"
	if parent != nil {
		parentName = fmt.Sprintf("%s (PID %d)", parent.Name, parent.PID)
	}
	if strings.HasPrefix(p.Name, "com.apple.") {
		return fmt.Sprintf("Apple system service. Started by %s.", parentName)
	}
	return fmt.Sprintf("Unknown process. Started by %s.", parentName)
}
