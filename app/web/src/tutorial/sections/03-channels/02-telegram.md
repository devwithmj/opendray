# Telegram

**Mode:** long-poll (no public URL)
**Capabilities:** text · card · buttons · update_message · typing · reply_to_message
**Setup time:** ~3 minutes

This is the fastest channel to set up because Telegram lets bots
long-poll their own API — no webhook, no public host required.

## 1. Create a bot via BotFather

1. In Telegram, search for [@BotFather](https://t.me/BotFather) and
   start a chat.
2. Send `/newbot`. BotFather walks you through:
   - **Name** — display name shown in the chat (e.g. `OpenDray`).
   - **Username** — must end with `bot` (e.g. `mycompany_opendray_bot`).
3. BotFather replies with a token like
   `123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11`. **Copy it** —
   you'll paste it into opendray.

![BotFather token reveal](/tutorial/telegram-botfather-token.png)

## 2. Find your chat ID

You need to know which chat (private DM or group) opendray should
default to.

**Easy path:**

1. Add the bot to a group, OR open a private chat with it.
2. Send any message in that chat (e.g. `hi`).
3. Open `https://api.telegram.org/bot<TOKEN>/getUpdates` in a
   browser, replacing `<TOKEN>` with your bot token.
4. Find `"chat":{"id":...}` in the JSON. Positive number = DM,
   negative number = group, larger negative starting with `-100`
   = supergroup.

```
https://api.telegram.org/bot123456:ABC-DEF.../getUpdates

{"ok":true,"result":[{
  "update_id":...,
  "message":{
    "chat":{"id":7831238986,"type":"private"},
    ...
  }
}]}
```

That `7831238986` is your chat id.

## 3. (Optional) Restrict the bot to groups only

By default the bot can be added to any group. To lock it to your
group only:

- BotFather → `/setjoingroups` → pick your bot → **Disable**.

## 4. Configure in opendray

Channels → **New channel** → kind **Telegram**.

| Field | Value |
|---|---|
| **Bot token** | the BotFather token from step 1 |
| **Default chat ID** | the chat id from step 2 (optional — used for outbound when no `ReplyCtx` is present) |
| Repeat policy | leave at "Once per session" |
| Terminal snippet | leave on, "No cap" |

Save with **Enabled = on**.

![New Telegram channel form](/tutorial/telegram-new-channel.png)

## 5. Verify

The card flips to `RUNNING` after a few seconds. Hit **Test**
on the card — you should see *"OpenDray channel test ✓"* in the
chat.

Send `/help` to the bot in chat — opendray replies with the list of
registered commands. That confirms inbound polling works.

## 6. (Optional) Add slash-command autocomplete

Telegram clients show a hint dropdown for known commands. Tell
BotFather what they are:

- `/setcommands` → pick the bot → paste:

```
help - List available commands
status - Show channel status and capabilities
notify - Toggle notifications: /notify on|off
sessions - List recently-notified sessions
select - Pin a session for replies
cancel - End a session
resume - Reply to resume a session
```

This is purely cosmetic — opendray accepts the commands either way.

## Limitations

- Bot tokens are bearer credentials. Anyone with the token can
  speak as your bot. If exposed: BotFather → `/revoke` → choose the
  bot → confirm.
- Inline-button `callback_data` is capped at 64 bytes by Telegram.
  opendray's command payloads (`cmd:/cancel <session-id>`) fit.
- Voice messages, stickers, locations — opendray doesn't decode
  these. Only text + button clicks become inbound to a session.

## Troubleshooting

**"telegram: getUpdates failed; backing off"** in the server log:
- Token wrong (paste again, no leading/trailing spaces)
- Or two opendray instances running with the same token —
  Telegram's getUpdates is single-consumer and rejects with 409.
  Stop the duplicate.

**Bot doesn't see messages in a group**:
- BotFather → `/setprivacy` → pick the bot → **Disable** (default
  is "Enable" which means the bot only sees commands and
  @mentions).
