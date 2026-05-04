# Sessions — overview

The Sessions page is opendray's daily-driver workbench. Every CLI
process opendray manages — Claude Code, Codex, Gemini, plain shell
— shows up here as one tab. You spawn, watch, drive, and clean up
sessions all in this one page.

Read the deep-dive sections below for any specific workflow:

| Topic | Section |
|---|---|
| Spawning a new CLI session | Spawning a session |
| The right-side Inspector panel | Inspector panel |
| What state pills mean (running/idle/ended/...) | Session lifecycle |
| Multi-tab management + keyboard shortcuts | Tabs & keyboard nav |

## At a glance

![Sessions page layout](/tutorial/sessions-layout.png)

1. **Tab strip (top)** — every running and recently-ended session.
2. **Terminal** — full xterm.js, copy-paste works, mouse scroll
   honours the provider's `mouseEvents` setting.
3. **Status pill** — `RUNNING` / `IDLE` / `ENDED` / `STOPPED` plus
   exit code when terminal.
4. **Inspector (right side)** — collapsible side panel with
   per-session sub-tabs.

## Why a "session" not a "process"

opendray models each CLI invocation as a **session row in the
database**. The PTY + child process come and go (you can stop and
restart a session — the row id survives), but the session record
keeps the working directory, provider, args, parent session id,
and bound Claude account.

This is what makes "restart in place" possible: when a Claude
session crashes, you can spawn a new one under the same id, same
cwd, same account binding, and the same Inspector tabs (linked
note, outline cache) come back.

## Where data lives

| Data | Location |
|---|---|
| Session metadata (id, cwd, args, state) | Postgres `sessions` table |
| Stdout history while running | In-memory ring buffer (1 MiB per session) |
| Per-session Inspector linked note | Notes vault file `<vault>/sessions/<sid>.md` |
| Claude transcript (assistant turns) | `~/.claude/projects/<encoded-cwd>/<sid>.jsonl` (Claude's own) |

The web UI streams stdout via WebSocket; closing your browser tab
doesn't kill the session — opendray keeps the ring buffer + the
process running, ready to replay history when you reconnect.
