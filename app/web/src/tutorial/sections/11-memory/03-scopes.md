# Scopes

Every memory carries a `scope` and a `scope_key` that determine
who can read it. Pick the right scope for what you're storing —
the choice trades **sharing** against **isolation**.

## The three scopes

| Scope | scope_key | Visibility |
|---|---|---|
| `session` | the session id | Only the session that wrote it |
| `project` *(default)* | the session's `cwd` | Every session in the same `cwd` (across CLIs) |
| `global` | (empty / unused) | Every session ever, anywhere |

## When to pick which

### Use `session` for

- Throwaway notes that shouldn't survive the conversation
- Sensitive context the operator wants self-cleaned (e.g. a token
  rotated mid-session)
- Tests that need isolation from real project memory

Effectively a private scratchpad — once the session ends, the
memories are still in the DB but no one will ever query them
under that scope_key (sessions are id'd uniquely).

### Use `project` for (default)

- Anything tied to a working directory: package-manager
  preferences, build commands, repo conventions, important
  filenames, ongoing tasks
- Cross-CLI handoff: Claude tells you something while implementing
  a feature → Codex picks up where Claude left off in the same
  `cwd`

This is the scope opendray's auto-attach uses by default and the
one users almost always want.

### Use `global` for

- Operator-level identity: "I'm based in Sydney", "I prefer
  TypeScript over JavaScript everywhere", "my GitHub username is
  …"
- Things that should follow you across every project

Be careful: global memories are visible to ANY session opendray
spawns, including ones in unrelated repos. If you have client
work in different projects, project scope is safer.

## Changing the default

Settings → Server → Memory → **Default scope** picks the scope used
when an agent calls `memory_store` without specifying. Restart
required (the default is read once at app startup).

You can also override per call: agents can pass `scope=global` /
`scope=session` to `memory_store` directly, but in practice they
use whatever default opendray's system-prompt guidance recommends.

## How scope_key works under the hood

`memory_store` flows through:

1. opendray-memory MCP subprocess receives the call from the agent
2. Subprocess pulls `scope` and `scope_key` from its env vars
   (`OPENDRAY_MEMORY_SCOPE`, `OPENDRAY_MEMORY_SCOPE_KEY` — set by
   the gateway when it rendered the session's mcp.json)
3. POSTs `/api/v1/memory/store` with `{text, scope, scope_key}`
4. Backend stores the row

The agent never knows or types the scope_key — it's filled in by
the gateway based on the session's cwd. This is intentional:
prevents the agent from writing into a scope it shouldn't.

## Querying across scopes

`memory_search` is single-scope. The MCP tool only exposes one
scope at a time (whatever the env var says). If you need to query
both project + global from the agent, you currently need two
calls. Phase 2 may add a "search-many-scopes" mode.

The Settings UI's Inspector lets the operator switch scope freely
since it's running in the admin context.

## Listing what's stored where

Settings → Server → Memory → Inspector. Pick a scope, type a
scope_key (defaults to the first session's cwd), and you'll see
every memory in that scope with provenance metadata.

Or via the API directly:

```bash
curl -s "http://127.0.0.1:8770/api/v1/memory/list?scope=project&scope_key=/path/to/cwd&n=50" \
  -H "Authorization: Bearer $TOKEN" | jq
```

## Privacy boundary

Scope is the **only** isolation mechanism — there's no per-user
filtering today (opendray is single-operator by design). If you
share an opendray instance with collaborators, they all see all
project + global memories. Phase 2 may add per-operator filtering.
