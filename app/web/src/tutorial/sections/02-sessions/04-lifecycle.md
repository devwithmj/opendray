# Session lifecycle

Each session walks through a small state machine. The state pill
at the top of the terminal pane tells you exactly where it is.

| State | Meaning | What you can do |
|---|---|---|
| **STARTING** | DB row inserted, PTY spawning | wait — usually <500ms |
| **RUNNING** | Process alive, recent stdout activity | type into the terminal |
| **IDLE** | Process alive, silent for ≥30s (configurable) | reply normally; idle is just an event signal |
| **STOPPED** | Operator hit ✕ → SIGTERM → process exited | view scrollback; **Restart** spawns under same id |
| **ENDED** | Process exited on its own | view scrollback; **Restart** to relaunch |
| **FAILED** | Spawn or runtime error before clean exit | check dialog error / log; usually unrecoverable without a config fix |

## Going idle

Idle is just "no stdout for N seconds" — the process is still
alive and listening. opendray uses idle as a signal to fire
`session.idle` on the event bus, which:

- [Channels](#channels-overview) push notifications based on
  their `notify_on` filter
- The state pill shows `IDLE` (yellow)
- The next byte the CLI emits flips state back to `RUNNING`

The threshold (default 30s) and the watcher poll interval
(default 5s) live in `[session]` in `config.toml`. Lower threshold
= more notifications, higher threshold = miss short pauses.

> Channels have their own per-channel notification policy that
> layers on top — see the *Notifications panel deep-dive* section
> under Channels for `once` / `cooldown` / `every` modes.

## Stopping a session

Three ways a session leaves `RUNNING`:

### Operator stop (× button)

Click the × on a running tab → confirm dialog → opendray sends
**SIGTERM**, waits 3 seconds, then **SIGKILL** if the process is
still alive. State flips to `STOPPED`, ring buffer is preserved
for read-back.

### Process exit

The CLI exits on its own — `q` / `Ctrl-D` / a script's `exit 0`
/ a panic. opendray captures the exit code, marks the row
`ENDED` (with `exit_code` populated), publishes
`session.ended` on the event bus.

If `exit_code != 0`, channels rendering session.ended cards show
the **red** colour template.

### Reconcile on opendray restart

When opendray itself restarts (new build, host reboot) any rows
that were in non-terminal state get marked `ENDED` with reason
`"previous gateway process exited; PTYs gone"`. The PTY couldn't
survive a parent process death, so the row is honest about it.

You see this as the log line on startup:

```
INFO reconciled stale sessions on startup count=N
```

## Restart from a stopped session

The Restart button (visible on stopped/ended tabs) re-runs the
spawn flow with the same:

- Provider id
- Working directory
- Args
- Claude account binding
- Parent session id

But assigns a **new session id**. The old row stays in the DB
for audit. The Inspector linked note travels to the new session
because it's keyed by file path, not session id.

## Closing tabs

The × button on an **ended** tab closes the tab visually but
**keeps the DB row**. You can find old sessions via the *History*
filter at the top of the Sessions list (TODO: confirm exact UI
location with the operator).

To truly delete a session row from the database, use the API:

```bash
curl -X DELETE -H "Authorization: Bearer $ADMIN_TOKEN" \
  http://localhost:8770/api/v1/sessions/<sid>
```

The web admin doesn't expose a destructive delete button on
purpose — accidental clicks would lose audit context.
