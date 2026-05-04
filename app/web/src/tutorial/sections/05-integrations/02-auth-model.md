# Auth model

opendray's API has two token types, applied through a dual-auth
middleware. Understanding the precedence + scoping prevents most
"why is this call returning 401?" debugging sessions.

## Admin token (full access)

The admin token is set at startup via:

- `[auth].password` (legacy — actually the bearer for `/login`)
- env var `OPENDRAY_ADMIN_TOKEN` for non-web callers
- Or generated and shown once on first run

Used by:

- The web admin UI (after login)
- Any admin-CLI script

The admin token bypasses **every** auth check. With it, you can
hit any `/api/v1/*` endpoint, including destructive ones like
DELETE on a session row.

Rotate from **Settings → Auth → Rotate admin password**. The new
value invalidates every web session immediately — operators get
kicked out and need to re-login.

## Integration keys (scoped)

Per-integration bearer tokens. Created from the Integrations page:

1. **New integration** → name (e.g. `grafana-webhook-receiver`).
2. opendray generates a random 32-byte hex key, hashes it, and
   stores only the hash. The plaintext is shown **once** in a
   modal — copy it now or rotate immediately.
3. Pick scopes (which `/api/v1/*` paths the key may hit). Default
   is "all read-only routes" — narrow it for production keys.
4. Save.

![Integration key reveal modal](/tutorial/integration-key-reveal.png)

The card shows the key id (visible) + a masked preview of the
plaintext. **Rotate** generates a new plaintext and invalidates
the old one immediately.

## How the middleware decides

Request hits opendray with `Authorization: Bearer <token>`.

1. `auth.Middleware` checks if the token equals the admin token
   → set `principal=admin`, continue.
2. Otherwise call `integration.Service.AuthenticateKey(token)`.
   Looks up the hash in the integration table. If found and
   the integration is `enabled` → set `principal=integration:<id>`.
3. Otherwise → 401.

Once the principal is set, the **call logger middleware** wraps
the response handler. After the response is written, it appends
a row to `integration_call_log` with:

- principal (admin or integration id)
- request method + path
- response status code
- duration
- request id (for correlation with the structured log)

Admin requests are **excluded** from the call log by default —
the table is for tracking external tools, not the operator
clicking around the admin.

## Scoping integration keys

Three scope models supported:

| Scope | Meaning |
|---|---|
| `read` | GET requests to all `/api/v1/*` |
| `write` | + POST/PATCH/DELETE on non-destructive routes |
| `admin` | Equivalent to admin token (use sparingly) |

The future direction is per-route ACLs but for now the three
levels cover most cases. Hard-restrict with reverse-proxy mounts
when you need a single endpoint exposed to a single integration.

## Common gotchas

- **401 right after rotate** — the rotated key invalidates the
  old one immediately. Update the consuming tool's config.
- **Two integrations sharing the same name** — the *display
  name* allows duplicates, but the *id* is unique. The list
  shows both; rotate / archive the duplicate.
- **Token in URL** — opendray accepts `?token=` query for the
  events WebSocket only (browsers can't set headers on a WS
  upgrade). For all other endpoints use the Authorization
  header.
- **Trailing whitespace** — copying from a terminal often
  appends a newline. The middleware trims, but verify when
  debugging "wrong" tokens.
