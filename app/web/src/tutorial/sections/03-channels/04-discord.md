# Discord

**Mode:** Gateway WebSocket (no public URL)
**Capabilities:** text · card (embeds) · buttons (components) · update_message · reply_to_message (message_reference)
**Setup time:** ~5 minutes

Discord uses a persistent Gateway WebSocket for inbound and a REST
API for outbound. The bot identifies once on connect; reconnects
are handled automatically by opendray.

## 1. Create a Discord application + bot

1. Visit
   [discord.com/developers/applications](https://discord.com/developers/applications)
   → **New Application** → name it (e.g. `OpenDray`).
2. Sidebar → **Bot** → **Reset Token** → confirm. Copy the token
   (looks like `MTIzNDU2.AbCdEf.…`). **You only see it once.** Save
   it somewhere safe before navigating away.

![Discord bot token reveal](/tutorial/discord-token.png)

## 2. Enable the Message Content Intent

Same Bot page, scroll to **Privileged Gateway Intents**:

- ✅ **Message Content Intent** — REQUIRED. Without this, every
  inbound message arrives with `content=""` and the bot is
  effectively useless.
- ✅ **Server Members Intent** — only needed if you want member
  metadata; safe to leave on.

Hit **Save Changes**.

> Bots in 100+ servers need explicit verification from Discord
> before they can use privileged intents. Single-server setups
> (most opendray users) are unrestricted.

## 3. Invite the bot to your server

1. Sidebar → **OAuth2** → **URL Generator**.
2. Scopes: ✅ `bot`, ✅ `applications.commands` (the second is
   needed if you ever register slash commands; harmless if not).
3. Bot Permissions: at minimum
   - `Send Messages`
   - `Embed Links`
   - `Read Message History`
   - `Use External Emojis` (renders the `●` / `└` markers nicely)
4. Copy the generated URL → open it in a browser → pick the server
   → **Authorize**.

The bot now appears in the server's member list (offline until
opendray connects).

## 4. Find the channel ID

1. Discord → User Settings (gear icon) → **Advanced** → enable
   **Developer Mode**.
2. Right-click any channel → **Copy Channel ID**.

The id looks like `1234567890123456789` (snowflake — long number).

## 5. Configure in opendray

Channels → **New channel** → kind **Discord**.

| Field | Value |
|---|---|
| **Bot token** | from step 1 |
| **Default channel ID** | from step 4 |

Save with **Enabled = on**.

## 6. Verify

- Server log shows `discord channel started` followed by the
  Gateway handshake (`READY` event).
- Bot status flips to **online** in the Discord member list.
- Hit **Test** on the card → bot posts a message.
- Type `/help` in any channel where the bot can read messages —
  opendray replies inline.

![Discord card with action buttons](/tutorial/discord-card-buttons.png)

## Embed + components rendering

- `CardHeader` → embed `title` + `color` (green / red / yellow /
  …; opendray maps named colours to RGB hex codes per `colorMap` in
  `internal/channel/discord/discord.go`)
- `CardMarkdown` → embed `description`
- `CardActions` → message `components` array of `action_row`s with
  `button` elements (style 1=primary, 4=danger, 2=secondary)
- `CardListItem` → embed `field` + a button row
- `CardSelect` → string select component
- `CardNote` → embed `footer.text`

`custom_id` on each button is what opendray emits as `cmd:/foo` —
well below Discord's 100-char limit.

## Limitations

- Bot tokens are highly sensitive: a leaked token lets anyone
  control the bot in every server it's joined. Reset from the dev
  portal if exposure is suspected.
- Discord's content moderation may filter messages — particularly
  ones that look like phishing (lots of links). Notifications from
  opendray rarely trip it but worth knowing.
- Embeds have a 6000-char total cap (title + description + fields).
  Long Claude responses get split into multiple embed messages
  automatically.
