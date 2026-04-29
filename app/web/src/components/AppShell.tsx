import { useState } from 'react'
import { Outlet } from '@tanstack/react-router'

import { SidebarNav } from './SidebarNav'
import { Topbar } from './Topbar'
import { CommandPalette, useCommandPaletteHotkey } from './CommandPalette'
import { TooltipProvider } from './ui/tooltip'

export function AppShell() {
  const [paletteOpen, setPaletteOpen] = useState(false)
  useCommandPaletteHotkey(setPaletteOpen)

  return (
    <TooltipProvider delayDuration={200}>
      <div className="h-svh flex flex-col bg-background text-foreground">
        <Topbar onOpenPalette={() => setPaletteOpen(true)} />
        <div className="flex-1 flex overflow-hidden min-h-0">
          <SidebarNav />
          <main className="flex-1 overflow-auto min-w-0">
            <Outlet />
          </main>
        </div>
      </div>
      <CommandPalette open={paletteOpen} onOpenChange={setPaletteOpen} />
    </TooltipProvider>
  )
}
