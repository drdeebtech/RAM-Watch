#!/bin/bash
# ram-watch — full RAM breakdown with safety labels
# Usage: ram-watch [--kill-mcp] [--kill-stale-claude]

PAGE=16384
TOTAL=$(sysctl -n hw.memsize)
GB=1073741824

eval "$(vm_stat | awk -v p=$PAGE '
  /Pages free/                   { free=$3+0 }
  /Pages active/                 { active=$3+0 }
  /Pages wired down/             { wired=$4+0 }
  /Pages occupied by compressor/ { comp=$5+0 }
  END {
    used = (active + wired + comp) * p
    free_b = free * p
    printf "USED=%d FREE=%d\n", used, free_b
  }
')"

USED_GB=$(echo "scale=1; $USED / $GB" | bc)
FREE_GB=$(echo "scale=1; $FREE / $GB" | bc)
TOTAL_GB=$(echo "scale=1; $TOTAL / $GB" | bc)
PRES=$(memory_pressure 2>/dev/null | grep "System-wide" | awk '{print $NF}' || echo "?")

echo ""
echo "╔══════════════════════════════════════════════════╗"
printf  "║  RAM  %s GB used / %s GB free / %s GB total" "$USED_GB" "$FREE_GB" "$TOTAL_GB"
echo ""
printf  "║  Pressure: %-38s║\n" "$PRES"
echo "╚══════════════════════════════════════════════════╝"
echo ""

# ── SAFE TO KILL ──────────────────────────────────────
echo "🔴 SAFE TO KILL (stale / background Claude sessions)"
echo "   These are bg-spare daemons from old sessions. Killing them is safe."
echo "   Command to kill all at once:"
echo "   pkill -f 'claude/versions.*bg-spare'"
echo ""

CURRENT_CLAUDE_PID=$$
ps -axm -o rss,pid,etime,comm 2>/dev/null | grep "claude/versions" | sort -rn | \
  awk '{
    mb=$1/1024
    pid=$2
    age=$3
    printf "   kill %s   # %.0f MB  age=%s\n", pid, mb, age
  }'
echo ""

# ── MCP SERVERS — SAFE TO KILL ────────────────────────
echo "🟡 MCP SERVERS (restart automatically when Claude opens)"
echo "   Safe to kill — Claude will respawn them when needed."
echo "   Command to kill all at once:"
echo "   pkill -f 'npm exec'"
echo ""

ps -axm -o rss,pid,comm 2>/dev/null | grep "npm exec" | sort -rn | \
  awk '{
    mb=$1/1024; pid=$2
    # extract package name from command
    name=$0; gsub(/.*npm exec /, "", name); gsub(/@[^ ]*/, "", name)
    printf "   kill %-6s  # %5.0f MB  %s\n", pid, mb, name
  }'
echo ""

# ── CHROMA / PYTHON (claude-mem) ──────────────────────
CHROMA_PID=$(ps aux 2>/dev/null | grep "chroma-mcp" | grep -v grep | awk 'NR==1{print $2}')
CHROMA_MB=$(ps -o rss= -p "$CHROMA_PID" 2>/dev/null | awk '{printf "%.0f", $1/1024}')
if [ -n "$CHROMA_PID" ]; then
  echo "🟡 CHROMA DB (claude-mem vector store) — safe to kill, will restart"
  echo "   Running since boot, uses ~${CHROMA_MB:-0} MB"
  echo "   kill $CHROMA_PID"
  echo ""
fi

# ── SYSTEM / DO NOT KILL ──────────────────────────────
echo "🟢 DO NOT KILL (system processes)"
echo ""
ps -axm -o rss,pid,comm 2>/dev/null | sort -rn | \
  grep -v "claude\|npm exec\|chroma\|grep\|ram-watch\|awk\|sort" | \
  head -15 | awk 'NR>0 && $1>0 {
    mb=$1/1024; pid=$2; name=$3
    gsub(".*/","",name)
    printf "   %6.0f MB  %-30s PID=%s\n", mb, name, pid
  }'
echo ""

# ── TOTALS ────────────────────────────────────────────
CLAUDE_MB=$(ps -axm -o rss,comm 2>/dev/null | grep "claude/versions" | awk '{s+=$1} END {printf "%.0f", s/1024}')
MCP_MB=$(ps -axm -o rss,comm 2>/dev/null | grep "npm exec" | awk '{s+=$1} END {printf "%.0f", s/1024}')
CHROMA_MB2=$(ps -o rss= -p "$CHROMA_PID" 2>/dev/null | awk '{printf "%.0f", $1/1024}')
CLAUDE_COUNT=$(ps aux 2>/dev/null | grep -c "claude/versions")
MCP_COUNT=$(ps aux 2>/dev/null | grep -c "npm exec")

echo "── Summary ──────────────────────────────────────────"
printf "  Claude daemons : %2s processes  ~%s MB\n" "$CLAUDE_COUNT" "${CLAUDE_MB:-0}"
printf "  MCP servers    : %2s processes  ~%s MB\n" "$MCP_COUNT" "${MCP_MB:-0}"
printf "  ChromaDB       :  1 process    ~%s MB\n" "${CHROMA_MB2:-0}"
KILL_SAFE=$((${CLAUDE_MB:-0} + ${MCP_MB:-0} + ${CHROMA_MB2:-0}))
printf "  ─────────────────────────────────────────\n"
printf "  Killable total :               ~%s MB\n" "$KILL_SAFE"
echo ""
echo "  To free it all at once:"
CHROMA_KILL_CMD="pkill -f 'bg-spare' && pkill -f 'npm exec' && kill ${CHROMA_PID} 2>/dev/null"
echo "    $CHROMA_KILL_CMD"
echo "  (Claude Code will reopen them fresh on next session)"
echo ""
