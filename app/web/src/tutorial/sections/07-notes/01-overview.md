# Notes — overview

The Notes page is a markdown editor + viewer over an Obsidian-
style vault. It serves two roles:

1. **Per-session linked notes.** Every session in the Sessions
   page has a note at `<vault>/sessions/<session-id>.md`
   embedded in the Inspector. Use it as the running scratchpad
   for that session's work.
2. **Standalone vault.** Project docs, decisions, references —
   anything you'd put in Obsidian. opendray works against the
   same vault Obsidian does, so you can edit from either side.

![Notes page](/tutorial/notes-layout.png)

## Vault root

Configured via `notes.root` in `config.toml`:

```toml
[notes]
root = "~/.opendray/vault"
```

opendray treats every `.md` file under that root as a note. The
default vault directory contains:

- `sessions/` — auto-managed per-session notes (don't rename
  files here; the linkage is by file path)
- `<your folders>/` — standalone notes, anything you create

## Read on

| Topic | Section |
|---|---|
| `[[Wiki Links]]` and how the backlinks pane works | Wiki links + backlinks |
| Auto-commit + push to a remote git host | Vault git sync |
| Source vs Preview mode + auto-save behaviour | Editor |
