# Topics catalogue

Every event opendray publishes with its full payload shape.
Useful as a reference when:

- Writing an external script that subscribes via [Events
  WebSocket](#integrations-events-ws).
- Building a custom plugin that needs to react to opendray
  events.
- Debugging "what does this topic actually carry?".

## session.*

### `session.started`

```json
{
  "session_id": "ses_abc123",
  "provider_id": "claude",
  "cwd": "/Users/me/projects/foo",
  "started_at": "2026-05-04T10:00:00Z"
}
```

### `session.idle`

```json
{
  "session_id": "ses_abc123",
  "idle_for_ms": 30000,
  "recent_output": "● Got it — let's design the API…"
}
```

`recent_output` is **only present** when the session has produced
output AND the snippet pipeline succeeded. Empty for fresh /
silent sessions.

### `session.ended`

```json
{
  "session_id": "ses_abc123",
  "exit_code": 0,
  "ended_at": "2026-05-04T10:30:00Z",
  "state": "ended"
}
```

`state` is `ended` for natural exits, `stopped` for SIGTERM-
driven shutdowns.

### `session.stopped`

Same payload as `session.ended` but published when the operator
explicitly stopped the session via the × button.

## channel.*

### `channel.message_received`

```json
{
  "channel_id": "ch_xyz",
  "channel_message_id": 12345,
  "conversation_id": "42",
  "author": "@alice",
  "text": "any plain text the user typed"
}
```

Fires when a non-command message arrives that **wasn't** routed
to a session. (Most non-command text these days routes to a
session and emits `channel.message_forwarded` instead.)

### `channel.message_forwarded`

```json
{
  "channel_id": "ch_xyz",
  "channel_message_id": 12345,
  "session_id": "ses_abc",
  "text": "the text that was written to the session's stdin"
}
```

Fires when opendray sucessfully forwarded inbound channel text
into a session via the routing pipeline.

### `channel.message_sent`

```json
{
  "channel_id": "ch_xyz",
  "topic": "session.idle"
}
```

Fires after opendray successfully posted an outbound
notification on the channel. The `topic` field carries the
*originating* event topic (so you can correlate idle → sent).

### `channel.command_received`

```json
{
  "channel_id": "ch_xyz",
  "channel_message_id": 12346,
  "command": "cancel",
  "args": ["ses_abc"],
  "source": "builtin"
}
```

`source` is `builtin` (opendray-registered) or `custom` (app-
registered).

### `channel.command_unknown`

Same shape as `command_received` but minus `source`. Fires when
the user typed an unrecognised slash command — opendray replies
with "try /help".

## integration.*

### `integration.call_logged`

```json
{
  "principal_kind": "integration",
  "principal_id": "int_abc",
  "method": "POST",
  "path": "/api/v1/proxy/anthropic/v1/messages",
  "status": 200,
  "duration_ms": 1234,
  "request_id": "yinglincuisMini/abc123-000042"
}
```

Fires once per logged call, **after** the response is fully
written.

### `integration.health_changed`

```json
{
  "integration_id": "int_abc",
  "previous": "healthy",
  "current": "degraded",
  "reason": "5 consecutive 5xx responses"
}
```

Fires when the periodic health check flips an integration's
status. (Status is updated even when it doesn't flip; the event
fires only on transitions.)

## audit.*

### `audit.event`

```json
{
  "actor_kind": "admin",
  "actor_id": null,
  "action": "channel.update",
  "subject_kind": "channel",
  "subject_id": "ch_xyz",
  "metadata": { ... }
}
```

Mirrors every row inserted into the `audit_log` table. Use this
for SIEM ingestion when you don't want to poll the table.

## vaultgit.*

### `vaultgit.sync_completed`

```json
{
  "files_changed": 3,
  "ahead": 5,
  "behind": 0,
  "duration_ms": 412
}
```

Fires after each automatic vault git sync cycle.

## Custom topics

Any plugin / extension can publish under its own prefix. By
convention, use `<plugin-name>.<event>` so consumers can filter
without collisions. opendray itself reserves `session.*`,
`channel.*`, `integration.*`, `audit.*`, `vaultgit.*`.
