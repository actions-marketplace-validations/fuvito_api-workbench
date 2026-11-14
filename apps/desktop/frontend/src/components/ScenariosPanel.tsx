import { useState } from 'react'
import { PlayCircle } from 'lucide-react'
import type { ScenarioInfo, TemplateInfo } from '@/lib/bridge'
import { listTemplates, createFromTemplate } from '@/lib/bridge'
import type { RunAllState } from '@/components/Sidebar'
import { WorkspaceScenarioSection } from './WorkspaceScenarioSection'
import { RunAllResults } from './WorkspaceRunResults'
import { ContextMenu } from './ContextMenu'
import { ScenarioTemplateGallery } from './ScenarioTemplateGallery'
import { buildScenarioMenuItems } from './workspaceMenus'

type ScenariosPanelProps = {
  workspacePath: string
  scenarios: ScenarioInfo[]
  scenarioError?: string | null
  runAllState?: RunAllState
  onRunAllScenarios?: () => void
  onDismissRunAll?: () => void
  onSelectScenario?: (path: string, name: string) => void
  onNewScenario?: (name: string) => void
  onNewScenarioFile?: (path: string, name: string) => void
  onDuplicateScenario?: (path: string) => void
  onDeleteScenario?: (path: string) => void
  onRenameScenario?: (path: string, newName: string) => void
}

export function ScenariosPanel({
  workspacePath, scenarios, scenarioError,
  runAllState, onRunAllScenarios, onDismissRunAll,
  onSelectScenario, onNewScenario, onNewScenarioFile,
  onDuplicateScenario, onDeleteScenario, onRenameScenario,
}: ScenariosPanelProps) {
  const [renamingScenarioPath, setRenamingScenarioPath] = useState<string | null>(null)
  const [showTemplates, setShowTemplates] = useState(false)
  const [templates, setTemplates] = useState<TemplateInfo[]>([])
  const [templatesLoading, setTemplatesLoading] = useState(false)
  const [scenarioCtxMenu, setScenarioCtxMenu] = useState<{
    x: number; y: number; path: string; name: string
  } | null>(null)

  async function openTemplateGallery() {
    setShowTemplates(true)
    if (templates.length === 0) {
      setTemplatesLoading(true)
      const list = await listTemplates()
      setTemplates(list)
      setTemplatesLoading(false)
    }
  }

  async function handleUseTemplate(templateID: string, name: string) {
    setShowTemplates(false)
    const result = await createFromTemplate(templateID, name)
    if (!result.error && result.path) {
      onNewScenarioFile?.(result.path, name)
    }
  }

  if (!workspacePath) {
    return (
      <div className="flex flex-col items-center justify-center flex-1 gap-3 p-6 text-center">
        <PlayCircle size={24} className="text-[var(--color-muted-foreground)]" />
        <p className="text-xs text-[var(--color-muted-foreground)]">
          Open a workspace to manage scenarios.
        </p>
      </div>
    )
  }

  return (
    <div className="flex flex-col h-full overflow-hidden">
      <div className="flex-1 overflow-auto py-1">
        <WorkspaceScenarioSection
          scenarios={scenarios}
          scenarioError={scenarioError}
          runAllState={runAllState}
          onRunAllScenarios={onRunAllScenarios}
          onNewScenario={onNewScenario}
          onSelectScenario={onSelectScenario}
          onDuplicateScenario={onDuplicateScenario}
          onDeleteScenario={onDeleteScenario}
          onRenameScenario={onRenameScenario}
          onOpenTemplateGallery={openTemplateGallery}
          renamingScenarioPath={renamingScenarioPath}
          onRenamingChange={setRenamingScenarioPath}
          onScenarioContextMenu={(e, path, name) =>
            setScenarioCtxMenu({ x: e.clientX, y: e.clientY, path, name })
          }
        />
        {runAllState && (
          <RunAllResults
            state={runAllState}
            onSelectScenario={onSelectScenario}
            onDismiss={onDismissRunAll}
          />
        )}
      </div>

      {scenarioCtxMenu && (
        <ContextMenu
          x={scenarioCtxMenu.x}
          y={scenarioCtxMenu.y}
          items={buildScenarioMenuItems(scenarioCtxMenu, {
            onOpen: (path, name) => onSelectScenario?.(path, name),
            onDuplicate: onDuplicateScenario,
            onRename: setRenamingScenarioPath,
            onDelete: onDeleteScenario,
          })}
          onClose={() => setScenarioCtxMenu(null)}
        />
      )}

      {showTemplates && (
        <ScenarioTemplateGallery
          templates={templates}
          loading={templatesLoading}
          onUse={handleUseTemplate}
          onClose={() => setShowTemplates(false)}
        />
      )}
    </div>
  )
}
