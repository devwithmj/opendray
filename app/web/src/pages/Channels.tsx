import { useState, type FormEvent, useEffect } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  Plus,
  Loader2,
  Send,
  Trash2,
  MessageSquare,
} from 'lucide-react'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import {
  Select,
  SelectTrigger,
  SelectContent,
  SelectItem,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Textarea } from '@/components/ui/textarea'
import {
  listChannels,
  createChannel,
  deleteChannel,
  testChannel,
  listChannelKinds,
  updateChannel,
} from '@/lib/channels'
import type { Channel } from '@/lib/types'
import { cn } from '@/lib/utils'

const NOTIFY_TOPICS = ['session.started', 'session.idle', 'session.ended']

export function ChannelsPage() {
  const qc = useQueryClient()
  const [createOpen, setCreateOpen] = useState(false)

  const { data: channels, isLoading } = useQuery({
    queryKey: ['channels'],
    queryFn: listChannels,
    refetchInterval: 6_000,
  })

  return (
    <div className="h-full flex flex-col bg-background">
      <header className="border-b border-border px-6 py-4 flex items-center gap-3">
        <div className="flex-1">
          <h1 className="text-[16px] font-semibold tracking-tight">Channels</h1>
          <p className="text-[12px] text-muted-foreground">
            Bidirectional messaging integrations. Outbound notifications are
            filtered by each channel's <code>notify_on</code>.
          </p>
        </div>
        <Button
          variant="accent"
          size="sm"
          onClick={() => setCreateOpen(true)}
        >
          <Plus className="size-3.5" /> New channel
        </Button>
      </header>

      <ScrollArea className="flex-1">
        <div className="p-6 max-w-[960px] flex flex-col gap-3">
          {isLoading && (
            <div className="flex items-center gap-2 text-[12px] text-muted-foreground">
              <Loader2 className="size-3.5 animate-spin" />
              Loading…
            </div>
          )}
          {!isLoading && (channels?.length ?? 0) === 0 && (
            <div className="flex flex-col items-center justify-center py-16 gap-3 text-center">
              <MessageSquare
                className="size-10 text-muted-foreground/40"
                strokeWidth={1.5}
              />
              <h2 className="text-[14px] font-semibold">No channels yet</h2>
              <p className="text-[12px] text-muted-foreground max-w-[360px]">
                Telegram is the bundled kind. Register a bot with @BotFather and
                paste the token here.
              </p>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setCreateOpen(true)}
              >
                <Plus className="size-3.5" /> Register telegram bot
              </Button>
            </div>
          )}
          {(channels ?? []).map((c) => (
            <ChannelCard
              key={c.id}
              channel={c}
              onTest={() => {
                testChannel(c.id)
                  .then(() => toast.success('Test message sent'))
                  .catch((err: Error) =>
                    toast.error('Test failed', { description: err.message }),
                  )
              }}
              onToggle={(enabled) => {
                updateChannel(c.id, { enabled })
                  .then(() => qc.invalidateQueries({ queryKey: ['channels'] }))
                  .catch((err: Error) => toast.error(err.message))
              }}
              onDelete={() => {
                if (!confirm(`Delete channel ${c.id}?`)) return
                deleteChannel(c.id)
                  .then(() => {
                    qc.invalidateQueries({ queryKey: ['channels'] })
                    toast.success('Channel deleted')
                  })
                  .catch((err: Error) => toast.error(err.message))
              }}
            />
          ))}
        </div>
      </ScrollArea>

      <CreateChannelDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
      />
    </div>
  )
}

function ChannelCard({
  channel,
  onTest,
  onToggle,
  onDelete,
}: {
  channel: Channel
  onTest: () => void
  onToggle: (enabled: boolean) => void
  onDelete: () => void
}) {
  const cfg = channel.config as {
    bot_token?: string
    chat_id?: number | string
    notify_on?: string[]
  }
  const tokenPreview = cfg.bot_token
    ? `${cfg.bot_token.slice(0, 6)}…${cfg.bot_token.slice(-4)}`
    : '—'
  return (
    <div className="border border-border rounded-md p-4 bg-card/30 flex flex-col gap-3">
      <div className="flex items-start gap-3">
        <span className="text-lg leading-none mt-0.5">📨</span>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 flex-wrap">
            <span className="text-[13px] font-semibold">
              {channel.kind}
            </span>
            <Badge variant="outline" className="font-mono normal-case">
              {channel.id}
            </Badge>
            {channel.running ? (
              <Badge variant="success">running</Badge>
            ) : channel.enabled ? (
              <Badge variant="warning">starting…</Badge>
            ) : (
              <Badge variant="muted">disabled</Badge>
            )}
          </div>
          <div className="text-[11px] text-muted-foreground/80 font-mono mt-1 flex flex-wrap gap-x-4 gap-y-0.5">
            <span>token: {tokenPreview}</span>
            {cfg.chat_id !== undefined && (
              <span>chat_id: {String(cfg.chat_id)}</span>
            )}
            {cfg.notify_on && cfg.notify_on.length > 0 && (
              <span>notify_on: {cfg.notify_on.join(', ')}</span>
            )}
          </div>
        </div>
        <div className="flex items-center gap-2 shrink-0">
          <Switch checked={channel.enabled} onCheckedChange={onToggle} />
          <Button
            variant="outline"
            size="sm"
            onClick={onTest}
            disabled={!channel.running}
            title={!channel.running ? 'Channel must be running' : undefined}
          >
            <Send className="size-3.5" /> Test
          </Button>
          <Button
            variant="ghost"
            size="icon"
            onClick={onDelete}
            aria-label="Delete channel"
            className="text-muted-foreground hover:text-destructive"
          >
            <Trash2 className="size-3.5" />
          </Button>
        </div>
      </div>
    </div>
  )
}

function CreateChannelDialog({
  open,
  onOpenChange,
}: {
  open: boolean
  onOpenChange: (v: boolean) => void
}) {
  const qc = useQueryClient()
  const { data: kinds } = useQuery({
    queryKey: ['channel-kinds'],
    queryFn: listChannelKinds,
    enabled: open,
  })

  const [kind, setKind] = useState<string>('telegram')
  const [botToken, setBotToken] = useState('')
  const [chatId, setChatId] = useState('')
  const [notifyText, setNotifyText] = useState(
    NOTIFY_TOPICS.join('\n'),
  )
  const [enabled, setEnabled] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (open) {
      setError(null)
    }
  }, [open])

  useEffect(() => {
    if (kinds && kinds.length > 0 && !kinds.includes(kind)) {
      setKind(kinds[0])
    }
  }, [kinds, kind])

  const create = useMutation({
    mutationFn: createChannel,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['channels'] })
      toast.success('Channel created')
      onOpenChange(false)
      setBotToken('')
      setChatId('')
      setNotifyText(NOTIFY_TOPICS.join('\n'))
    },
    onError: (e: Error) => setError(e.message),
  })

  const submit = (e: FormEvent) => {
    e.preventDefault()
    setError(null)
    if (!botToken.trim()) {
      setError('bot_token is required')
      return
    }
    const config: Record<string, unknown> = { bot_token: botToken.trim() }
    const chat = chatId.trim()
    if (chat) {
      const n = Number(chat)
      config.chat_id = Number.isFinite(n) ? n : chat
    }
    const notifyOn = notifyText
      .split('\n')
      .map((s) => s.trim())
      .filter((s) => s.length > 0)
    if (notifyOn.length > 0) config.notify_on = notifyOn

    create.mutate({ kind, config, enabled })
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-[480px]">
        <DialogHeader>
          <DialogTitle>Register channel</DialogTitle>
          <DialogDescription>
            Bot token and default chat for outbound notifications.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={submit} className="flex flex-col gap-4 mt-2">
          <div className="space-y-1.5">
            <Label htmlFor="kind">Kind</Label>
            <Select value={kind} onValueChange={setKind}>
              <SelectTrigger id="kind">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {(kinds ?? ['telegram']).map((k) => (
                  <SelectItem key={k} value={k}>
                    {k}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="bot_token">Bot token</Label>
            <Input
              id="bot_token"
              type="password"
              autoComplete="off"
              value={botToken}
              onChange={(e) => setBotToken(e.target.value)}
              placeholder="123456:ABC-DEF..."
              required
              autoFocus
              className="font-mono"
            />
            <p className="text-[11px] text-muted-foreground/80">
              From @BotFather. Stored as-is in channel config; protected by
              admin auth on REST.
            </p>
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="chat_id">Default chat ID</Label>
            <Input
              id="chat_id"
              value={chatId}
              onChange={(e) => setChatId(e.target.value)}
              placeholder="42"
              className="font-mono"
            />
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="notify_on">Notify on (one topic per line)</Label>
            <Textarea
              id="notify_on"
              rows={3}
              value={notifyText}
              onChange={(e) => setNotifyText(e.target.value)}
              className="font-mono text-[12px]"
            />
          </div>

          <div className="flex items-center gap-2">
            <Switch
              id="enabled"
              checked={enabled}
              onCheckedChange={setEnabled}
            />
            <Label htmlFor="enabled" className="!text-[12px]">
              Enabled (start polling immediately)
            </Label>
          </div>

          {error && (
            <div className={cn('text-[12px] text-destructive bg-destructive/10 border border-destructive/30 rounded-md px-3 py-2')}>
              {error}
            </div>
          )}

          <DialogFooter>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={() => onOpenChange(false)}
              disabled={create.isPending}
            >
              Cancel
            </Button>
            <Button
              type="submit"
              variant="accent"
              size="sm"
              disabled={create.isPending}
            >
              {create.isPending && <Loader2 className="size-3.5 animate-spin" />}
              {create.isPending ? 'Creating…' : 'Create'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
