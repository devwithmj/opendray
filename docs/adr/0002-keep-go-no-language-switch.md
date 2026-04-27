# ADR 0002 — Keep Go for the gateway; no language switch

**Status:** Accepted
**Date:** 2026-04-27
**Decider:** Linivek

## Context

During v2 planning, the question of switching the backend language arose, motivated by interest in hot-reload capabilities (Erlang/Elixir BEAM) or unified frontend/backend language (Node/TypeScript).

## Decision

Keep Go 1.25+ as the gateway language.

## Consequences

- Reuse Go ecosystem strengths: single static binary, goroutines for PTY/WS fanout, mature `chi`, `pgx`, `gorilla/websocket`, `creack/pty`.
- Hot reload of opendray's own code requires restart (≤5 s). Acceptable for a single-maintainer self-hosted tool.
- Hot reload of *integration logic* is delivered by the integration system: integrations run as separate processes and hot-reload independently in whatever language they choose.

## Rationale by alternative

| Alternative | Why rejected |
|---|---|
| Elixir / Erlang | BEAM hot-reload is real and excellent, but designed for high-availability telecom switches. Single self-hosted tool with restart-tolerable downtime gains marginal value. Cost: 6–12 months rewrite + ecosystem learning. |
| Node.js / TypeScript | Frontend-backend uniformity is appealing but Node has weaker concurrency for PTY fanout, GC pauses in long-lived sessions, harder single-binary deploy. |
| Rust | Performance + safety wins are real, but hot-reload story is worse than Go's, and the rewrite cost dwarfs Go's. |
| Bun / Deno | Same drawbacks as Node, smaller ecosystem maturity for ops needs. |
| Go + embedded scripting (Lua / Starlark / QuickJS) | Not rejected, deferred. If a real need for hot-reloadable in-gateway behaviour surfaces post-v1.0, add a scripting layer then. Cost: ~200 lines + a sandbox decision. |

## Open future trigger

Revisit if **any of** the following becomes true:
- Multiple users contribute integrations and ask for in-gateway scripting.
- A real-time use case (e.g. sub-second multi-session orchestration) hits Go's GC.
- A specific feature requires runtime code modification of gateway logic.
