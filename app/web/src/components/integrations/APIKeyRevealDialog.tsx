import { useState } from 'react'
import { Copy, AlertTriangle, Check } from 'lucide-react'

import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'

interface APIKeyRevealDialogProps {
  open: boolean
  apiKey: string
  title?: string
  description?: string
  onClose: () => void
}

/**
 * Strict one-time API key reveal. Caller cannot dismiss until they
 * confirm they've stored the key (checkbox). Copy button switches to
 * a checkmark for 1.5s on success.
 */
export function APIKeyRevealDialog({
  open,
  apiKey,
  title = 'API key issued',
  description = 'This is the only time the plaintext key will be shown. Copy it now and store it in a secret manager.',
  onClose,
}: APIKeyRevealDialogProps) {
  const [acknowledged, setAcknowledged] = useState(false)
  const [copied, setCopied] = useState(false)

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(apiKey)
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    } catch {
      /* clipboard blocked — user can still select+copy from the input */
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(v) => {
        if (!v && acknowledged) {
          setAcknowledged(false)
          setCopied(false)
          onClose()
        }
      }}
    >
      <DialogContent
        className="max-w-[520px]"
        hideClose
        onEscapeKeyDown={(e) => e.preventDefault()}
        onInteractOutside={(e) => e.preventDefault()}
      >
        <DialogHeader>
          <div className="flex items-center gap-2">
            <AlertTriangle className="size-4 text-state-idle" />
            <DialogTitle>{title}</DialogTitle>
          </div>
          <DialogDescription>{description}</DialogDescription>
        </DialogHeader>

        <div className="flex items-center gap-2 mt-3">
          <input
            readOnly
            value={apiKey}
            onFocus={(e) => e.currentTarget.select()}
            className="flex-1 font-mono text-[12px] bg-input/40 border border-border rounded-md h-9 px-3"
          />
          <Button variant="outline" size="sm" onClick={copy}>
            {copied ? (
              <>
                <Check className="size-3.5" />
                Copied
              </>
            ) : (
              <>
                <Copy className="size-3.5" />
                Copy
              </>
            )}
          </Button>
        </div>

        <label className="flex items-start gap-2 mt-3 text-[12px] text-muted-foreground cursor-pointer">
          <input
            type="checkbox"
            checked={acknowledged}
            onChange={(e) => setAcknowledged(e.target.checked)}
            className="mt-0.5 accent-accent"
          />
          <span>
            I have copied the key. I understand opendray will not display it
            again.
          </span>
        </label>

        <DialogFooter>
          <Button
            variant="accent"
            size="sm"
            disabled={!acknowledged}
            onClick={() => {
              setAcknowledged(false)
              setCopied(false)
              onClose()
            }}
          >
            Done
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
