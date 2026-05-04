# Tabs & keyboard nav

The Sessions page is the heaviest user of keyboard shortcuts in
the admin. Once you know them, you can run a dozen CLIs without
ever touching the mouse.

## Tab strip

![Multi-tab strip](/tutorial/sessions-tab-strip.png)

- **Click a tab** to switch.
- **Click ×** on a running tab → confirms before stopping.
- **Click ×** on a stopped/ended tab → closes visually (row
  stays in DB).
- **Drag a tab** to reorder (state persists per-user).
- **Right-click a tab** opens context menu: Restart, Rename,
  Close.

Long names get ellipsised in the middle (`my-project-foo-…-feat-x`)
so the trailing context (`-feat-x`) stays readable.

## Keyboard shortcuts

### Navigation between sessions

| Shortcut | Action |
|---|---|
| `g s` | Jump to Sessions page |
| `Ctrl + Tab` | Next tab |
| `Ctrl + Shift + Tab` | Previous tab |
| `Ctrl + 1` … `Ctrl + 9` | Jump to tab N |
| `Ctrl + W` | Close current tab (confirms if running) |

`g`-prefixed shortcuts are vim-style — press `g`, then the second
key within ~1.5s. The status bar shows a tiny breadcrumb when `g`
is pending so you know it registered.

### Inside the terminal

xterm.js handles keystrokes directly. opendray's only special
intercept is **`Ctrl + Shift + ↑/↓`** for scrollback bypass — see
your provider's docs for everything else (Claude has its own
`Ctrl-G`-prefixed cycle for permission modes etc.).

### Inspector

| Shortcut | Action |
|---|---|
| `g i` | Toggle Inspector panel |
| `1` / `2` / `3` / `4` (with Inspector focused) | Switch between Outline / Notes / Context / Activity tabs |

Focus the Inspector by clicking inside it once, then the digit
keys take effect. ESC returns focus to the terminal.

### Spawn

| Shortcut | Action |
|---|---|
| `n s` | Open Spawn dialog (when on Sessions page) |
| `Esc` | Close any open dialog |
| `Cmd/Ctrl + Enter` | Submit dialog (when in any field of the Spawn form) |

### Help

The hint bar at the top-right of every page shows the most-relevant
shortcuts for that page. Hover the `?` to see the full keymap.

## Touch / mobile

The Sessions page works on tablets but not phones — the terminal
needs at least 600px wide to be usable. On narrow viewports the
sidebar collapses to icon-only mode, the Inspector overlays
instead of sitting alongside, and the tab strip becomes a
horizontal scroll.

For phone-only usage, use a [channel](#channels-overview) instead:
get the idle notification, reply from your phone, and let opendray
forward your text into the right session via Telegram / Slack /
etc.
