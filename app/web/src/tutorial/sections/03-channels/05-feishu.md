# Feishu (飞书 / Lark)

**Mode:** event webhook (requires public URL) + REST outbound
**Capabilities:** text · card (interactive v2) · buttons · reply_to_message
**Setup time:** ~15 minutes (the dev console is content-heavy)

Feishu / Lark pushes events to a webhook URL on opendray, and
opendray replies via the standard `/open-apis/im/v1/messages` API
authenticated by a refreshable `tenant_access_token`.

> **Public URL prerequisite:** opendray's webhook endpoint must be
> reachable from feishu.cn (or larksuite.com for international
> Lark). Use Cloudflare Tunnel, ngrok, or a real public hostname.

## 1. Create the app

1. Visit [open.feishu.cn/app](https://open.feishu.cn/app) (or
   [open.larksuite.com/app](https://open.larksuite.com/app) for
   Lark).
2. **Create custom app** → enter name + icon.
3. Sidebar → **Credentials & Basic Info**. Copy:
   - **App ID** — looks like `cli_a1b2c3d4...`
   - **App Secret**

![Feishu app credentials](/tutorial/feishu-credentials.png)

## 2. Add bot capability

Sidebar → **Add features** → **Bot** → **Add**.

This is required — without the Bot feature flag, sending messages
returns "app does not have bot ability".

## 3. Configure permissions

Sidebar → **Permissions & Scopes** → search and enable:

| Permission | Why |
|---|---|
| `im:message` | Read messages |
| `im:message:send_as_bot` | Send as bot |
| `im:chat` | Read chat metadata |
| `im:chat:readonly` | List chats the bot is in |
| `im:resource` | Read images/files in messages (optional) |

Click **Apply for permissions**. For self-built apps in your own
tenant the approval is instant.

## 4. Create the channel in opendray (FIRST — you need its webhook URL)

Even though events haven't been wired yet, create the channel in
opendray now so it has an id:

Channels → **New channel** → kind **Feishu (飞书)**.

| Field | Value |
|---|---|
| **App ID** | from step 1 |
| **App Secret** | from step 1 |
| **Verification token** | leave blank for now (we'll fill in later) |
| **Default chat ID** | leave blank for now |

Save with **Enabled = on**.

The card now shows `webhook:` with a URL like
`https://your-host/api/v1/channels/ch_abc.../webhook`. Copy this
URL — Feishu will call it.

![Feishu channel card with webhook URL](/tutorial/feishu-channel-webhook.png)

## 5. Wire the webhook in Feishu

Back in Feishu's dev console:

Sidebar → **Event Subscriptions** → toggle on.

- **Request URL:** paste the URL you copied from opendray.
- Feishu **immediately** calls that URL with a JSON challenge
  (`{"type":"url_verification","challenge":"..."}`). opendray echoes
  the challenge back automatically — you should see ✅ within a
  second or two.

If verification fails:
- Check the URL is reachable from the public internet
  (`curl -X POST <url> -H 'content-type: application/json' -d '{"type":"url_verification","challenge":"x","token":""}'`
  should return `{"challenge":"x"}`)
- Check opendray's server log for the request
- For local dev: ensure your tunnel (cloudflared/ngrok) actually
  forwards POST requests with bodies

After the URL is verified, copy the **Verification Token** shown
on the same page.

Under **Subscribe to events**, **Add events**:

- `im.message.receive_v1` — bot receives a message

Save.

## 6. Re-edit the opendray channel

Edit the channel and fill in:

- **Verification token** — from step 5 (helps reject forged
  webhook calls)
- **Default chat ID** — see step 7 below

## 7. Find the chat ID

Two paths:

**Path A — bot in a chat already:** chat → ⋯ → settings → bot details. Some
Feishu admin panels expose `oc_…` ids directly.

**Path B — list via API:**

```bash
curl -X POST 'https://open.feishu.cn/open-apis/im/v1/chats' \
  -H "Authorization: Bearer <tenant_access_token>"
```

Get the token via `POST /open-apis/auth/v3/tenant_access_token/internal`
with your app id + secret. The chat id you want looks like
`oc_xxxxxxxxxxxxxxxxxxxx`.

## 8. Add the bot to a chat

Feishu chat → ⋯ → **Settings** → **Bots** → **Add bot** → pick yours.

## 9. Verify

Send a message in the chat — opendray's server log should show
the inbound. Send `/help` — opendray replies with the command list.

## Card schema

opendray emits Card v2 (`schema: "2.0"`) JSON. The `header.template`
field maps colour names to Feishu's palette:

| opendray colour | Feishu template |
|---|---|
| blue | blue |
| green | green |
| red | red |
| orange | orange |
| yellow | yellow |
| violet / indigo | purple |
| turquoise | turquoise |

## Limitations

- **AES encryption of webhook payloads is NOT supported** in v1.
  Leave the *Encrypt Key* setting blank in Feishu — events arrive
  in plain JSON. Verification token alone is sufficient for most
  setups.
- Card v1 (the older schema) is not used; everything is v2.
- The webhook URL must be HTTPS for production. Cloudflare Tunnel
  works (the tunnel terminates TLS on Cloudflare's edge).
