import { useState } from 'react'
import { startDemo, stopDemo } from '@/lib/bridge'
import type { FileNode } from '@/lib/bridge'
import { createBlankTab } from '@/lib/tabState'
import type { TabState } from '@/lib/tabState'

type TabsStateTuple = { tabs: TabState[]; activeTabId: string }

export function useDemo(deps: {
  refreshWorkspace: () => Promise<FileNode[]>
  refreshHistory: () => Promise<void>
  setTabsState: React.Dispatch<React.SetStateAction<TabsStateTuple>>
  openFirstFileInTree: (tree: FileNode[]) => Promise<void>
  onWorkspaceOpened?: () => void
}) {
  const { refreshWorkspace, refreshHistory, setTabsState, openFirstFileInTree, onWorkspaceOpened } = deps

  const [demoRunning, setDemoRunning] = useState(false)
  const [demoPort, setDemoPort] = useState(0)
  const [demoWorkspacePath, setDemoWorkspacePath] = useState('')
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
      setDemoWorkspacePath(result.workspacePath ?? '')
      const tree = await refreshWorkspace()
      await Promise.all([openFirstFileInTree(tree), refreshHistory()])
      onWorkspaceOpened?.()
    } catch (err) {
      console.error('demo start error:', err)
    } finally {
      setDemoStarting(false)
    }
  }

  async function handleStopDemo() {
    try {
      const port = demoPort
      const wsPath = demoWorkspacePath
      await stopDemo()
      setDemoRunning(false)
      setDemoPort(0)
      setDemoWorkspacePath('')
      setTabsState(prev => {
        const urlPrefix = port > 0 ? `http://localhost:${port}` : null
        const normWs = wsPath ? wsPath.replace(/\\/g, '/') : null
        const isDemoTab = (t: TabState): boolean => {
          if (urlPrefix && t.url.startsWith(urlPrefix)) return true
          const fp = (t.filePath ?? '').replace(/\\/g, '/')
          if (normWs && fp.startsWith(normWs)) return true
          const sp = (t.scenarioPath ?? '').replace(/\\/g, '/')
          if (normWs && sp.startsWith(normWs)) return true
          return false
        }
        const remaining = prev.tabs.filter(t => !isDemoTab(t))
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
      await refreshWorkspace()
    } catch (err) {
      console.error('demo stop error:', err)
    }
  }

  function clearDemoState() {
    setDemoRunning(false)
    setDemoPort(0)
    setDemoWorkspacePath('')
  }

  return {
    demoRunning, demoPort, demoWorkspacePath, demoStarting,
    handleStartDemo, handleStopDemo, clearDemoState,
  }
}
