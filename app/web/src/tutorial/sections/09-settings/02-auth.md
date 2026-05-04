# Auth

The auth model is covered in detail under [Integrations →
Auth model](#integrations-auth-model). This section is the
operator-side view: where to rotate things, where the
fallback for "I'm locked out" lives, and what to do on first
install.

## First-install setup

opendray's first-run flow sets the admin password from one of
three sources, in priority order:

1. `OPENDRAY_ADMIN_PASSWORD` env var (overrides everything)
2. `[auth].password` in `config.toml`
3. Random — generated, written to log, also surfaced in the
   first-run wizard if you have a browser open

The web UI's `/login` page accepts username `admin` plus the
password from any of those sources.

## Rotating the admin password

Settings → **Auth** → **Rotate admin password**.

1. Enter new password (min 12 chars).
2. opendray hashes it (bcrypt cost 10) and persists in the
   `auth_credentials` table.
3. **All web sessions are kicked** — every existing browser
   session tab needs to re-login.
4. Integration keys are unaffected.

If you're locked out (lost password, no env override):

```bash
# Stop opendray, then:
OPENDRAY_ADMIN_PASSWORD=new-temporary-password ./opendray serve -config config.toml
```

The env var wins. Login with the temporary password, rotate to
something proper from the UI, restart without the env var.

## Integration tokens

The Settings page links to **Integrations** for token management
— the table of integration keys lives there, not here. See
[Integrations → Auth model](#integrations-auth-model) for the
rotation flow.

## API token for the admin user

Some external scripts want a long-lived bearer they can rely on
without going through `/login`. Settings → **Auth** → **Generate
API token** creates one bound to the admin account:

| Field | Notes |
|---|---|
| Name | label, shown in the audit log |
| Expires | optional — `never` is allowed but discouraged |
| Scopes | always `admin` — full access |

Use the token in `Authorization: Bearer <token>` headers, same
as any integration key. Revoke any time via **Revoke** on the
token's row.

## Audit log

Every admin action lands in `audit_log` (separate from
`integration_call_log` — that's per-call, this is per-action):

- Auth: login, logout, password rotate, token create/revoke
- Channels: create, update, delete
- Providers: manifest override, claude account add
- Sessions: spawn, stop, restart, delete row
- Integrations: create, rotate, archive
- Plugins: skill add, MCP server add, secret add/rotate
- Notes: vault sync events

Each row has actor, action, subject, timestamp, optional
metadata blob. Search + filter at Settings → **Audit log**.

For external SIEM ingestion subscribe to `audit.event` on the
[Events WebSocket](#integrations-events-ws).
