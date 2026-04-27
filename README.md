# OpenDray v2

> Multiplexer + integration gateway for AI agent CLIs.
> Mobile / web remote control of Claude Code, Codex, Gemini CLI, and more.
> Lets your other applications consume one shared subscription instead of paying per-token API.

**Status:** Greenfield rewrite. Pre-implementation. See [`docs/design.md`](docs/design.md) for the full plan.

## Why v2

v1 lives at `../opendray`. It works, but ~28k lines of plugin infrastructure (bridge / install / marketplace / host sidecar) were built for a third-party plugin economy that never materialised — none of the 16 bundled "plugins" actually use that machinery. v2 starts from the actual product needs and stays small.

## Core capabilities (target)

1. **Mobile + web control** of multiple AI agent CLI sessions.
2. **Subscription cost arbitrage** — one Claude Pro subscription serves the whole personal app ecosystem.
3. **Multi-CLI orchestration** — claude / codex / gemini in parallel.
4. **Integration system** — external apps register via reverse proxy + scoped API keys + event WS, no code in this repo.
5. **Channel hub** — Telegram / Slack / iMessage on a single contract, two-way.

## Documentation

- [`docs/design.md`](docs/design.md) — north-star design (mission, architecture, subsystems, API, data, roadmap)
- `docs/adr/` — architecture decision records (per-decision, dated)
- `docs/api.md` — REST + WS reference (generated from code, comes online once gateway scaffolding lands)
- `docs/integration-guide.md` — how to write an integration that consumes OpenDray
- `docs/operator-guide.md` — deploy + config

## Status / roadmap snapshot

Pre-M0. Bootstrap scaffolding only. Implementation begins after design doc review.

See [§18 of the design doc](docs/design.md#18-roadmap--milestones) for week-by-week milestones.

## Relationship to v1

v1 (`../opendray`) keeps running in production. v2 reaches feature parity before the user-facing switchover. After v2 v1.0 release, v1 is archived for one quarter then retired.

## License

Apache 2.0 (carried over from v1).
