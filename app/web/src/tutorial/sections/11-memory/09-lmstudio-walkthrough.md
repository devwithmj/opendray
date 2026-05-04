# LM Studio walkthrough

LM Studio is an alternative to ollama with a GUI for managing
local models. opendray's HTTP backend works against LM Studio
identically — the only difference is the default port.

## Why pick LM Studio over ollama

| | LM Studio | ollama |
|---|---|---|
| GUI for browsing / loading models | ✅ | ❌ (CLI only) |
| Default port | `1234` | `11434` |
| Mac silicon optimised | ✅ (MLX builds available) | ✅ |
| GGUF / MLX support | both | GGUF |
| systemd / launchd background | manual | `brew services` / `systemctl` |
| Model search inside app | ✅ | external `ollama search` |

Operationally identical from opendray's view. Pick whichever GUI
you prefer; you can run both side-by-side too.

## Step 1 · Install LM Studio

Download the macOS / Linux / Windows installer from
<https://lmstudio.ai>. Open it. The first launch walks through
picking a model.

## Step 2 · Load an embedding model

In the LM Studio app:

1. **Search** tab → search `nomic-embed-text` (or `bge-m3`, or
   `qwen3-embedding`).
2. Pick a quantization (Q4_K_M is a sensible default — fast +
   small).
3. Download.
4. **Local Server** tab → click the model dropdown → select your
   downloaded embedding model.
5. **Start Server** button. Default port is 1234.

Verify from a terminal:

```bash
curl http://localhost:1234/v1/models | jq '.data[].id'
```

Should list at least the model you loaded.

## Step 3 · Configure opendray

Settings → Server → Memory:

```
Backend:               http
Similarity threshold:  0.5
```

Under HTTP backend, two options:

**Option A — click the "Auto-detected" badge**
opendray probes both ports at startup. If LM Studio was running,
you'll see a green badge above the form: `lmstudio · http://localhost:1234/v1 (N models)`. Click it → base URL + first
embedding-looking model auto-fill.

**Option B — preset button**

```
Click the [LM Studio] preset → fills http://localhost:1234/v1
Type the model id (e.g. text-embedding-nomic-embed-text-v1.5)
Leave API key blank
```

Click **Test connection** to confirm. Then **Save changes** +
**Restart server**.

After restart, status strip shows:

```
http:text-embedding-nomic-embed-text-v1.5 · 768-dim · enabled
```

## Step 4 · Verify

Same flow as ollama. Test embedder roundtrips, store a memory
from one CLI, search from another. Cross-CLI memory works.

## Tuning

Same model-specific threshold table as ollama:

| Model family | Suggested threshold |
|---|---|
| nomic-embed-text-v1.5 | 0.5 |
| qwen3-embedding-0.6b | 0.5 |
| qwen3-embedding-8b | 0.55 |
| mxbai-embed-large | 0.55 |
| bge-m3 | 0.6 |

LM Studio shows model latency in its server log — `~30ms` for
nomic, `~80ms` for bge-m3 on M-series silicon (matches ollama
roughly).

## Switching between ollama and LM Studio

You can have both daemons running. opendray's auto-detect surfaces
both — switch by clicking whichever you want. Memory rows tagged
with the previous embedder name **stay searchable when you switch
back** (we filter by embedder name to keep cosine math honest), so
A/B testing is harmless.

To wipe and re-embed under a new model: delete via the Inspector
(small datasets) or run a SQL `DELETE FROM memories WHERE
embedder = 'http:old-model-name'`.

## Troubleshooting

**Test connection returns "unreachable"**

LM Studio's server isn't running. Open the app → Local Server
tab → Start Server. Confirm `curl http://localhost:1234/v1/models`
works.

**Model loads but every embedding call returns empty vectors**

You loaded a chat model, not an embedding model. Embedding model
ids start with `text-embedding-` in LM Studio's list. Stop the
server, switch the loaded model, restart server.

**Auto-detect shows LM Studio but I prefer ollama**

The presets are a starting point; click the **ollama** preset
manually after the auto-detect badge to override. Or just edit
the base URL field directly.

**LM Studio crashes on first call after long idle**

LM Studio unloads models after inactivity (default 5 min). First
call after unload triggers a 1-3s reload. opendray's HTTP backend
has a 30s timeout, so it survives — but the agent waiting on
that call sees the latency. Configure LM Studio's "Keep model
loaded" setting in the server panel.
