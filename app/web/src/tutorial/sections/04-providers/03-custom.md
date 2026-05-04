# Custom provider manifest

Adding a CLI that opendray doesn't ship is a matter of dropping
one JSON file. No code changes, no rebuild — though you do need
to restart opendray to pick up the new manifest.

## Manifest schema

```json
{
  "$schema": "https://opendray.dev/schemas/manifest-v2.json",
  "id": "myshell",
  "displayName": "Plain Shell",
  "displayName_zh": "纯 Shell",
  "description": "Interactive shell session, no AI assistance.",
  "description_zh": "交互式 shell 会话，无 AI 辅助。",
  "icon": "🐚",
  "version": "1.0.0",
  "kind": "cli",
  "executable": "zsh",
  "args": ["-l"],
  "env": {
    "TERM": "xterm-256color"
  },
  "spawnHint": {
    "cwdPlaceholder": "/Users/me/projects",
    "argsExample": "--login"
  }
}
```

| Field | Purpose |
|---|---|
| `id` | Unique provider id (URL-safe, lowercase) |
| `displayName` / `displayName_zh` | Shown in dropdowns; CJK variant for Chinese locale |
| `description` | One-line description in the same picker |
| `icon` | Single emoji, or `🟣`-style decoration |
| `version` | Free-form; shown in the provider card |
| `kind` | Always `cli` for now |
| `executable` | Path or `$PATH` name |
| `args` | Default args appended to every spawn |
| `env` | Extra env merged on top of the host environment |
| `spawnHint` | UI hints for the spawn dialog |

## Where to put the file

The bundled manifests live inside the Go binary via `embed.FS`,
under `internal/catalog/builtin/<id>.json`. To add a new one in
source you'd:

1. Drop your JSON file in that directory.
2. Rebuild opendray (`go build ./cmd/opendray`).
3. Restart.

For runtime additions without rebuilding, the API supports
posting a manifest:

```bash
curl -X POST -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  http://localhost:8770/api/v1/catalog/providers \
  -d @my-provider.json
```

The catalog package validates the manifest against the schema
and persists the row. The next sync cycle picks it up.

## Validation

opendray runs strict validation on every manifest:

- `id` must be `[a-z0-9-]+`, 2–40 chars
- `executable` must resolve at startup; missing binaries cause
  the provider to be marked **unavailable** (greyed-out in the
  dropdown).
- Unknown top-level keys cause the manifest to be **rejected**
  outright.

Check the server log on startup:

```
INFO catalog synced count=4
WARN provider unavailable id=myshell err="executable not found in $PATH: zsh"
```

## Runtime overrides

The web UI lets you patch `executable` / `args` / `env` /
`displayName` / `disabled` per host without editing the source
manifest. Overrides live in the DB and apply on top of the
bundled defaults — handy when the same opendray binary runs on
multiple hosts with different filesystem layouts.

The **Reset** button on the provider card drops your overrides
and reverts to the manifest's bundled values.
