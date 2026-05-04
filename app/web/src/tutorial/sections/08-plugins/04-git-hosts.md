# Git hosts

Credentials for remote git pushes. The Notes vault sync uses
these tokens to push the vault repo over HTTPS without prompting
for credentials. Other parts of opendray that need to clone or
push (custom integrations) reuse the same registry.

## Provider flavours

| Provider | Auth flavour |
|---|---|
| GitHub | Personal Access Token (classic or fine-grained) |
| GitLab | Personal Access Token with `write_repository` scope |
| Gitea / Forgejo | application token with repo-write |
| Bitbucket | App password with repo-write |
| Custom (SSH-only) | SSH key trusted by the host's `~/.ssh/known_hosts` — no token needed; this page is for HTTPS auth only |

## Adding a host

Plugins → **Git hosts** → **Add host**.

| Field | Purpose |
|---|---|
| **Provider** | dropdown picking known patterns (sets default URL prefix) |
| **Display name** | shown in the list |
| **HTTPS host** | e.g. `github.com`, `gitlab.com`, `gitea.example.com` |
| **Username** | your username on the platform |
| **Token** | the PAT / app token / app password |

The token writes to the same encrypted secrets vault as
[MCP secrets](#plugins-mcp). You can't read the plaintext back
through the UI; rotate by saving a new value.

## How tokens get used

The vault syncer (`internal/vaultgit/syncer.go`) when invoking
git push/pull:

1. Looks up the remote URL.
2. Matches the host portion against the registered git hosts.
3. If found, prepends `https://<user>:<token>@<host>/...` to
   the URL for the duration of that operation.
4. Token never lands in `~/.opendray/vault/.git/config` — only
   the bare URL does, so the repo can be cloned by a different
   host without leaking credentials.

## Listing repos (read-only feature)

Plugins → **Git hosts** → click a host → **List repos**.

opendray hits the host's API (`/user/repos` for GitHub, etc.)
with the configured token and shows your accessible repositories.
Useful sanity check ("am I auth'd?") and the seed for *Clone
this repo as a new session cwd* in future versions.

## Token rotation

Each host card shows the **last used** timestamp. When that's
old (90+ days), GitHub-style tokens may have been auto-revoked
on the platform side. The vault syncer surfaces 401s by flipping
the host card status to red and emitting a notification on the
event bus (`vaultgit.host_auth_failed`).

To rotate:

1. Generate a new token on the platform.
2. Plugins → **Git hosts** → click the host → **Rotate token**.
3. Save.

Old token stays in the encrypted vault until next sync; first
successful sync with the new value clears the old.

## SSH-only setups

If you prefer SSH (no tokens at all):

- Configure the host's SSH key normally (`~/.ssh/config` + agent).
- Use SSH-format remotes (`git@github.com:me/vault.git`).
- Skip the Git hosts page entirely — opendray's syncer shells
  out to system `git`, which uses your SSH agent.

The Git hosts page is **specifically for HTTPS-with-token**
setups, which are friendlier on hosts where SSH agents are a
pain (Docker containers, Windows, ephemeral cloud VMs).
