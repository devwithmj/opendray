# Providers — overview

A *provider* is the catalogued definition of one CLI binary
opendray can spawn — Claude Code, OpenAI Codex, Google Gemini,
and any custom CLI you drop a manifest for. Each session row
points at a provider id, so when you "restart in place" the
provider's exec path / args / env apply to the new process.

## What's on the page

![Providers page](/tutorial/providers-layout.png)

Two panels:

1. **Provider list** — every catalogued CLI, with edit / disable
   buttons. The icon, display name, and description come from
   the provider's manifest.
2. **Claude accounts** — multi-account binding helper specific
   to Claude Code (other providers use a single credential set
   from env vars).

## What's configurable per provider

Each provider has these editable fields:

| Field | Purpose |
|---|---|
| **Executable** | Absolute path or `$PATH`-resolved name (e.g. `claude`, `/usr/local/bin/codex`) |
| **Default args** | Appended to every spawn before user-supplied args |
| **Environment** | Extra env vars merged into the process environment |
| **Display name + icon** | Shown in the spawn dropdown and tab strip |
| **Working-dir hint** | Pre-fills the spawn dialog's cwd field |
| **Disabled** | Hides this provider from the spawn dropdown |

The bundled JSON manifests live in `internal/catalog/builtin/`
inside the binary. The web UI lets you override fields at runtime
without editing the source manifest — overrides are stored in the
DB and apply on top of the bundled defaults.

## Reset to defaults

Every provider card has a **Reset** button that drops your
runtime overrides and restores the bundled manifest values.
Useful when an experiment goes wrong and you want a clean slate
without restarting opendray.

## Read on for specifics

| Topic | Section |
|---|---|
| Bundled providers and their gotchas | Bundled providers |
| Adding a custom provider via manifest | Custom provider manifest |
| Multi-account Claude setup | Claude accounts |
