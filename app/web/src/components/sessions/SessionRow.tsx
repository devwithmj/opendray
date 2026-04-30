import { formatDistanceToNow } from 'date-fns'
import { X } from 'lucide-react'

import { cn } from '@/lib/utils'
import type { Session } from '@/lib/types'
import { StatePill } from './StatePill'

interface SessionRowProps {
  session: Session
  active?: boolean
  onClick?: () => void
  onDelete?: () => void
}

function displayName(s: Session): string {
  if (s.name && s.name.length > 0) return s.name
  const parts = s.cwd.split('/').filter(Boolean)
  return parts.length ? parts[parts.length - 1] : s.cwd
}

function relativeStarted(s: Session): string {
  const t = s.ended_at ?? s.started_at
  try {
    return formatDistanceToNow(new Date(t), { addSuffix: true })
  } catch {
    return '—'
  }
}

export function SessionRow({
  session,
  active,
  onClick,
  onDelete,
}: SessionRowProps) {
  return (
    <div
      role="button"
      tabIndex={0}
      onClick={onClick}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault()
          onClick?.()
        }
      }}
      className={cn(
        'group relative w-full flex flex-col gap-1 px-2.5 py-2 rounded-md text-left transition-colors cursor-pointer',
        'border border-transparent',
        active
          ? 'bg-card border-border'
          : 'hover:bg-card/60 hover:border-border/60',
      )}
    >
      <div className="flex items-center gap-2 min-w-0 pr-5">
        <span className="text-[12px] font-medium truncate flex-1 text-foreground">
          {displayName(session)}
        </span>
        <StatePill state={session.state} exitCode={session.exit_code} />
      </div>
      <div className="flex items-center gap-2 min-w-0 pr-5">
        <span className="text-[11px] text-muted-foreground/70 truncate flex-1 font-mono">
          {session.provider_id} · {session.cwd}
        </span>
        <span className="text-[10px] text-muted-foreground/60 shrink-0">
          {relativeStarted(session)}
        </span>
      </div>
      {onDelete && (
        <button
          type="button"
          onClick={(e) => {
            e.stopPropagation()
            onDelete()
          }}
          aria-label="Delete session"
          title={
            session.state === 'ended'
              ? 'Remove from history'
              : 'Terminate and remove'
          }
          className={cn(
            'absolute top-1.5 right-1.5 size-5 rounded-sm flex items-center justify-center',
            'text-muted-foreground/50 hover:text-destructive hover:bg-destructive/10',
            'opacity-0 group-hover:opacity-100 focus-visible:opacity-100 transition-opacity',
          )}
        >
          <X className="size-3" />
        </button>
      )}
    </div>
  )
}
