import { create } from 'zustand'
import { persist } from 'zustand/middleware'

interface LayoutState {
  /** Global left navigation collapsed to icon-only mode. */
  sidebarCollapsed: boolean
  /** Sessions inner list panel hidden so the workbench takes full width. */
  sessionListCollapsed: boolean
  /** Soft-keyboard / shortcut bar under the terminal. */
  terminalToolbarOpen: boolean

  toggleSidebar: () => void
  toggleSessionList: () => void
  toggleTerminalToolbar: () => void
  setTerminalToolbarOpen: (v: boolean) => void
}

export const useLayout = create<LayoutState>()(
  persist(
    (set, get) => ({
      sidebarCollapsed: false,
      sessionListCollapsed: false,
      terminalToolbarOpen: false,

      toggleSidebar: () =>
        set({ sidebarCollapsed: !get().sidebarCollapsed }),
      toggleSessionList: () =>
        set({ sessionListCollapsed: !get().sessionListCollapsed }),
      toggleTerminalToolbar: () =>
        set({ terminalToolbarOpen: !get().terminalToolbarOpen }),
      setTerminalToolbarOpen: (v) => set({ terminalToolbarOpen: v }),
    }),
    { name: 'opendray.layout' },
  ),
)
