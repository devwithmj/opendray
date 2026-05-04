# Claude accounts

Claude Code looks for credentials at `~/.claude/`. To run multiple
Claude accounts on the same host (e.g. `personal` and `work`)
without losing your mind, opendray supports per-session account
binding — each session points at `~/.claude-accounts/<name>/`
instead of the default location.

## When to use this

- You have a personal Claude account and a work account, both
  on the same Anthropic billing.
- You're testing two different API tier subscriptions.
- You want to run two parallel Claude sessions in the same cwd
  with different model defaults.

If you only ever use one Claude account, ignore this — the
default `~/.claude/` path works fine.

## Setting up an account

The Claude accounts panel is on the **Providers** page, below
the provider list.

![Claude accounts panel](/tutorial/providers-claude-accounts.png)

1. Click **Add account**.
2. Enter a short name (e.g. `personal`, `work-codex`).
3. Run the Claude CLI's login flow under the new directory:

```bash
CLAUDE_CONFIG_DIR="$HOME/.claude-accounts/personal" claude login
# walk through the OAuth flow
```

opendray watches `~/.claude-accounts/` for new directories and
shows them in the panel automatically.

4. (Optional) Set this account as the **default** by clicking
   the star icon. New Claude sessions without an explicit
   account binding inherit this default.

## Binding a session to an account

In the [Spawn dialog](#sessions-spawning), the **Claude
account** dropdown only appears when provider = `claude`. Pick
the account; opendray sets `CLAUDE_CONFIG_DIR` for the spawned
process so Claude reads from the right directory.

The binding is persisted on the session row (`claude_account_id`)
so a Restart of an ended session reuses the same account.

## Switching mid-session

Live account switching: Sessions page → terminal pane → in
top-right of the terminal there's an **Account switcher** drop-
down. Picking a different account:

1. Sends SIGTERM to the running process
2. Waits for clean exit
3. Re-spawns the same provider + args + cwd, but with the new
   `CLAUDE_CONFIG_DIR`
4. The session id stays the same — same tab, same Inspector
   linked note

The terminal contents reset (new process, new TUI). Treat it
as "the same session, different credential" rather than "a
fresh start".

## Limitations

- Only Claude has this binding. Codex / Gemini use env vars
  set per-process at spawn time; if you need multi-account
  Codex, set `OPENAI_API_KEY` differently in the spawn dialog's
  *Args* (or wrap in a custom provider manifest).
- Account names cannot contain `/`, `..`, or non-printable
  characters — the directory lookup is sandboxed to
  `~/.claude-accounts/<name>` strictly.
- Deleting an account directory while a bound session is
  running breaks the next Claude API call. opendray doesn't
  prevent the directory deletion — be careful.
