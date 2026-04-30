import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Plus, Loader2 } from 'lucide-react'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { listSessions, terminateSession } from '@/lib/sessions'
import type { Session } from '@/lib/types'
import { useSessionTabs } from '@/stores/sessionTabs'
import { SessionRow } from './SessionRow'

interface SessionListProps {
  onSpawn: () => void
  onOpen: (session: Session) => void
}

function order(a: Session, b: Session): number {
  // Live sessions first, then by started_at desc.
  const aLive = a.state !== 'ended' ? 0 : 1
  const bLive = b.state !== 'ended' ? 0 : 1
  if (aLive !== bLive) return aLive - bLive
  return new Date(b.started_at).getTime() - new Date(a.started_at).getTime()
}

export function SessionList({ onSpawn, onOpen }: SessionListProps) {
  const qc = useQueryClient()
  const { data: sessions, isLoading } = useQuery({
    queryKey: ['sessions'],
    queryFn: listSessions,
    refetchInterval: 4_000,
  })

  const currentId = useSessionTabs((s) => s.currentId)
  const closeTab = useSessionTabs((s) => s.close)
  const sorted = (sessions ?? []).slice().sort(order)
  const live = sorted.filter((s) => s.state !== 'ended')
  const ended = sorted.filter((s) => s.state === 'ended')

  const remove = useMutation({
    mutationFn: (s: Session) => terminateSession(s.id),
    onMutate: (s) => closeTab(s.id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['sessions'] })
    },
    onError: (err: Error) =>
      toast.error('Delete failed', { description: err.message }),
  })

  const handleDelete = (s: Session) => {
    if (s.state !== 'ended') {
      if (
        !confirm(`Terminate and remove ${s.name || s.provider_id}?`)
      ) {
        return
      }
    }
    remove.mutate(s)
  }

  return (
    <aside className="w-72 shrink-0 border-r border-border flex flex-col bg-background">
      <div className="h-9 px-3 flex items-center justify-between border-b border-border">
        <div className="flex items-center gap-2">
          <span className="text-[11px] font-semibold tracking-tight uppercase text-muted-foreground">
            Sessions
          </span>
          <span className="text-[10px] text-muted-foreground/60 font-mono">
            {live.length} live · {ended.length} ended
          </span>
        </div>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="ghost"
              size="icon"
              onClick={onSpawn}
              aria-label="Spawn new session"
              className="size-6"
            >
              <Plus className="size-3.5" />
            </Button>
          </TooltipTrigger>
          <TooltipContent>
            New session <kbd className="ml-1">⌘N</kbd>
          </TooltipContent>
        </Tooltip>
      </div>

      <ScrollArea className="flex-1">
        <div className="p-1.5 flex flex-col gap-0.5">
          {isLoading && (
            <div className="flex items-center gap-2 px-2 py-3 text-[12px] text-muted-foreground">
              <Loader2 className="size-3.5 animate-spin" />
              Loading…
            </div>
          )}
          {!isLoading && sorted.length === 0 && (
            <div className="px-3 py-6 text-center text-[12px] text-muted-foreground">
              No sessions yet.
              <br />
              Press <kbd>⌘N</kbd> to spawn.
            </div>
          )}
          {live.map((s) => (
            <SessionRow
              key={s.id}
              session={s}
              active={s.id === currentId}
              onClick={() => onOpen(s)}
              onDelete={() => handleDelete(s)}
            />
          ))}
          {ended.length > 0 && (
            <div className="px-2 py-1.5 mt-1 flex items-center justify-between gap-2">
              <span className="text-[10px] uppercase tracking-wider text-muted-foreground/60">
                Ended ({ended.length})
              </span>
              {ended.length > 1 && (
                <button
                  type="button"
                  onClick={() => {
                    if (!confirm(`Remove all ${ended.length} ended sessions?`))
                      return
                    ended.forEach((s) => remove.mutate(s))
                  }}
                  className="text-[10px] text-muted-foreground/70 hover:text-destructive transition-colors"
                >
                  Clear all
                </button>
              )}
            </div>
          )}
          {ended.map((s) => (
            <SessionRow
              key={s.id}
              session={s}
              active={s.id === currentId}
              onClick={() => onOpen(s)}
              onDelete={() => handleDelete(s)}
            />
          ))}
        </div>
      </ScrollArea>
    </aside>
  )
}
