# Note editor

The note editor is the heart of the Notes page (and the Notes
tab in the Sessions Inspector). It's a paired textarea +
preview, with auto-save and the wiki-link suggestion popup.

## Source vs Preview

Tabs at the top of every note view:

- **Source** — raw markdown, plain textarea. Type, paste, edit.
- **Preview** — rendered output via react-markdown +
  remark-gfm: tables, code fences with hljs syntax highlighting,
  task list checkboxes (clickable!), wiki-links rendered as
  pill buttons.

Two states pivot based on context:

- **Per-session linked notes** open in **Source** by default —
  you're scratchpad'ing.
- **Standalone vault notes** opened from the Notes page open in
  **Preview** by default — you're reading project docs.

Override with the `?mode=source` / `?mode=preview` URL
parameter, or just click the tab.

![Source / Preview tabs](/tutorial/notes-source-preview.png)

## Auto-save

Edits debounce 1 second after the last keystroke, then save to
disk. The status indicator next to the title shows:

| Indicator | Meaning |
|---|---|
| `Saved` | latest content is on disk |
| `Saving…` | write in flight |
| `Unsaved` | edits pending; debounce timer active |
| `Save error` | last write failed (permissions, disk full); content stays in the textarea |

If the host crashes mid-save, you lose at most the last second
of typing. The vault git sync (if enabled) gives you a 5-minute
crash budget on top.

## Keyboard shortcuts

| Shortcut | Action |
|---|---|
| `Cmd / Ctrl + S` | Force save now (skip debounce) |
| `Cmd / Ctrl + B` | Wrap selection in `**bold**` |
| `Cmd / Ctrl + I` | Wrap selection in `*italic*` |
| `Cmd / Ctrl + K` | Wrap selection in `[selection](url)` |
| `Tab` (in Source mode) | Insert literal tab (does NOT shift focus) |
| `Esc` | Close any open suggestion popup |

The shortcuts are a tiny subset of what a full editor would
provide — opendray's note editor is intentionally minimal. If
you want power features (multi-cursor, snippets, vim mode), the
vault is just a directory of `.md` files; open it in Obsidian
or VS Code.

## Linked-note context

When opened from the Sessions Inspector, the editor knows it's
the linked note and shows:

- **Session id** below the title
- **"Open standalone"** link to navigate to the note in the
  full Notes page
- **Backlinks** show all the places the note is referenced from

Closing the session tab does NOT close the note — it stays in
the vault under `sessions/<sid>.md`. You can re-open it any
time, even after the session itself has ended.

## File operations

The Notes page sidebar has a tree of every directory + .md file
in the vault. Right-click for:

- **New note** in this directory
- **Rename** the file (auto-updates wiki-links pointing at it)
- **Delete** with confirmation
- **Move** between directories (drag, or right-click → Move to)

opendray atomically renames + updates references via a single
worktree commit — if you have vault git sync enabled, the next
sync cycle includes the rename as one commit not two.
