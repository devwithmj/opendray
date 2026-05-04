# Vault git sync

opendray can auto-commit + push the entire notes vault to a
remote git host on a timer. This is how the vault stays
consistent across multiple opendray hosts (laptop ↔ home server)
and how you avoid losing notes when a single host dies.

## Prerequisites

- `git` available on the opendray host (`which git` returns a
  path).
- The vault root has a `.git` directory **or** opendray will
  `git init` on first run.
- A remote configured:
  - SSH: standard `git@github.com:...` URL with the host's SSH
    key trusted.
  - HTTPS: a [Git host token](#plugins-git-hosts) configured in
    Plugins → Git hosts so opendray can authenticate
    non-interactively.

## Configuring auto-sync

`config.toml`:

```toml
[notes]
root = "~/.opendray/vault"
git_root = "~/.opendray/vault"

[notes.sync]
enabled = true
interval = "5m"           # commit every 5 minutes when there are changes
push_on_commit = true     # also push to origin after a commit
remote = "origin"
branch = "main"
author_name  = "opendray"
author_email = "opendray@example.com"
```

After save + opendray restart, the syncer logs:

```
INFO vault auto-sync started component=vaultgit.sync interval=5m
```

## What the syncer does

Every `interval`:

1. `git status --porcelain` — any pending changes?
2. If yes:
   - `git add .`
   - `git commit -m "vault sync: <N> file(s) changed"`
3. If `push_on_commit` and we just committed:
   - `git push <remote> <branch>`
4. If `behind` upstream (commits exist on remote that we don't):
   - `git pull --rebase <remote> <branch>` (best-effort; on
     conflict, falls back to a full clone-replace as a recovery
     mechanism — the conflict file is kept as `<name>.conflict.md`)

The cycle's outcome publishes `vaultgit.sync_completed` on the
event bus so external monitoring can react.

## Status indicators

The Notes page shows a sync status pill at the top:

| Pill | Meaning |
|---|---|
| 🟢 *In sync* | local matches remote, no pending changes |
| 🟡 *Pending commit* | changes since last commit; will commit at next tick |
| 🔵 *Pushing…* | mid-sync (push in progress) |
| 🔴 *Conflict* | rebase failed; a `.conflict.md` file was created |

Click the pill → expand to see last sync time, files changed,
and a manual *Sync now* button.

## Branching

opendray uses a single branch (default `main`). For per-host
isolation, point different hosts at different branches:

- Host A: `[notes.sync] branch = "main"`
- Host B: `[notes.sync] branch = "host-b"`

Then merge between branches manually when you want cross-pollination.

## Manual escape hatches

The vault is just a git repo — open a shell on the host and use
git directly any time. opendray's syncer is best-effort and
doesn't lock the repo; just pause auto-sync (set `enabled =
false` and reload) before doing anything destructive (like a
force-push).

## Remote auth

For HTTPS remotes, opendray injects credentials from the **Git
hosts** plugin (Plugins page). The `Authorization` header is
attached to push/pull requests. Tokens never reach the worktree
or the commit log.

For SSH, use whatever SSH key the opendray host already trusts
— `git push` shells out to system `git` which uses the standard
agent.

## Things to know

- **Don't put `node_modules/` or build outputs in the vault.**
  The syncer doesn't auto-`.gitignore` for you. Add a
  `.gitignore` at the vault root.
- **Big binary files** (PDFs, images) inflate the repo fast.
  Use git-lfs (opendray won't fight it) or move binaries
  outside the vault.
- **First run after `git init`** has no remote — set one up
  manually:
  ```bash
  cd ~/.opendray/vault
  git remote add origin git@github.com:me/vault.git
  git push -u origin main
  ```
  After that, opendray's sync takes over.
