# Reverse proxy

Mount any HTTP backend behind opendray's auth + audit pipeline.
The reverse-proxy section of the Integrations page is what makes
opendray a true "API gateway" rather than just a session manager.

## When to use

Two main cases:

1. **Aggregating internal tools.** You want one Bearer-token
   surface for Grafana, your custom dashboard, the webhooks
   collector, etc. Mount each behind a path prefix on opendray.
2. **Auditing AI provider calls.** Mount Anthropic's API behind
   `/api/v1/proxy/anthropic`, hand out scoped integration keys
   to internal tools, and every call shows up in the call log
   with caller attribution.

## Adding a proxy mount

Integrations → **Reverse proxy** sub-tab → **Add mount**.

| Field | Purpose |
|---|---|
| **Path prefix** | Where on opendray the mount lives (e.g. `/api/v1/proxy/anthropic`) |
| **Upstream base URL** | Where to forward to (e.g. `https://api.anthropic.com`) |
| **Integration ids** | Whitelist of integration keys allowed to hit this mount; empty = admin-only |
| **Strip prefix** | When `true`, strip the path prefix before forwarding |
| **Header passthrough** | Comma-separated list of incoming headers to forward |
| **Header injection** | Map of headers to set on the outgoing request (e.g. inject `Authorization: ...` from a secret) |

![Reverse proxy mount form](/tutorial/integrations-proxy-mount.png)

## Header injection example

Mount Anthropic's API with the upstream API key injected from an
opendray-managed secret:

```
Path prefix:        /api/v1/proxy/anthropic
Upstream base URL:  https://api.anthropic.com
Strip prefix:       true
Header injection:   x-api-key=$ANTHROPIC_API_KEY
```

Now any integration key hitting
`https://opendray/api/v1/proxy/anthropic/v1/messages` gets the
request forwarded to `https://api.anthropic.com/v1/messages`
with the right `x-api-key` set. The internal tool never sees the
upstream API key.

`$ANTHROPIC_API_KEY` interpolates from opendray's environment;
operators rotate it once and every mount picks up the new value.

## What the proxy does

1. **Auth** — the request goes through the standard dual-auth
   middleware. Admin token or whitelisted integration key.
2. **Strip prefix** — if enabled, drops the mount prefix from
   the forward URL.
3. **Header rewrite** — passthrough + injection.
4. **Forward** — `httputil.ReverseProxy` does the actual
   request, streaming the body in both directions (so SSE
   from Anthropic flows through unbuffered).
5. **Call log** — middleware records the call with the calling
   integration id + status code + duration.

Streaming: opendray uses `Flush()` after every response chunk,
so Server-Sent Events from upstream API providers (Anthropic's
`/v1/messages?stream=true`, OpenAI's `/v1/chat/completions`) work
without buffering latency.

## Limitations

- **No path rewrite beyond strip-prefix.** If you need
  `/foo/bar` → `/baz/bar`, run a real reverse proxy (Caddy,
  nginx) in front. opendray's proxy is intentionally simple.
- **No retry on upstream 5xx.** A failure surfaces immediately
  to the caller. Run idempotent jobs through external retries.
- **No body inspection.** opendray doesn't parse JSON bodies;
  request size limit is 10 MiB by default.
- **WebSocket proxying** is not supported (the proxy is HTTP-
  request-scoped). Use Events WS for opendray-native event
  subscription.
