import { useState } from 'react'
import { PlayCircle } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { ScenarioInfo, TemplateInfo } from '@/lib/bridge'
import { listTemplates, createFromTemplate } from '@/lib/bridge'
import type { RunAllState } from '@/types'
import type { FileClipboard } from '@/types'
import { WorkspaceScenarioSection } from './WorkspaceScenarioSection'
import { RunAllResults } from './WorkspaceRunResults'
import { ContextMenu } from './ContextMenu'
import { ScenarioTemplateGallery } from './ScenarioTemplateGallery'
import { ScenarioRunLog } from './ScenarioRunLog'
import { buildScenarioMenuItems, buildScenarioFolderMenuItems } from './workspaceMenus'

type ScenariosPanelProps = {
  workspacePath: string
  scenarios: ScenarioInfo[]
  scenarioError?: string | null
  runAllState?: RunAllState
  onRunAllScenarios?: () => void
  onCancelRunAll?: () => void
  onDismissRunAll?: () => void
  onSelectScenario?: (path: string, name: string) => void
  onNewScenario?: (name: string, folderPath?: string) => void
  onNewScenarioFile?: (path: string, name: string) => void
  onDuplicateScenario?: (path: string) => void
  onDeleteScenario?: (path: string) => void
  onRenameScenario?: (path: string, newName: string) => void
  onNewFolder?: (parentFolderPath: string, name: string) => void
  onDeleteScenarioFolder?: (folderPath: string) => void
  onRenameScenarioFolder?: (folderPath: string, newName: string) => void
  onRunFolder?: (folderPath: string) => void
  onRefresh?: () => void
  onMoveScenario?: (srcPath: string, destFolderRelPath: string) => void
  clipboard?: FileClipboard | null
  onCopyScenario?: (path: string) => void
  onCutScenario?: (path: string) => void
  onPasteToScenarioFolder?: (relFolderPath: string) => void
  runLogTrigger?: number
}

export function ScenariosPanel({
  workspacePath, scenarios, scenarioError,
  runAllState, onRunAllScenarios, onCancelRunAll, onDismissRunAll,
  onSelectScenario, onNewScenario, onNewScenarioFile,
  onDuplicateScenario, onDeleteScenario, onRenameScenario,
  onNewFolder, onDeleteScenarioFolder, onRenameScenarioFolder, onRunFolder,
  onRefresh, onMoveScenario, clipboard, onCopyScenario, onCutScenario, onPasteToScenarioFolder,
  runLogTrigger,
}: ScenariosPanelProps) {
  const [activeTab, setActiveTab] = useState<'scenarios' | 'log'>('scenarios')
  const [renamingScenarioPath, setRenamingScenarioPath] = useState<string | null>(null)
  const [renamingFolderPath, setRenamingFolderPath] = useState<string | null>(null)
  const [showTemplates, setShowTemplates] = useState(false)
  const [templates, setTemplates] = useState<TemplateInfo[]>([])
  const [templatesLoading, setTemplatesLoading] = useState(false)
  const [templateError, setTemplateError] = useState<string | null>(null)
  const [scenarioCtxMenu, setScenarioCtxMenu] = useState<{
    x: number; y: number; path: string; name: string
  } | null>(null)
  const [folderCtxMenu, setFolderCtxMenu] = useState<{
    x: number; y: number; folderPath: string; name: string
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
    setTemplateError(null)
    const result = await createFromTemplate(templateID, name)
    if (result.error) {
      setTemplateError(result.error)
      return
    }
    if (result.path) {
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
      {/* Tab strip */}
      <div className="shrink-0 flex border-b border-[var(--color-border)] bg-[var(--color-background)]">
        {(['scenarios', 'log'] as const).map(tab => (
          <button
            key={tab}
            type="button"
            onClick={() => setActiveTab(tab)}
            className={cn(
              'px-3 py-1.5 text-[10px] font-semibold uppercase tracking-widest transition-colors border-b-2',
              activeTab === tab
                ? 'text-[var(--color-foreground)] border-[var(--color-accent)]'
                : 'text-[var(--color-muted-foreground)] border-transparent hover:text-[var(--color-foreground)]',
            )}
          >
            {tab === 'scenarios' ? 'Scenarios' : 'Run Log'}
          </button>
        ))}
      </div>

      {templateError && (
        <div className="shrink-0 px-2 py-1.5 text-[10px] text-red-400 bg-red-500/10 border-b border-red-500/20 flex items-center gap-1">
          <span className="flex-1">Failed to create from template: {templateError}</span>
          <button type="button" onClick={() => setTemplateError(null)} className="shrink-0 text-red-400/60 hover:text-red-400">✕</button>
        </div>
      )}

      {activeTab === 'log' && (
        <ScenarioRunLog workspacePath={workspacePath} refreshTrigger={runLogTrigger} onSelectScenario={onSelectScenario} />
      )}

      {activeTab === 'scenarios' && (
      <>
        <div className="flex-1 overflow-auto py-1">
        <WorkspaceScenarioSection
          scenarios={scenarios}
          scenarioError={scenarioError}
          runAllState={runAllState}
          onRunAllScenarios={onRunAllScenarios}
          onNewScenario={onNewScenario}
          onNewFolder={onNewFolder}
          onSelectScenario={onSelectScenario}
          onDuplicateScenario={onDuplicateScenario}
          onDeleteScenario={onDeleteScenario}
          onRenameScenario={onRenameScenario}
          onDeleteFolder={onDeleteScenarioFolder}
          onRenameFolder={onRenameScenarioFolder}
          onRunFolder={onRunFolder}
          onOpenTemplateGallery={openTemplateGallery}
          onRefresh={onRefresh}
          onMoveScenario={onMoveScenario}
          clipboard={clipboard}
          renamingScenarioPath={renamingScenarioPath}
          onRenamingChange={setRenamingScenarioPath}
          renamingFolderPath={renamingFolderPath}
          onRenamingFolderChange={setRenamingFolderPath}
          onScenarioContextMenu={(e, path, name) =>
            setScenarioCtxMenu({ x: e.clientX, y: e.clientY, path, name })
          }
          onFolderContextMenu={(e, folderPath, name) =>
            setFolderCtxMenu({ x: e.clientX, y: e.clientY, folderPath, name })
          }
        />
        {runAllState && (
          <RunAllResults
            state={runAllState}
            onSelectScenario={onSelectScenario}
            onCancel={onCancelRunAll}
            onDismiss={onDismissRunAll}
          />
        )}
      </div>
      </>
      )}

      {scenarioCtxMenu && (
        <ContextMenu
          x={scenarioCtxMenu.x}
          y={scenarioCtxMenu.y}
          items={buildScenarioMenuItems(scenarioCtxMenu, {
            onOpen: (path, name) => onSelectScenario?.(path, name),
            onDuplicate: onDuplicateScenario,
            onRename: setRenamingScenarioPath,
            onDelete: onDeleteScenario,
            onCopy: onCopyScenario,
            onCut: onCutScenario,
          })}
          onClose={() => setScenarioCtxMenu(null)}
        />
      )}

      {folderCtxMenu && (
        <ContextMenu
          x={folderCtxMenu.x}
          y={folderCtxMenu.y}
          items={buildScenarioFolderMenuItems(folderCtxMenu.folderPath, {
            onRename: fp => { setFolderCtxMenu(null); setRenamingFolderPath(fp) },
            onDelete: fp => { setFolderCtxMenu(null); onDeleteScenarioFolder?.(fp) },
            onRunFolder: fp => { setFolderCtxMenu(null); onRunFolder?.(fp) },
            onPasteToFolder: onPasteToScenarioFolder,
            clipboard,
          })}
          onClose={() => setFolderCtxMenu(null)}
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
