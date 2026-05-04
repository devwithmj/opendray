# Integrations — overview

The Integrations page turns opendray into a managed reverse-proxy
+ signed-token gateway for third-party tools. This is the page
that earns "third-party API gateway" in the project description.

## What you get

| Capability | Page |
|---|---|
| Per-integration API keys (rotatable) | The Integrations list |
| Reverse-proxy other HTTP services through opendray with auth | Reverse proxy mounts |
| Audit log + per-call attribution | Call log |
| Subscribe to opendray's event bus from external tools | Events WebSocket |

Read the deep-dive sections below for each.

## Why route through opendray?

Three concrete use cases:

### 1. Single auth surface

You're running 4 internal tools (Grafana, a custom dashboard,
your AI usage tracker, a webhook receiver). Each has its own
auth flavor. opendray can proxy all of them, presenting one
admin token + one set of integration keys to the outside world.

### 2. Auditable AI usage

A tool running on your laptop wants to call Anthropic's API.
Routing it through opendray means every call is logged with
caller (integration key) + endpoint + duration + status — a
billing-quality audit trail.

### 3. Webhook fan-out

External services (GitHub, Stripe, a CI pipeline) send webhooks.
opendray accepts them on a public endpoint, signs them with the
integration's key, and republishes on the event bus where any
session, channel, or admin script can react.

## Auth model in one paragraph

Two flavors of token, both Bearer:

- **Admin token** — full access. Used by the web UI and admin
  CLIs. Stored in `config.toml` or env.
- **Integration key** — scoped, per-integration, rotatable.
  Stored hashed in the DB; the plaintext is shown once on
  creation. Used by external callers.

Endpoints under `/api/v1/` accept either; middleware attributes
the call to the integration when an integration key is used so
the call log shows who-did-what.

The dual-auth middleware order matters: `auth` runs first, then
`integration call logger` wraps the response so call attribution
sees the principal.

![Integrations page](/tutorial/integrations-layout.png)

## Read on

| Topic | Section |
|---|---|
| Token types and how they interact | Auth model |
| Mounting third-party APIs through opendray | Reverse proxy |
| The audit / attribution table | Call log |
| Subscribing to events from outside opendray | Events WebSocket |
