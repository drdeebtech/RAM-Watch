# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

macOS-only RAM monitoring utility written in pure Bash. No build system, no dependencies, no tests. Single script: `ram-watch.sh`.

## Commands

```bash
# Install globally (copies to /opt/homebrew/bin/ram-watch)
bash install.sh

# Run
ram-watch              # one-shot snapshot
watch -n 5 ram-watch   # live refresh every 5s
```

## Architecture

`ram-watch.sh` is a single-file script with four logical sections:

1. **Memory stats** — queries `sysctl hw.memsize` for total RAM, parses `vm_stat` page counts (free/active/wired/compressor) to compute used/free GB, reads `memory_pressure` for system pressure string.

2. **Safe-to-kill: Claude bg-spare daemons** — processes matching `claude/versions` that are background spare instances from old sessions. Kill command: `pkill -f 'claude/versions.*bg-spare'`.

3. **Safe-to-kill: MCP servers** — processes matching `npm exec` (Claude's MCP servers). Auto-restart when Claude opens, so killing is safe. Kill command: `pkill -f 'npm exec'`.

4. **ChromaDB** — matched via `chroma-mcp`, used by the claude-mem plugin. Single PID shown with direct `kill` command.

5. **System processes** — top 15 by RSS from `ps -axm`, filtered to exclude Claude/npm/chroma/ram-watch itself.

6. **Summary** — totals all three killable categories and emits a one-liner to free them all.

## Key Implementation Details

- `PAGE=16384` — macOS ARM page size in bytes
- Memory math: `used = (active + wired + compressor) * PAGE`
- Process listing uses `ps -axm -o rss,pid,etime,comm` (RSS in KB, sorted descending)
- The script accepts `--kill-mcp` and `--kill-stale-claude` flags (defined in the shebang comment) but they are not yet implemented in the body
