# ADR 0011 — Channel rich content + bridge protocol

**Status:** Accepted
**Date:** 2026-05-03
**Decider:** Linivek
**Relates to:** ADR 0005 (channel inbound routing), ADR 0006 (integration auth)
**Code:**
- `internal/channel/{channel.go,card.go,command.go,hub.go}`
- `internal/channel/telegram/`
- `internal/channel/bridge/`
- `docs/bridge-protocol.md`

## Context

The M4 channel hub had a single-method `Channel` interface that only
shipped plain text. In practice this meant:

1. Every outbound notification was a one-line string ("Session abc went
   idle"). No buttons, no formatting, no thread routing.
2. Inbound was equally bare — text in, text out — so users could not
   reply with `/cancel` or click a "Resume" button to drive a session.
3. Adding a new platform (Slack, Feishu, WeChat, Discord) required
   writing Go and recompiling. The Telegram impl in M4 was already 270
   LOC after deliberately stripping the v1 multi-select / question /
   slash-command machinery.

The reference implementation we studied — chenhg5/cc-connect — solves
all three by (a) exposing capability-scoped sub-interfaces, (b)
modeling a `Card` widget set with text fallback, and (c) shipping a
WebSocket Bridge protocol that lets external adapters register at
runtime.

## Decision

Adopt the same three-pronged approach, scoped to opendray's channels-
as-rows model.

### 1. Capability-aware Channel interface

`Channel` keeps only `Kind/ID/Start/Stop/Send`. Optional capability
interfaces — `CardSender`, `ButtonSender`, `MessageUpdater`,
`TypingIndicator`, `ImageSender`, `FileSender`, `ReplyCapable` — are
type-asserted by the Hub. A capability impl may also return
`channel.ErrNotSupported` to opt out at call time; the Hub treats this
identically to "interface not implemented" and falls back to
`Send(text)` with `Card.RenderText()`.

`Capabilities(Channel) []Capability` powers admin UI display and is
exposed on `GET /api/v1/channels`. Channels can override the default
type-assertion detector via `CapabilityProvider.DeclaredCapabilities()`
— used by Bridge to mirror the live adapter's capability set.

### 2. Card widget set

`Card` ⊃ `CardHeader` + `CardElement[]` where each element is one of
`CardMarkdown` / `CardDivider` / `CardActions` (button rows) /
`CardListItem` (text + single button) / `CardSelect` (dropdown) /
`CardNote`. Every element implements `RenderText() string` so the Hub
always has a plain-text fallback.

`ButtonOption.Value` is the opaque callback payload. Conventional
prefixes: `cmd:/<command> [args]` for slash-command routing, `nav:/...`
for UI links. The Hub's `ParseCommand` recognises `act:` (button
wrapper) + `cmd:` + leading `/` interchangeably.

The session-event card built by `buildSessionCard` adds Resume / End /
Mute buttons to `session.idle` and Open-log / Spawn-similar buttons to
`session.ended`.

### 3. Slash command registry

`CommandRegistry` carries built-in commands `/help`, `/notify on|off`,
`/status`. App code adds session-aware commands via
`Hub.RegisterCommand` (e.g. `/cancel`, `/resume`, `/spawn-like`).
Inbound text starting with `/` (or button payloads with `cmd:` /
`act:cmd:`) parses into `(name, args)` and runs through the registry.
Unknown commands publish `channel.command_unknown` and reply with a
`/help` hint. `/notify off` flips a `muted` field in channel config;
`Hub.dispatch` skips muted channels.

### 4. Bridge protocol (kind=bridge)

A new built-in channel kind, with one row = one adapter slot. The row's
config carries `{name, token, accept_capabilities[]}`. The HTTP server
exposes a public WebSocket endpoint
`/api/v1/channels/bridge/ws` (token-only auth, no admin bearer needed).

An external adapter dials in, sends a `register` frame with platform +
capabilities, and is matched to the bridge channel guarding that token.
From then on the bridge channel implements every capability interface,
gating each by what the adapter claimed; outbound calls become
`send`/`send_card`/`send_buttons`/`update_message`/etc. JSON frames.
Inbound frames (`message`, `card_action`) flow back through the
standard `InboundFunc` path.

Wire format: see `docs/bridge-protocol.md`.

## Consequences

**Positive**
- Channel impls grow incrementally — Telegram already implements every
  optional capability; future Slack/Discord can ship with just `Send`
  and add features one PR at a time.
- New platforms no longer need Go: one bridge channel + a Python /
  Node script is enough. Aligns with the project's primary positioning
  as a third-party API gateway.
- Slash commands give end-users a real interaction surface (resume,
  cancel, mute) without the operator opening the admin web.

**Negative / risks**
- `ChannelMessage.ReplyCtx` is `any` — type-safety is lost across the
  boundary. Each impl casts to its own concrete type. Documented in
  `Channel`'s package doc; tests live per impl.
- Bridge token rotation = mutate the channel row's config. Old tokens
  remain valid until the row is updated. (No revocation list.)
- The bridge WebSocket endpoint is public. Authentication relies on
  the per-bridge token having sufficient entropy — the admin UI
  generates 24 random bytes by default.
- Card schema is informal — element types serialise as their Go struct
  shape (`{Content}`, `{Buttons}`, ...). Adapters must mirror this.
  A formal schema (JSON-Schema or protobuf) is a follow-up.

## Alternatives considered

- **Multi-tenant bridge ("one endpoint, N platforms")** like
  cc-connect — rejected for v1 because it complicates token management
  and per-row introspection. One channel = one adapter is easier to
  reason about; revisit if operators ask for multi-platform single
  binding.
- **Reuse admin bearer for bridge auth** — rejected because bridge
  tokens have a different lifecycle (often longer-lived, often
  embedded in less-trusted adapter scripts). Per-bridge tokens limit
  blast radius if leaked.
- **gRPC instead of WebSocket** — rejected to keep adapter authoring
  one-file simple in any language with WS support. WS already works in
  Python/Node/Go/Rust without code generation.

## Deferred

- Adapter-side `message_ack` frame returning a `preview_handle` so
  `update_message` works without metadata coupling.
- Per-bridge connection metrics (last_seen_at, frames_sent,
  frames_received).
- Card v2 schema with formal JSON-Schema definition.
- Bot-to-bot relay analogue to cc-connect's `cc-connect relay send`
  (would let two opendray sessions on different channels chat).
