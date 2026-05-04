# Keyboard & theme

Two pure-cosmetic settings panels.

## Theme

Settings → **Theme** picker:

- **Dark** — default, matches the rest of the admin design.
- **Light** — for daylight-monitor use.
- **System** — follows the OS preference; auto-flips when your
  OS dark-mode toggles.

The theme persists in localStorage per browser. There's no
"theme per device" toggle — every browser tab on your account
shares one.

Custom theme tokens (when you want corporate branding) live
under `app/web/src/index.css` — change the CSS variables and
rebuild. opendray doesn't ship a theme editor.

## Keyboard shortcuts

Settings → **Keyboard shortcuts** lists every binding the admin
respects, with an inline editor.

![Keyboard shortcuts editor](/tutorial/settings-shortcuts.png)

| Default | Action |
|---|---|
| `g s` | Sessions |
| `g c` | Channels |
| `g p` | Providers |
| `g i` | Integrations |
| `g a` | Activity |
| `g n` | Notes |
| `g l` | Plugins |
| `g ,` | Settings |
| `g h` | Tutorial (this page) |
| `n s` | New session (when on Sessions page) |
| `Cmd / Ctrl + K` | Command palette |
| `?` | Keyboard shortcut help dialog |
| `Esc` | Close any open dialog / popup |

To rebind:

1. Click the binding row.
2. Press the new key combination.
3. The new combo replaces the old one. Conflicts are flagged
   ("This combo is already bound to <other action>").
4. Save → applies immediately (no restart).

Bindings persist in localStorage. **Reset to defaults** drops
all customisations.

## Command palette

Cmd/Ctrl + K opens a fuzzy-searchable command palette. Every
admin action that has a shortcut also lives there, plus things
that don't (e.g. *Create new note*, *Restart this session*).

For multi-step workflows, the palette beats memorising
shortcuts — it shows you what's possible inline.
