# CONTEXT.md — ram-watch domain glossary

## Process
A running OS process identified by a PID. Characterized by RAM usage (RSS), CPU%, age, and command name. Every process belongs to exactly one **Safety Class**.

## Safety Class
The label assigned to a process determining what actions are available on it. One of:
- **safe-to-kill** — can be terminated without data loss or disruption
- **auto-restart** — safe to kill; will be respawned automatically by Claude or a service manager
- **system** — do not kill; owned by the OS or a critical app

## bg-spare daemon
A background spare Claude Code process matching `claude/versions` in its command path. Spawned automatically by Claude Code from previous sessions. Classified as **safe-to-kill**. Cannot be restarted manually — only Claude Code can spawn them. Kill-only action available.

## MCP server
A Model Context Protocol server running as an `npm exec` process. Supports Claude Code operations. Classified as **auto-restart** — Claude Code respawns them on the next session. Kill-only action available (with informational note about respawn).

## ChromaDB
The vector store process for the claude-mem plugin, identified by `chroma-mcp` in its command. Classified as **auto-restart**. Supports both Kill and Restart actions. The Restart action captures the full command string from `ps` before killing, then re-runs it.

## System process
Any process not matching the above categories. Classified as **system**. Read-only in the UI — no kill or restart actions.

## Memory pressure
The macOS system-wide memory pressure string returned by the `memory_pressure` command. Values: Normal, Warning, Critical. Displayed in the dashboard header alongside the RAM gauge.

## RAM gauge
The primary visual in the dashboard header. A 3D arc rendered with Three.js showing the RAM split across four segments: Active (blue), Wired (purple), Compressed (orange), Free (gray). Accompanied by stat numbers: used GB / free GB / total GB / pressure.

## Killable total
The sum of RAM held by all **safe-to-kill** and **auto-restart** processes. Shown in the dashboard summary as the maximum memory recoverable in one action.
