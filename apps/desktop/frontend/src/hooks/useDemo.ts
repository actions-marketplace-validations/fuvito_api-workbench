import { useState } from 'react'
import { startDemo, stopDemo } from '@/lib/bridge'
import type { FileNode } from '@/lib/bridge'
import { createBlankTab } from '@/lib/tabState'
import type { TabState } from '@/lib/tabState'

type TabsStateTuple = { tabs: TabState[]; activeTabId: string }

export function useDemo(deps: {
  refreshWorkspace: () => Promise<FileNode[]>
  setTabsState: React.Dispatch<React.SetStateAction<TabsStateTuple>>
  openFirstFileInTree: (tree: FileNode[]) => Promise<void>
}) {
  const { refreshWorkspace, setTabsState, openFirstFileInTree } = deps

  const [demoRunning, setDemoRunning] = useState(false)
  const [demoPort, setDemoPort] = useState(0)
  const [demoStarting, setDemoStarting] = useState(false)

  async function handleStartDemo() {
    if (demoRunning || demoStarting) return
    setDemoStarting(true)
    try {
      const result = await startDemo()
      if (result.error) {
        console.error('demo start failed:', result.error)
        return
      }
      setDemoRunning(true)
      setDemoPort(result.port)
      const tree = await refreshWorkspace()
      await openFirstFileInTree(tree)
    } catch (err) {
      console.error('demo start error:', err)
    } finally {
      setDemoStarting(false)
    }
  }

  async function handleStopDemo() {
    try {
      const port = demoPort
      await stopDemo()
      setDemoRunning(false)
      setDemoPort(0)
      if (port > 0) {
        setTabsState(prev => {
          const prefix = `http://localhost:${port}`
          const remaining = prev.tabs.filter(t => !t.url.startsWith(prefix))
          if (remaining.length === prev.tabs.length) return prev
          if (remaining.length === 0) {
            const blank = createBlankTab()
            return { tabs: [blank], activeTabId: blank.id }
          }
          const activeStillOpen = remaining.some(t => t.id === prev.activeTabId)
          return {
            tabs: remaining,
            activeTabId: activeStillOpen ? prev.activeTabId : remaining[remaining.length - 1].id,
          }
        })
      }
      await refreshWorkspace()
    } catch (err) {
      console.error('demo stop error:', err)
    }
  }

  function clearDemoState() {
    setDemoRunning(false)
    setDemoPort(0)
  }

  return {
    demoRunning, demoPort, demoStarting,
    handleStartDemo, handleStopDemo, clearDemoState,
  }
}
