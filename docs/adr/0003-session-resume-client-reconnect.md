# ADR 0003 — Session resume: client-reconnect, not PTY supervisor

**Status:** Accepted
**Date:** 2026-04-28
**Decider:** Linivek

## Context

Design §8.1 lists "Resume reconnects to existing PTY without restarting CLI"
as an M1 invariant. Two distinct mechanisms can satisfy that wording:

- **(A) Client-reconnect.** Same opendray process, same in-memory PTY.
  Client (mobile/web) drops the WS, comes back later, reconnects to the
  same `session_id`. The PTY child process never died — it just had no
  live consumer for a while. Ring-buffer replay catches the client up
  on bytes emitted while disconnected.
- **(B) Server-restart durability.** opendray's own process exits and
  restarts; child PTYs survive across the restart. Implementation
  requires a separate supervisor process holding the PTY file
  descriptors (transferred via Unix domain socket `SCM_RIGHTS`) so
  that opendray's restart does not kill its descendants.

The design wording fits both. The product wording from §2 also fits both:

> "I'm at the airport, my agent at home is mid-task. I want to check
> progress and reply on my phone."

This scenario is fully served by (A). The user's phone moving in and out
of network coverage is the dominant disconnect cause; the home server
restarting is rare.

## Decision

M1β implements **(A) only**. (B) is deferred.

Concretely M1β ships:

- `RingBuffer.SnapshotSince(since int64)` — caller's cursor into the
  rolling output. Returns the bytes since the cursor plus a new cursor.
- `GET /api/v1/sessions/{id}/buffer?since=<n>` exposing it; response
  headers `X-OpenDray-Buffer-Start` and `X-OpenDray-Buffer-Cursor`
  let the client detect dropped bytes when the client lagged the ring
  buffer's capacity.
- WS `/stream` continues to replay the full ring on connect (M1α
  behaviour) for naive clients. Cursor-aware clients use `/buffer`
  for catch-up and skip the WS replay if desired.

(B) is not implemented. Design changes that would be required:
- A supervisor process owning PTY fds.
- A Unix domain socket protocol between opendray and the supervisor.
- Re-binding session state on opendray startup by re-reading sessions
  table + reattaching PTY fds via SCM_RIGHTS.
- Process tree management for the supervisor itself (launchd/systemd
  run the supervisor; opendray talks to it).

## Consequences

- A `kill -9 opendray` or graceful shutdown ends every running session.
  The audit_log captures exit_code=-1 for sessions terminated this way.
- Client UX is unaffected for the airport-and-back scenario, which is
  the primary use case from design §2.
- M1β code stays small (~150 LOC for the resume bits) instead of the
  ~800–1500 LOC a supervisor mode would add.
- This decision is reversible: a supervisor process can be added in
  M2+ without changing the gateway / client API. Cursor-aware clients
  will already work; naive clients will see existing sessions resume
  transparently after the upgrade.

## Trigger to revisit

Add (B) only if **both** become true:

1. opendray restarts more than once a week in real use (planned
   redeploys included).
2. At least one user (operator or integration) has reported losing a
   live session because of a restart.

Until then, document the limitation in `docs/operator-guide.md` once
that file lands, and recommend `screen` / `tmux` inside the CLI session
for users who need a stronger guarantee.
