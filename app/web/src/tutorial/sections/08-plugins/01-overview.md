# Plugins — overview

The Plugins page registers tool extensions that opendray injects
into compatible CLI sessions on spawn. Three plugin types live
here:

| Plugin type | What it does |
|---|---|
| **Skills** | Reusable prompts / patterns (markdown files) Claude can invoke as slash commands |
| **MCP servers** | Model Context Protocol servers exposing tool catalogues to compatible CLIs |
| **Git hosts** | Credentials for remote git pushes (vault sync, codebase clones) — used by the Notes vault and the integrations side |

![Plugins page](/tutorial/plugins-layout.png)

## Read on

| Topic | Section |
|---|---|
| Skills directory + slash command registration | Skills |
| MCP server catalogue + secrets vault | MCP |
| SSH keys / HTTPS PATs for git remotes | Git hosts |

## Where plugin state lives

| Plugin | Storage |
|---|---|
| Skills | Markdown files under `~/.opendray/vault/skills/` (vault-managed, syncs via git) |
| MCP servers | JSON config in `~/.opendray/mcp/servers.json` |
| MCP secrets | Encrypted file `~/.opendray/secrets.env` (key in OS keychain) |
| Git hosts | Postgres `git_hosts` table; HTTPS tokens in the same secrets vault |

The split is intentional: skill content lives in the vault (so
multi-host sync just works), but secret material stays per-host
in the OS-keychain-protected vault.
