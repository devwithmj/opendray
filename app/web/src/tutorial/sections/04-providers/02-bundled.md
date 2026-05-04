# Bundled providers

opendray ships with manifests for the most common AI coding CLIs.
Each section below covers what to expect when launching that
provider for the first time.

## Claude Code

| Field | Value |
|---|---|
| Provider id | `claude` |
| Default executable | `claude` (resolved via `$PATH`) |
| Default args | none |
| Multi-account support | yes — see [Claude accounts](#providers-claude-accounts) |
| JSONL transcript | yes — opendray reads it for channel notifications |
| Privileged intents | n/a |

Notes:

- Claude Code stores per-cwd transcripts at
  `~/.claude/projects/<encoded-cwd>/<session-id>.jsonl`. opendray
  reads this file directly to populate notification snippets — no
  screen-scraping required.
- The `--continue` flag resumes the most-recent conversation in
  the cwd. Drop it in the spawn dialog's *Args* field when
  picking up where you left off.
- Permission modes (`bypass permissions`, etc.) are a Claude TUI
  feature — opendray's chrome filter strips the hint banners
  from notification snippets but doesn't change the underlying
  behaviour.

## Codex

| Field | Value |
|---|---|
| Provider id | `codex` |
| Default executable | `codex` |
| Default args | none |
| Multi-account support | one credential per env (no per-session binding) |
| JSONL transcript | no — opendray uses screen snapshot for notifications |

Notes:

- Codex uses its own JSON-RPC protocol for tool use; opendray
  treats it as an opaque CLI and just relays bytes through the
  PTY.
- Free-tier rate limits apply; if Codex returns "rate limit
  exceeded" it'll surface in the terminal, opendray doesn't
  intercept it.

## Gemini CLI

| Field | Value |
|---|---|
| Provider id | `gemini` |
| Default executable | `gemini` |
| Default args | none |
| Multi-account support | env-based |
| JSONL transcript | no |

Notes:

- Gemini's free quota resets daily; check your quota dashboard
  if a session starts erroring with 429.
- The CLI's interactive prompt is more shell-like than Claude's
  TUI; chrome filtering is a no-op (nothing to strip).

## Plain shell

If you want a regular shell session (no AI), you can register a
custom provider pointing at `bash` / `zsh` / `fish` (see [Custom
provider manifest](#providers-custom)). opendray treats it
identically — same PTY, same idle detection, same ring buffer.

Useful for when you need a quick interactive session on the
opendray host without SSH'ing in.
