# Bridge — custom platforms via WebSocket

When you need a platform opendray doesn't bundle (LINE, KakaoTalk,
your company's internal chat, …), the **bridge** kind exposes a
WebSocket protocol an external adapter can speak. Run the adapter
in any language — Python, Node, Rust — and it appears as a regular
Channel inside opendray.

**Setup time:** depends on how complex the target platform is. The
opendray side is 30 seconds (a token). The adapter side is whatever
the platform demands.

## When to reach for bridge

- The target platform isn't in the bundled list
- You need custom routing logic between users + sessions
- You want to integrate a webhook source other than messaging
  (cron triggers, build status, etc.)

## 1. Create a bridge channel slot in opendray

Channels → **New channel** → kind **bridge**.

| Field | Value |
|---|---|
| **Bridge name** | human label, e.g. `wechat`, `discord-custom`, `whatsapp` |
| **Adapter token** | auto-generated 24-byte random hex; click ↻ to regenerate or 📋 to copy |
| **Accept capabilities** | optional whitelist — when empty, the bridge accepts whatever the adapter declares; when non-empty, only the selected capabilities are honoured |

Save with **Enabled = on**.

![Bridge channel form](/tutorial/bridge-create-form.png)

After saving, the **Adapter setup** dialog opens automatically with
the WebSocket URL + ready-to-paste Python / Node / wscat starter
snippets.

![Bridge adapter setup with code snippets](/tutorial/bridge-adapter-setup.png)

You can re-open this dialog any time via the **Setup** button on
the channel card.

## 2. Run the adapter

Connect to the WebSocket URL. Auth is the bridge token via:

| Method | Example |
|---|---|
| Header | `X-Bridge-Token: <token>` |
| Header | `Authorization: Bearer <token>` |
| Query | `?token=<token>` |
| First WS frame | `{"type":"register", "token":"…", …}` |

The first frame **must** be a `register` declaring the adapter's
identity and capabilities:

```json
{
  "type": "register",
  "platform": "wechat-custom",
  "capabilities": ["text", "card", "buttons", "image"],
  "metadata": { "version": "1.0.0" }
}
```

opendray replies with `{"type":"register_ack","ok":true}` (or
`ok:false` with an `error` field).

## 3. Inbound: adapter → opendray

When a user sends a message in the upstream platform, the adapter
translates it into:

```json
{
  "type": "message",
  "session_key": "wechat-custom:gid42:user123",
  "conversation_id": "gid42",
  "user_id": "user123",
  "user_name": "Alice",
  "text": "Hello opendray",
  "reply_ctx": "<adapter-opaque-handle>"
}
```

`reply_ctx` is whatever the adapter needs to send a reply back
later — opendray echoes it on every outbound frame. Often it's
the platform's message id, but the adapter chooses the format.

For button clicks (when the adapter supports cards):

```json
{
  "type": "card_action",
  "session_key": "...",
  "conversation_id": "...",
  "action": "cmd:/cancel sess1",
  "reply_ctx": "..."
}
```

opendray's Hub recognises `cmd:/...` actions and dispatches them
through the slash-command registry.

## 4. Outbound: opendray → adapter

When a session goes idle (or any other event the channel notifies
on), opendray sends frames the adapter must render in the upstream
platform:

```json
{ "type": "send", "session_key": "...", "reply_ctx": "...", "text": "Acknowledged." }

{ "type": "send_card",
  "session_key": "...",
  "card": {
    "header": { "title": "Session idle", "color": "yellow" },
    "elements": [
      { "Content": "Session abc went idle." },
      { "Buttons": [[
        { "text": "Resume", "value": "cmd:/resume abc", "style": "primary" },
        { "text": "End", "value": "cmd:/cancel abc", "style": "danger" }
      ]]}
    ]
  } }

{ "type": "send_buttons", "session_key": "...", "text": "...", "buttons": [...] }
{ "type": "update_message", "session_key": "...", "preview_handle": "<id>", "text": "..." }
{ "type": "send_image", "session_key": "...", "image": { "path": "...", "url": "..." } }
{ "type": "send_file", "session_key": "...", "file": { "path": "...", "filename": "..." } }
{ "type": "start_typing", "session_key": "..." }
{ "type": "stop_typing", "session_key": "..." }
{ "type": "pong" }
```

The adapter only receives frame types corresponding to capabilities
it claimed on register — opendray gates by the
`accept_capabilities` whitelist + the adapter's declared list.

## 5. Heartbeat

opendray sends WebSocket-level Ping frames every ~54s. Most WS
libraries auto-reply Pong. The adapter can also send
application-level `{"type":"ping"}` and opendray replies with
`{"type":"pong"}`.

## 6. Reconnection

The bridge's broker registration in opendray persists across WS
disconnects — the adapter can reconnect at any time with the same
token. A fresh `register` frame replaces any prior connection.
While disconnected, opendray returns `ErrNotSupported` for
outbound calls (which the Hub treats as "fall back to text").

## Reference: minimal Python adapter

Already available in the **Adapter setup** dialog of every bridge
channel. The dialog substitutes your specific URL + token + name
+ capability list, so you can paste-and-run.

## Full protocol spec

`docs/bridge-protocol.md` in the repository has the complete frame
catalogue + edge cases. The adapter setup dialog is for getting
started; the spec is for production-quality adapters.
