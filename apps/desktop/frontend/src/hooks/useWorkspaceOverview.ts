import { useCallback, useEffect, useState } from 'react'
import { getWorkspaceOverview } from '@/lib/bridge'
import type { WorkspaceOverview } from '@/lib/bridge'

type Options = {
  workspacePath: string
  // True while something that shows the overview is on screen (the Workspace tab or the Workspace
  // sidebar). It reads again whenever that becomes true, so a problem fixed elsewhere is gone when
  // the user comes back.
  enabled: boolean
  // Bumped by the app after something that changes what is shown (a run, an environment change).
  refreshTrigger?: number
}

// One snapshot of the workspace shared by the Workspace page and the Workspace sidebar, so the two
// never disagree.
export function useWorkspaceOverview({ workspacePath, enabled, refreshTrigger = 0 }: Options) {
  const [overview, setOverview] = useState<WorkspaceOverview | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  const reload = useCallback(async () => {
    setLoading(true)
    try {
      setOverview(await getWorkspaceOverview())
      setError(null)
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (enabled && workspacePath) void reload()
  }, [reload, enabled, workspacePath, refreshTrigger])

  return { overview, error, loading, reload }
}
