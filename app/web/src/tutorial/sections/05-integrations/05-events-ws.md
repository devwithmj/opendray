# Events WebSocket

`/api/v1/integrations/_events` exposes opendray's internal event
bus as a WebSocket stream. Any process holding a valid token
(admin or integration with the right scope) can subscribe and
react to events as they happen.

## When to use

- **Local dashboards** that want a live tail without polling.
- **Custom alerting** scripts that act on `session.idle` or
  `channel.message_forwarded`.
- **Cross-host orchestration** — one opendray's events drive
  another opendray instance via integration keys.

The Activity page uses this same endpoint internally, so any
event you can see there is reachable from external code.

## Connecting

```python
import websocket
import json

ws = websocket.WebSocketApp(
    "ws://opendray-host/api/v1/integrations/_events",
    header={"Authorization": "Bearer YOUR_INTEGRATION_KEY"},
    on_message=lambda _, msg: print(json.loads(msg)),
)
ws.run_forever()
```

Or with `wscat`:

```bash
wscat -c "ws://opendray-host/api/v1/integrations/_events" \
      -H "Authorization: Bearer YOUR_INTEGRATION_KEY"
```

For browsers (which can't set Authorization on a WS upgrade), the
endpoint also accepts `?token=` in the query string. Use this
sparingly — query tokens leak into proxy access logs.

## Frame shape

Every frame is one JSON object:

```json
{
  "topic": "session.idle",
  "ts": "2026-05-04T10:32:14.123Z",
  "data": {
    "session_id": "ses_abc123",
    "idle_for_ms": 30000,
    "recent_output": "● Got it — let's design the API.\n\n● Write(...)..."
  }
}
```

| Field | Notes |
|---|---|
| `topic` | dotted name; subscribe filter matches by prefix |
| `ts` | server-side timestamp at publish |
| `data` | topic-specific payload |

## Filtering

Pass `?topics=session.,channel.message_` (comma-separated
prefixes) on the WS upgrade URL. Server-side filter — only
matching events stream to that connection.

If `topics` is omitted, you get **everything** — useful for the
Activity page where the operator filters client-side, but bad
for production scripts (you waste bandwidth).

## Topics catalogue

See [Activity → Topics catalogue](#activity-topics-catalogue) for
the full list. The most-used:

| Topic | Payload highlights |
|---|---|
| `session.started` | `session_id`, `provider_id`, `cwd` |
| `session.idle` | `session_id`, `idle_for_ms`, `recent_output` |
| `session.ended` | `session_id`, `exit_code`, `state` |
| `channel.message_received` | `channel_id`, `text`, `author` |
| `channel.message_forwarded` | `channel_id`, `session_id`, `text` |
| `channel.command_received` | `channel_id`, `command`, `args` |
| `integration.call_logged` | call log row, post-write |

## Backpressure

Slow subscribers get **dropped** rather than blocking the bus —
opendray prioritises the producer side of every event channel.
If your script stalls on a long DB write while events fan in, you
will silently lose events. Two options:

1. Spawn a goroutine / async task per frame and return
   immediately from `on_message`.
2. Subscribe with a tight `topics` filter so the per-second
   rate is low.

Lost events do **not** trigger a reconnect — there's no "you
missed N events" signal. For audit-grade processing, use the
call log (which is durable Postgres) and treat the events stream
as a UI/dashboard tool.

## Connection lifecycle

- WS-level pings every ~54s; clients should respect WebSocket
  Pong (most libraries do automatically).
- Idle close: 5 minutes with no inbound or outbound activity.
- Server restart: the connection drops cleanly with code 1001
  ("going away"). Reconnect after a backoff.
