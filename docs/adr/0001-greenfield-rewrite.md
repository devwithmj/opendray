# ADR 0001 — Greenfield rewrite, not v1 refactor

**Status:** Accepted
**Date:** 2026-04-27
**Decider:** Linivek

## Context

v1 (at `../opendray`) accumulated architectural debt that surgical extraction cannot resolve:

- Two conflated "plugin" concepts: (A) UI extension consumed by Flutter, (B) external-app integration. Only (A) ever existed in code; (B) is what the user actually needs.
- ~28k lines of plugin infrastructure (bridge / install / marketplace / host sidecar) used by 0 of 16 bundled "plugins". All 16 builtins are `form: declarative`.
- Vendor modules (`gateway/telegram`, `gateway/sourcecontrol`, `gateway/mcp`, `gateway/claude_accounts`) hardcoded into the gateway router; new channel/forge requires editing core.
- Marketplace, signing, consent, capability-gate built for a third-party plugin economy that does not exist for a single-maintainer self-hosted tool at v0.x.

A surgical refactor was attempted on branch `refactor/plugin-driven` (commit `9f58b71`, Phase 0). Mid-stream we realised the design (in-process Go capability registration) targeted (A), while the actual need is (B). Phase 0 is being abandoned, not extended.

## Decision

Start a greenfield rewrite at `../opendray-v2`. Migrate by lifting the proven parts (PTY core, Flutter app, manifest format) and rebuilding the rest against a clean small architecture.

## Consequences

- v1 stays in production until v2 reaches feature parity.
- v1 repository becomes read-only reference after v2 v1.0 ships.
- Estimated 12 weeks to v1.0 (per design doc §18).
- Lost work: ~3 hours on the abandoned Phase 0 contract layer. Acceptable.
- Forced clarity: the integration system is now first-class instead of fighting the existing plugin abstraction.

## Alternatives considered

1. **Continue v1 surgical refactor.** Rejected: (A)/(B) entanglement makes every PR cross-cut, every deletion risky.
2. **Switch language (Elixir / Node / Rust).** Rejected: see ADR 0002. Hot-reload demand is satisfied by external integration processes; gateway language is not the bottleneck.
3. **Leave v1 as-is.** Rejected: (B) integration system has real user demand and cannot be added cleanly to v1 without first dismantling plugin substrate.
