# Call log

Every authenticated `/api/v1/*` call (admin or integration) lands
in the `integration_call_log` Postgres table. The Integrations
page surfaces a search + filter UI on top of it.

## What's logged

| Column | Example |
|---|---|
| `id` | bigint serial |
| `ts` | `2026-05-04 10:32:14+00` |
| `principal_kind` | `admin` or `integration` |
| `principal_id` | admin uid (always `1`) or `int_abc...` |
| `method` | `GET` / `POST` / etc. |
| `path` | `/api/v1/sessions/ses_xyz/stream` |
| `status` | HTTP status code |
| `duration_ms` | round-trip time |
| `request_id` | matches the structured log line |

opendray excludes admin calls from the log by default — the
table is for *external* tools, not the operator clicking around.
Toggle the **Include admin calls** filter if you want to see
your own admin actions too.

## What it's good for

### Billing / chargeback

```sql
select principal_id, count(*), avg(duration_ms)
from integration_call_log
where ts >= now() - interval '30 days'
  and path like '/api/v1/proxy/anthropic/%'
group by 1
order by 2 desc;
```

Tells you which integration burned the most Anthropic API calls
last month — useful when you're scaring chargeback to internal
teams.

### Anomaly detection

The Activity page can subscribe to `integration.call_logged`
events live and surface them in real time. Easy ad-hoc alerts
("ping me if any call to /v1/messages takes >30s") without
running a full APM stack.

### Forensics

When an integration's behaviour changes unexpectedly, the call
log gives you:

- timing — when did the volume spike start?
- correlation — `request_id` lets you find the structured log
  line for any call
- attribution — which key was used (helps pin the misbehaving
  caller)

## Filters

The table supports filters on:

- Time range (default last 24h; presets for 1h / 24h / 7d / 30d)
- Principal (any specific integration id, or `admin`)
- Path prefix (auto-completes on existing path patterns)
- Status code range
- Duration ≥ N ms

Filters apply to a server-side SQL query, not in-memory — so
filtering 30 days of high-volume traffic is fast.

## Retention

Default retention is **30 days**. Configurable in `config.toml`:

```toml
[integration]
call_log_retention_days = 30
```

A daily background job runs a `DELETE FROM integration_call_log
WHERE ts < now() - retention`. To keep forever, set
`call_log_retention_days = 0`.

## Exporting

For external SIEM ingestion, the API exposes:

```
GET /api/v1/integrations/_call-log/export?since=<ts>
```

Streams JSONL to the response body — one row per call. Use the
admin token; the call log is read-only via this endpoint
(insertion is server-internal).

## Privacy note

opendray does **not** log request or response bodies. The path is
recorded but query parameters that look like tokens (`?token=...`,
`?api_key=...`, `?password=...`) are redacted to `?token=REDACTED`
before storage. If you embed sensitive content in path segments
(don't), adjust `internal/integration/calllog.go`'s `redactPath`
to match.
