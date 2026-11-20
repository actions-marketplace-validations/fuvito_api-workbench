import { useState } from 'react'
import { startDemo, stopDemo } from '@/lib/bridge'
import type { FileNode } from '@/lib/bridge'
import { withoutTabsIn } from '@/lib/tabState'
import type { TabState } from '@/lib/tabState'

type TabsStateTuple = { tabs: TabState[]; activeTabId: string }

export function useDemo(deps: {
  refreshWorkspace: () => Promise<FileNode[]>
  refreshHistory: () => Promise<void>
  setTabsState: React.Dispatch<React.SetStateAction<TabsStateTuple>>
  openFirstFileInTree: (tree: FileNode[]) => Promise<void>
  onWorkspaceOpened?: () => void
  onError?: (message: string) => void
}) {
  const { refreshWorkspace, refreshHistory, setTabsState, openFirstFileInTree, onWorkspaceOpened, onError } = deps

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
        onError?.(result.error)
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
      setTabsState(prev => withoutTabsIn(prev, port > 0 ? `http://localhost:${port}` : null, wsPath))
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
