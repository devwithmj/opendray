# ADR 0004 — v1 builtin disposition: catalog ships 4, not 16

**Status:** Accepted
**Date:** 2026-04-28
**Decider:** Linivek

## Context

v1 ships 16 declarative "builtin plugins" under `plugins/builtin/`.
Design ADR 0001 already established that v1's plugin substrate (bridge,
host, market, install — ~28k LOC) was over-built for use cases that
never materialised. Open question for v2: which of those 16 builtins
become v2 Catalog entries, which move to other v2 subsystems, and
which are dropped entirely.

The blunt-force option — port all 16 to `internal/catalog/builtin/` —
preserves v1 surface but bloats v2 with backend code that v1 never
actually needed as backend code. Most v1 builtins are GUI tools (file
browser, log viewer, in-app browser); they were implemented as
plugins because v1 had no other extension axis. v2 has three real
extension axes (Catalog for CLI providers, Channel for messaging
services, Integration for external apps), plus the clients themselves
(Flutter mobile, web). Each builtin can be re-homed to its natural
axis instead of being forced through "plugin".

## Decision

| # | v1 builtin       | v2 destination                                           | Why |
|---|------------------|----------------------------------------------------------|-----|
| 1 | claude           | **M2 Catalog**                                           | Core AI agent CLI |
| 2 | codex            | **M2 Catalog**                                           | Core AI agent CLI |
| 3 | gemini           | **M2 Catalog**                                           | Core AI agent CLI |
| 4 | shell            | **M2 Catalog** (already seeded in M1α)                   | Generic PTY launcher |
| 5 | mcp              | **M2 Catalog config dimension**, not standalone provider | v1 description: "configurations are injected into Claude / Codex sessions as temporary files on spawn." That is a *spawn-time hook* on existing providers, not a separate provider. v2 puts `mcp_servers` on each provider's config; session.Manager writes the temp file and injects the env var on Create. |
| 6 | telegram         | **M4 Channel Hub**                                       | Bidirectional messaging service. Belongs behind the unified Channel interface (design §8.3), not a "plugin". |
| 7 | terminal         | **dropped (M5 client renders it)**                        | Backend-side `shell` already covers PTY launch. Terminal rendering is a client-side concern. |
| 8 | file-browser     | **M5 Flutter app built-in**                               | UI tool with no backend semantics. The mobile/web client already needs filesystem access for in-app browsing; that does not require a backend Catalog entry. |
| 9 | log-viewer       | **M5 Flutter app built-in**                               | Same as file-browser — UI on top of files. |
| 10 | web-browser     | **M5 Flutter app built-in** (already in design §3)        | "In-app browser preview" is listed in the design's value props. Native to the client. |
| 11 | source-control  | **post-M4 channel-side feature**                          | Per design §17 already deferred. Will rebuild "git status / push" as channel-style commands routed through the Channel Hub. |
| 12 | task-runner     | **M5 client + session API**                               | "Find Makefile/npm targets and run them with live stream" = client lists targets, then POSTs `/api/v1/sessions` against the `shell` provider. No new subsystem needed. |
| 13 | pg-browser      | **M3 Integration candidate (or drop)**                    | If kept, deploy a third-party PG GUI as a separate process and register it via M3 Integration. Default is to drop and use psql / TablePlus. Decision deferred to M3. |
| 14 | opencode        | **dropped**                                               | Operator confirmed no longer in use. Removable without migration. |
| 15 | qwen-code       | **dropped**                                               | Same as opencode. |
| 16 | obsidian-reader | **dropped**                                               | Operator marked as not useful. |
| 17 | simulator-preview | **dropped**                                             | iOS/Android emulator streaming sits outside the AI-multiplexer mission (design §1) and is rarely used by the operator. Re-add as M3 integration only if a real need surfaces. |

## Consequences

- **v2 Catalog backend ships 4 providers (claude, codex, gemini, shell), down from v1's 16.**
  This is the concrete proof that v1's "plugin model failed" — 12 of the
  16 builtins were not Catalog material to begin with.
- **MCP becomes a provider config field, not a subsystem.** Provider
  manifests gain `mcp_servers: []` in their config schema; session.Manager
  resolves it at spawn time, writes a per-session temp file, and exposes
  its path via the env var the target CLI expects. Cleanup runs in the
  exit detector after `session.ended` fires.
- **Client-side responsibilities are larger but cleaner.** The Flutter
  mobile app (M5) absorbs file-browser, log-viewer, web-browser, terminal
  rendering, and task-runner UI. Each of those is "list / view / route
  to existing API" — not new backend.
- **Three drops are reversible** if a need re-emerges; manifests can be
  copied from v1 archive.
- **One deferred axis (pg-browser → M3 integration)** is captured here so
  it does not get forgotten when M3 lands.

## Trigger to revisit

Re-add a backend Catalog entry only if **all three** are true:

1. The thing is launched as a long-lived process the user wants OpenDray
   to manage (PTY lifecycle, not request/response).
2. Multiple clients (mobile + web + integration) all need to start it
   the same way.
3. The launch needs server-side state (DB row, audit, event bus).

If any one of those is false, the right v2 home is a Channel, an
Integration, or the client.
