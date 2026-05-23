# ram-watch

macOS RAM monitor — shows everything using memory, what's safe to kill, and exact kill commands.

## Install

```bash
bash install.sh
```

## Usage

```bash
ram-watch              # one-shot snapshot
watch -n 5 ram-watch   # live, refreshes every 5s
```

## What it shows

- Total RAM used / free
- 🔴 Safe-to-kill Claude background sessions (with PIDs + MB each)
- 🟡 MCP servers (auto-restart when Claude opens)
- 🟡 ChromaDB vector store
- 🟢 System processes — do not kill
- One-liner to free everything at once
