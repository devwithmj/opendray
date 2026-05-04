# MCP servers

Model Context Protocol (MCP) is the standard for exposing tool
catalogues to AI CLIs. opendray manages MCP server registrations,
their config, and a per-host encrypted secrets vault that holds
API keys MCP servers need at runtime.

## What you can register

Any MCP server — there are dozens in the public catalogue:

- **Filesystem** — read/write under a specific root
- **Git** — query repos
- **Linear / Asana / Trello** — read tickets
- **Brave Search / Tavily** — web search
- **Slack / Discord** — read channels
- And whatever you write yourself

Each server is one entry in opendray's MCP registry.

## Adding a server

Plugins → **MCP servers** → **Add server**.

| Field | Purpose |
|---|---|
| **Id** | URL-safe, e.g. `linear`, `fs-projects` |
| **Display name** | shown in the registry |
| **Command** | how to launch the server (e.g. `npx -y @modelcontextprotocol/server-linear`) |
| **Args** | extra args |
| **Env** | env vars to inject (use `$SECRET_NAME` to interpolate from the secrets vault) |
| **Enabled** | hide / show in session spawn |

![MCP server registration](/tutorial/plugins-mcp-add.png)

## Secrets vault

API keys for MCP servers live in `~/.opendray/secrets.env` —
encrypted with AES-GCM, key stored in the OS keychain (macOS
Keychain, Linux secret-service, Windows Credential Manager).

To add a secret:

1. Plugins → **MCP secrets** → **Add secret**.
2. Enter name (e.g. `LINEAR_API_KEY`) + value.
3. Save. The plaintext writes to the encrypted blob; the form
   field clears.

Reference in MCP server env:

```
LINEAR_API_KEY=$LINEAR_API_KEY
```

opendray substitutes from the vault at MCP launch time. The
plaintext never lands in the registry's plaintext config — only
the variable name does, so the registry can be safely committed
or backed up alongside the rest of opendray's state.

## How sessions see the MCP servers

When a Claude session spawns, opendray:

1. Generates `~/.claude/mcp.json` for that session's
   `CLAUDE_CONFIG_DIR` containing all enabled MCP servers
   (with their resolved env, secrets included).
2. The server processes start lazily when Claude asks for them.
3. The `mcp.json` is regenerated on every spawn — edits in the
   admin UI propagate to new sessions automatically. Running
   sessions need a `/mcp restart` (Claude command) to pick up
   server-level changes.

## Per-session MCP overrides

Future feature — for now, MCP is host-wide. If you need to
disable a specific server for one session, set it `disabled` in
the registry, spawn the session, then re-enable.

## Health probes

Every 60s opendray runs `mcp <server> ping` (a no-op tool
call) and tracks the result. The MCP page shows:

- 🟢 *Healthy* — last probe within 60s, success
- 🟡 *Slow* — last probe took > 5s
- 🔴 *Unreachable* — last 3 probes failed

Click a row to see the last probe's stderr.

## Limitations

- **Stdio transport only.** MCP supports stdio + WebSocket; only
  stdio is wired in. WS would let you point at a remote MCP
  server, which v1 doesn't support.
- **No per-session config**. Every Claude session inherits the
  same MCP catalogue.
- **Concurrent rate limits** on shared API quotas can interfere
  when multiple sessions hit the same MCP simultaneously.
  opendray doesn't fan-in or queue.
