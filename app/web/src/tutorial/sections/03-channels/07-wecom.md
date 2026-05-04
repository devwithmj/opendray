# WeCom / Enterprise WeChat (企业微信)

**Mode:** group robot webhook (outbound only)
**Capabilities:** text · card (markdown) — no callback buttons
**Setup time:** ~2 minutes

The simplest WeCom integration. Like DingTalk's group robot it's
outbound-only — bidirectional WeCom needs the app-platform path
which isn't shipped yet.

## 1. Add a group robot

1. Open the target WeCom group on **desktop** (the mobile clients
   hide the group-bot UI in some versions).
2. Tap the group name to enter **Group settings**.
3. Scroll to **Group bots (群机器人)** → **Add Robot**.
4. Choose **Group robot (Webhook)**.
5. Name it (e.g. `OpenDray`).
6. Confirm. WeCom reveals the **Webhook URL**:
   ```
   https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=abc-123-...
   ```

![WeCom group robot URL](/tutorial/wecom-robot-url.png)

The `key=` query parameter is what opendray needs — but you can
also paste the whole URL.

## 2. Configure in opendray

Channels → **New channel** → kind **WeCom (企业微信)**.

| Field | Value |
|---|---|
| **Webhook key** | the `key=` value alone (`abc-123-…`) — preferred |
| **Or full webhook URL** | paste the entire URL if you prefer |

If both are set, the full URL wins. Save with **Enabled = on**.

## 3. Verify

- Hit **Test** → text message arrives.
- Trigger a session.idle → markdown message with bold title +
  body. URL buttons render as inline link rows; `cmd:` buttons
  are dropped.

## Card rendering

WeCom group robots support a small set of message types. opendray
uses **markdown**:

| Card element | Output |
|---|---|
| `CardHeader.Title` | `**Title**` (bold) |
| `CardMarkdown` | passthrough markdown |
| `CardDivider` | `---` |
| `CardActions` URL buttons | `[label](url)` link row at bottom |
| `CardActions` `cmd:` buttons | dropped |
| `CardNote` | `> note` blockquote |

WeCom markdown subset:
- `**bold**`, `_italic_`
- `[label](url)`
- `<font color="info|warning|comment">…</font>`
- Inline code `` `code` ``
- **No tables**, **no fenced code blocks**, **no headers (`#`)**

## Limitations

- **Outbound only.** Bidirectional WeCom requires the app-platform
  path with corp_id + agent_id + secret + AES-encrypted callback —
  not yet shipped.
- **Rate limit:** 20 messages/min per robot. Same as DingTalk.
- **Webhook URL is a bearer credential.** Anyone with the URL can
  post to the group. **Don't commit it to source control.**
- **Markdown rendering is limited.** Tables and code blocks look
  bad — the `formatForTelegram` HTML conversion doesn't apply
  here, so longer Claude responses with tables won't render as
  cleanly.
