# Inspector panel

The Inspector is the collapsible right-side panel on the Sessions
page. It carries metadata, linked notes, and tooling that doesn't
fit inline in the terminal.

![Inspector panel tabs](/tutorial/sessions-inspector.png)

Toggle the panel with the keyboard shortcut **`g i`** or the icon
in the top-right of the terminal pane.

## Sub-tabs

### Outline

Auto-extracted markdown headers from the active session's most-
recent assistant message (Claude only — pulls from the JSONL
transcript) or from any markdown block in the live ring buffer
output (other CLIs).

Click a heading → terminal jumps to its location in the
scrollback. Useful when Claude has written a long structured
response and you want to skim.

### Notes

The session's linked Obsidian note. Each session gets one note at
`<vault-root>/sessions/<session-id>.md` automatically. The
Inspector embeds the same Markdown editor you'd see on the Notes
page:

- **Source / Preview** tabs at the top of the note pane — pick
  whichever matches the moment.
- **Wiki-link suggestions** trigger when you type `[[`.
- **Backlinks** show on the right (other notes that link to this
  one).
- **Auto-save** debounced 1s after the last keystroke.

This is the right place for the operator's running scratchpad:
"things to ask Claude about", "pending decisions", "TODO before
ending the session". Survives session restart because it's
file-based, not in-memory.

### Context

A scrollable tree of the session's working directory. Helpful when
you've forgotten what files are in the project, or when Claude
references a file you want to peek at without leaving the terminal.

Click a file → opens a read-only viewer in the Inspector.

The tree is collapsed by default to 3 levels deep so a `node_modules`
or `.venv` doesn't blow it up; expand specific subtrees on demand.

### Activity

Per-session live event feed. Filtered to the active session id
automatically — you see only events that mention this session.

What lands here:

- `session.idle` / `session.ended` lifecycle events
- `channel.message_sent` — when a notification fires for this
  session on any channel
- `channel.message_forwarded` — when a Telegram (etc.) reply gets
  routed to this session's stdin
- Any custom topic an integration publishes

Useful when debugging: "did the notification go out?" — check
Activity here, see the `channel.message_sent` line, confirm.

## Hiding the inspector

Two options:

- **Toggle** with `g i` to slide the panel off-screen (state per-
  user, persists across reloads).
- **Collapse a single tab** by clicking its tab pill twice.

When opened, the panel takes ~360px on the right. On narrow
windows (<1200px) it overlays the terminal instead of side-by-side.
