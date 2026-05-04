# WeChat (个人微信)

**Mode:** push via WxPusher (no public URL)
**Capabilities:** text · card (markdown)
**Setup time:** ~5 minutes

Personal WeChat has no official open API for third-party apps.
opendray uses **WxPusher**, a free push-relay service: each
recipient subscribes once via QR code in WeChat, then opendray
pushes notifications by App Token.

> **Outbound only.** Push services don't relay user replies. For
> bidirectional personal WeChat use a bridge channel with a
> WeChaty / iPad-protocol adapter (account-ban risk applies — you
> assume the risk).

## 1. Create a WxPusher application

1. Visit [wxpusher.zjiecode.com/admin](https://wxpusher.zjiecode.com/admin)
   → log in (WeChat scan).
2. **应用管理 → 创建应用** → fill name + description.
3. After creation, copy the **APP_TOKEN** (starts with `AT_`).

![WxPusher app creation](/tutorial/wechat-wxpusher-app.png)

## 2. Subscribe recipients

You can address recipients two ways — pick one or use both:

### Option A — by UID (specific people)

1. **用户管理** → **复制关注二维码 (Copy follow QR)**.
2. Each recipient scans this QR in WeChat → follows your app's
   official-account-style endpoint → returns to your **用户管理**
   list with a `UID_…` token.
3. Copy each UID and paste them, one per line, into opendray's
   form.

### Option B — by topic (broadcast channel)

1. **主题管理** → **新建主题** → returns a numeric **topicId**.
2. Share the topic's QR code; each subscriber scans → joins.
3. Paste the topicId in opendray. Everyone subscribed receives
   every push.

You can mix both — opendray pushes to the union of UIDs + topic
subscribers.

## 3. Configure in opendray

Channels → **New channel** → kind **WeChat (个人微信)**.

| Field | Value |
|---|---|
| **App token (AT_…)** | from step 1 |
| **Recipient UIDs** | one per line (optional if topic IDs given) |
| **Topic IDs** | one per line, numeric (optional if UIDs given) |
| **Tap-through URL** | optional — when set, tapping the WeChat notification opens this URL |

At least one of UIDs / Topic IDs is required.

Save with **Enabled = on**.

## 4. Verify

- Hit **Test** → a one-line text notification lands on each
  recipient's WeChat.
- Trigger a session.idle → markdown push: bold header + body.
  Buttons whose value is a clickable URL appear as inline links;
  `cmd:` buttons are dropped silently.

## Notification banner

WxPusher's banner preview is capped at ~20 characters. opendray
uses the card header (or the first 20 chars of the body when no
header) for the preview, then the full markdown content below.

## Card rendering

| Card element | Output |
|---|---|
| `CardHeader.Title` | `## Title` (renders as small bold heading in WxPusher's web view) |
| `CardMarkdown` | passthrough CommonMark |
| `CardDivider` | `---` |
| `CardActions` URL buttons | bottom link row `· [Open](url) ·` |
| `CardActions` `cmd:` buttons | dropped |
| `CardListItem` | `- text  [btn](url)` |
| `CardNote` | `> note` blockquote |

## Limitations

- **Outbound only.** Replies are physically impossible through a
  push relay. For two-way personal-WeChat use a bridge + WeChaty.
- **WxPusher rate limits:** ~5 messages/sec per app token, 40 KB
  message size. Plenty for opendray's notification volume.
- **App token leakage** lets anyone push to all your recipients.
  Rotate via the WxPusher console if exposed.
- **Offline recipients** receive the push when their phone next
  connects (WeChat caches official-account messages).
