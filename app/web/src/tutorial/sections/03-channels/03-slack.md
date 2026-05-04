# Slack

**Mode:** Socket Mode (no public URL)
**Capabilities:** text · card (Block Kit) · buttons · update_message · reply_to_message (thread_ts)
**Setup time:** ~10 minutes (Slack admin console has many tabs)

Slack's Socket Mode lets bots open an outbound WebSocket back to
Slack instead of receiving webhooks — opendray can run behind a NAT
without exposing anything publicly.

## 1. Create a Slack app

1. Visit [api.slack.com/apps](https://api.slack.com/apps) →
   **Create New App** → *From scratch*.
2. Name it (e.g. `OpenDray`) and pick the target workspace.
3. After creation you land on the app's *Basic Information* page —
   keep it open, you'll come back.

![Slack app create flow](/tutorial/slack-app-create.png)

## 2. Enable Socket Mode + create the App-Level token

1. Sidebar → **Socket Mode** → toggle on.
2. Slack prompts you to create an *App-Level Token*:
   - **Name:** `opendray-socket`
   - **Scope:** `connections:write`
   - **Generate**.
3. Copy the **xapp-…** token. This is the **App-Level Token**.

## 3. Add bot OAuth scopes

Sidebar → **OAuth & Permissions** → scroll to *Bot Token Scopes* →
**Add an OAuth Scope** → add at minimum:

- `chat:write` — post messages
- `channels:history` — read messages from channels the bot is in
- `groups:history` — same for private channels
- `im:history` — same for DMs

Optional but recommended:

- `chat:write.public` — post into channels the bot isn't a member
  of (handy for `#general`-style notifications)

Then scroll up and click **Install to Workspace**. Approve.

After install, copy the **Bot User OAuth Token** (starts with
`xoxb-`).

## 4. Subscribe to events

Sidebar → **Event Subscriptions** → toggle on. (Socket Mode delivers
events over the WS — no Request URL to enter.)

Under *Subscribe to bot events* add:

- `message.channels` — messages in public channels
- `message.groups` — messages in private channels
- `message.im` — DMs

Save changes.

## 5. Enable interactivity

Sidebar → **Interactivity & Shortcuts** → toggle on. (No URL needed
in Socket Mode.) Required for button clicks to flow back.

## 6. Invite the bot to a channel

In Slack: open the target channel and run `/invite @OpenDray`.

Right-click the channel → **View channel details** → scroll to
**Channel ID** at the bottom. Copy it (looks like `C0123ABC456`).

![Slack channel ID](/tutorial/slack-channel-id.png)

## 7. Configure in opendray

Channels → **New channel** → kind **Slack**.

| Field | Value |
|---|---|
| **Bot token (xoxb-…)** | from step 3 |
| **App-level token (xapp-…)** | from step 2 |
| **Default channel ID** | from step 6 |

Save with **Enabled = on**.

## 8. Verify

- Server log should show `slack socket-mode connected`.
- Card flips to `RUNNING`.
- Hit **Test** → message appears in the channel.
- DM the bot `/help` — opendray replies in-thread.

## Block Kit rendering

opendray converts the internal Card model to Slack Block Kit blocks:

- `CardHeader` → `header` block (large bold)
- `CardMarkdown` → `section` with `mrkdwn` text type
- `CardDivider` → `divider`
- `CardActions` → `actions` block with `button` elements; styles
  `primary`/`danger` map to Slack's primary/danger button styles
- `CardListItem` → section with accessory button
- `CardSelect` → `static_select` element
- `CardNote` → `context` block (small grey footer)

Threading: when a session reply lands as a thread under the
notification message, opendray sends `thread_ts` so the entire
back-and-forth stays in one thread.

## Limitations

- Slack's `mrkdwn` ≠ standard Markdown:
  - bold = `*text*` (single asterisk, **not** `**`)
  - italic = `_text_`
  - link = `<https://url|label>`
  - **Headings (`#`) and tables don't render** — they appear
    literally
- The Free plan caps message history; older notifications may
  disappear from search.
- Apps in `public_distribution` mode (sharing via App Directory)
  require an extra review process — keep the app private until
  you're sure.
