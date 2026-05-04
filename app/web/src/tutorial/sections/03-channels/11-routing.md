# Multi-session routing

When you're running 3+ sessions concurrently, replying to "the
right one" matters. opendray picks the target session for an
inbound non-command message using this priority:

| Priority | Source | Trigger |
|---|---|---|
| 1 | Reply-to-message | Long-press a notification → Reply (Telegram, Slack thread, Discord ref, Feishu reply) |
| 2 | `/select <sid>` pin | Explicit slash command |
| 3 | Last-notified session | Fallback — most-recent notification on this channel |

## Reply-to-message routing (best UX)

Every chat platform supports "reply to a specific message" —
long-press / right-click → Reply. opendray captures the
referenced message id from the inbound payload and looks it up
against an in-memory index of `(channel, outbound_msg_id) →
session_id`.

As long as the original notification message is still in chat
history (it usually is — the platforms don't auto-delete),
replying to it routes your text to that specific session.

The index holds the most recent ~256 outbound notifications per
channel, with LRU eviction beyond that. Old idle notifications
(weeks back) eventually drop out — at that point the reply
fallbacks to priority 2 or 3.

![Telegram long-press reply](/tutorial/routing-reply-to-message.png)

## `/select` for explicit pinning

When you're going to send several messages to the same session,
pin it once:

```
/select ses_abc123
→ Now routing replies to session ses_abc123. Use /select clear to unpin.
```

Subsequent non-command text routes to that session, overriding
the *last-notified* fallback.

To unpin:

```
/select clear
→ Pinned session cleared. Routing falls back to last-notified.
```

## `/sessions` to find the IDs

Don't have the session id memorised? Run:

```
/sessions
→ Recently-notified sessions (most recent first):
    /select ses_xyz789 ← /select
    /select ses_abc123 (last)
    /select ses_old456

   Tip: replying to a notification routes to *that* session directly.
```

The marker `← /select` shows which one is currently pinned;
`(last)` shows the most-recent notification target. Each line is
the literal slash-command, so you can tap to copy + paste into the
input bar.

## Where the bytes go

When the routing target is determined, opendray:

1. Forwards your text + a trailing `\r` (carriage return —
   Enter) into the session's stdin via
   `Manager.Input(sid, payload)`.
2. Clears the *Once*-mode suppression entry for that session, so
   the next idle event re-notifies.
3. Publishes a `channel.message_forwarded` event for audit.

The `\r` is critical: TUIs running in raw mode (Claude Code,
Codex, Gemini) treat `\r` as Enter (submit) and `\n` as
shift-Enter (insert newline). Sending `\n` puts the text in the
input box but doesn't submit it.

## Slash commands always trump routing

Any text that parses as a slash command (`/help`, `/cancel`,
`/notify`, `/select`, `/sessions`, `/status`, plus any custom
commands the app registered) skips routing entirely and runs
through the command dispatcher. The reply lands in the same chat
but never touches a session's stdin.

This is why `/cancel ses_abc` doesn't accidentally type "/cancel
ses_abc" into Claude's input box.

## Failure modes

- **"Could not deliver to ses_xxx: session not found"** — the
  pinned session has ended. Use `/sessions` to find an active
  one and `/select` again, or just send a fresh non-command
  message which falls back to last-notified.
- **Routing to the wrong session** — usually because the *last*
  notification was from a different session than the one you
  meant. Use reply-to-message or `/select`.
- **Reply-to-message returns "Could not deliver: session ended"**
  — the message routed to the right session but it's no longer
  running. Restart it via the web UI or `/spawn-like ses_xxx`.
