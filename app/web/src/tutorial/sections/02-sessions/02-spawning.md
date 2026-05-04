# Spawning a session

Click **New session** in the top-right of the Sessions page. The
spawn dialog has every field you need to launch a CLI under
opendray's lifecycle management.

![Spawn dialog](/tutorial/spawn-dialog.png)

## Required fields

### Provider

Pick the CLI to launch. The dropdown lists every provider
configured under [Providers](#providers-overview). Bundled
defaults: Claude, Codex, Gemini. The provider's icon and display
name show up here so you don't accidentally launch the wrong one
on a multi-CLI host.

### Working directory

The session's `pwd`. Two ways to fill it:

- **Type the absolute path** directly into the input.
- **Click the 📁 button** to open a file browser rooted at your
  home directory. Navigate, select a folder, click *Use this
  directory*.

opendray rejects paths that don't exist or aren't readable —
`stat()` is called pre-flight so you don't get an opaque
"fork/exec: no such file or directory" later.

## Optional fields

### Name

Friendly tab label. Default = the directory's basename (e.g.
`/Users/me/projects/foo` → `foo`). Override when you want a
custom label (e.g. `foo (refactor)`).

### Args

Extra CLI flags appended to the provider's defaults. Examples:

- Claude: `--continue` to resume the latest conversation in this
  cwd
- Codex: `--model gpt-5` to override the default
- Plain shell: leave blank

The provider's bundled args are not editable from here — they
live in the provider manifest (Providers page).

### Claude account

**Only shown when provider = `claude`.** Drops down all accounts
registered in **Providers → Claude accounts**. Picking one binds
the session to that credential — Claude Code reads
`~/.claude-accounts/<name>/` instead of the default
`~/.claude/`, so you can run `personal` and `work` accounts in
parallel without re-login dance.

The account binding is stored on the session row, so a Restart
of an ended session reuses the same account.

### Parent session

Drop-down of recently-stopped sessions. Selecting one fills the
spawn dialog with that session's provider + cwd + args + claude
account, ready to relaunch under a new id.

Useful patterns:

- A Claude session crashed mid-task → fork from it, same context
  comes back.
- Compare two model variants on the same task → spawn from the
  same parent twice with different `--model` args.

opendray persists `parent_session_id` so the family tree is
queryable later.

## What happens when you hit Spawn

1. opendray validates the cwd, generates a session id, inserts
   the DB row in state `STARTING`.
2. Forks a PTY, runs the provider's executable with the merged
   args + env.
3. Hooks up the stdout pump (writes to ring buffer + fans out to
   any subscribed WebSocket clients).
4. Marks state `RUNNING` once the first byte arrives, or after
   500ms if the process is silent on launch.
5. Switches the active tab to the new session.

If the launch fails (binary not found, cwd unreadable, exit-on-
start), the dialog stays open with a red banner showing the
specific error from `cmd.Start`.
