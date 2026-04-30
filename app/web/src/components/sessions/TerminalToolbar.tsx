import { cn } from '@/lib/utils'

interface TerminalToolbarProps {
  onKey: (sequence: string) => void
}

interface KeyDef {
  label: string
  /** Bytes injected into the PTY's stdin. */
  seq: string
  /** Optional className override (e.g. wider for ENTER). */
  className?: string
  /** Optional title shown on hover. */
  title?: string
}

// ANSI escape sequences that the on-screen toolbar emits. ESC is a
// single byte (0x1b); arrows are CSI sequences; ⌃X are control codes
// in the C0 range.
const KEYS: KeyDef[] = [
  { label: 'ESC', seq: '\x1b', title: 'Escape' },
  { label: 'TAB', seq: '\t', title: 'Tab' },
  { label: '↑', seq: '\x1b[A', title: 'Arrow up' },
  { label: '↓', seq: '\x1b[B', title: 'Arrow down' },
  { label: '←', seq: '\x1b[D', title: 'Arrow left' },
  { label: '→', seq: '\x1b[C', title: 'Arrow right' },
  { label: '⌃C', seq: '\x03', title: 'Ctrl+C — interrupt' },
  { label: '⌃D', seq: '\x04', title: 'Ctrl+D — EOF / logout' },
  { label: '⌃L', seq: '\x0c', title: 'Ctrl+L — clear screen' },
  { label: '⌃R', seq: '\x12', title: 'Ctrl+R — reverse search' },
  { label: '⌃Z', seq: '\x1a', title: 'Ctrl+Z — suspend' },
  { label: '⌃A', seq: '\x01', title: 'Ctrl+A — line start' },
  { label: '⌃E', seq: '\x05', title: 'Ctrl+E — line end' },
  { label: '⌃U', seq: '\x15', title: 'Ctrl+U — clear line' },
  { label: '↵', seq: '\r', title: 'Enter', className: 'min-w-[44px]' },
]

export function TerminalToolbar({ onKey }: TerminalToolbarProps) {
  return (
    <div className="border-t border-border bg-card/40 shrink-0">
      <div className="flex gap-1 p-1.5 overflow-x-auto scrollbar-hide">
        {KEYS.map((k) => (
          <button
            key={k.label}
            type="button"
            onClick={() => onKey(k.seq)}
            title={k.title ?? k.label}
            className={cn(
              'shrink-0 h-8 px-2.5 rounded-md border border-border bg-background',
              'text-[12px] font-mono text-foreground',
              'transition-colors hover:bg-card active:bg-accent/15 active:border-accent/40',
              'min-w-[36px] flex items-center justify-center',
              k.className,
            )}
          >
            {k.label}
          </button>
        ))}
      </div>
    </div>
  )
}
