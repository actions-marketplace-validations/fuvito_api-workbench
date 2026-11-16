import { useState } from 'react'
import { Clock } from 'lucide-react'
import { cn } from '@/lib/utils'
import { METHOD_BADGE, statusColor } from '@/lib/httpUtils'
import { WorkspacePanel } from '@/components/WorkspacePanel'
import { ScenariosPanel } from '@/components/ScenariosPanel'
import { CoverageSidebarPanel } from '@/components/CoverageSidebarPanel'
import type { RailTab } from '@/components/ActivityRail'
import type { HistoryEntry, FileNode, ScenarioInfo } from '@/lib/bridge'
import type { FileClipboard, RunAllEntry, RunAllState, FolderRunState } from '@/types'

export type { RunAllEntry, RunAllState, FolderRunState }

export type SidebarProps = {
  tab: RailTab
  workspacePath: string
  fileTree: FileNode[]
  history: HistoryEntry[]
  scenarios?: ScenarioInfo[]
  onOpenWorkspace: () => void
  onNewWorkspace: () => void
  onSelectFile: (path: string) => void
  onDeleteFile: (path: string) => void
  onDuplicateFile: (path: string) => void
  onDeleteFolder: (path: string) => void
  onNewRequestInFolder: (folderPath: string) => void
  onCreateFolder: (parentPath: string) => void
  onRenameFile: (path: string, newName: string) => void
  onRenameFolder: (path: string, newName: string) => void
  onMoveFile: (srcPath: string, destDir: string) => void
  onSelectHistory: (entry: HistoryEntry) => void
  onClearHistory: () => void
  onImportOpenAPI: () => void
  onImportPostman?: () => void
  onImportInsomnia?: () => void
  onTryDemo?: () => void
  demoStarting?: boolean
  onRefresh?: () => void
  importing?: boolean
  importError?: string
  activeFilePath?: string
  dirtyFilePaths?: Set<string>
  onSelectScenario?: (path: string, name: string) => void
  onNewScenario?: (name: string, folderPath?: string) => void
  onNewScenarioFile?: (path: string, name: string) => void
  onDuplicateScenario?: (path: string) => void
  onDeleteScenario?: (path: string) => void
  onRenameScenario?: (path: string, newName: string) => void
  onNewScenarioFolder?: (parentFolderPath: string, name: string) => void
  onDeleteScenarioFolder?: (folderPath: string) => void
  onRenameScenarioFolder?: (folderPath: string, newName: string) => void
  runAllState?: RunAllState
  runLogTrigger?: number
  onRunAllScenarios?: () => void
  onCancelRunAll?: () => void
  onDismissRunAll?: () => void
  scenarioError?: string | null
  onRunFolder?: (folderPath: string) => void
  onRunScenarioFolder?: (folderPath: string) => void
  folderRunState?: FolderRunState | null
  onDismissFolderRun?: () => void
  selectedCoverageSpecFile?: string | null
  onSelectCoverageSpec?: (specFile: string) => void
  clipboard?: FileClipboard | null
  onCopyFile?: (path: string) => void
  onCutFile?: (path: string) => void
  onPasteToRequestFolder?: (folderPath: string) => void
  onPasteToRequestRoot?: () => void
  onMoveScenario?: (srcPath: string, destFolderRelPath: string) => void
  onCopyScenario?: (path: string) => void
  onCutScenario?: (path: string) => void
  onPasteToScenarioFolder?: (relFolderPath: string) => void
}

export function Sidebar({
  tab,
  workspacePath, fileTree, history, scenarios,
  onOpenWorkspace, onNewWorkspace,
  onSelectFile, onDeleteFile, onDuplicateFile, onDeleteFolder, onNewRequestInFolder, onCreateFolder,
  onRenameFile, onRenameFolder, onMoveFile,
  onSelectHistory, onClearHistory,
  onImportOpenAPI, onImportPostman, onImportInsomnia, onTryDemo, demoStarting, onRefresh, importing, importError,
  activeFilePath, dirtyFilePaths,
  onSelectScenario, onNewScenario, onNewScenarioFile,
  onDuplicateScenario, onDeleteScenario, onRenameScenario,
  onNewScenarioFolder, onDeleteScenarioFolder, onRenameScenarioFolder,
  runAllState, onRunAllScenarios, onCancelRunAll, onDismissRunAll, scenarioError, runLogTrigger,
  onRunFolder, onRunScenarioFolder, folderRunState, onDismissFolderRun,
  selectedCoverageSpecFile, onSelectCoverageSpec,
  clipboard, onCopyFile, onCutFile, onPasteToRequestFolder, onPasteToRequestRoot,
  onMoveScenario, onCopyScenario, onCutScenario, onPasteToScenarioFolder,
}: SidebarProps) {
  const [confirmingClear, setConfirmingClear] = useState(false)

  return (
    <div className="flex flex-col h-full overflow-hidden">
      {tab === 'explorer' && (
        <WorkspacePanel
          workspacePath={workspacePath}
          fileTree={fileTree}
          onOpenWorkspace={onOpenWorkspace}
          onNewWorkspace={onNewWorkspace}
          onSelectFile={onSelectFile}
          onDeleteFile={onDeleteFile}
          onDuplicateFile={onDuplicateFile}
          onDeleteFolder={onDeleteFolder}
          onNewRequestInFolder={onNewRequestInFolder}
          onCreateFolder={onCreateFolder}
          onRenameFile={onRenameFile}
          onRenameFolder={onRenameFolder}
          onMoveFile={onMoveFile}
          onImportOpenAPI={onImportOpenAPI}
          onImportPostman={onImportPostman}
          onImportInsomnia={onImportInsomnia}
          onTryDemo={onTryDemo}
          demoStarting={demoStarting}
          onRefresh={onRefresh}
          importing={importing}
          importError={importError}
          activeFilePath={activeFilePath}
          dirtyFilePaths={dirtyFilePaths}
          onRunFolder={onRunFolder}
          folderRunState={folderRunState}
          onDismissFolderRun={onDismissFolderRun}
          clipboard={clipboard}
          onCopyFile={onCopyFile}
          onCutFile={onCutFile}
          onPasteToFolder={onPasteToRequestFolder}
          onPasteToRoot={onPasteToRequestRoot}
        />
      )}

      {tab === 'scenarios' && (
        <ScenariosPanel
          workspacePath={workspacePath}
          scenarios={scenarios ?? []}
          scenarioError={scenarioError}
          runAllState={runAllState}
          onRunAllScenarios={onRunAllScenarios}
          onCancelRunAll={onCancelRunAll}
          onDismissRunAll={onDismissRunAll}
          onSelectScenario={onSelectScenario}
          onNewScenario={onNewScenario}
          onNewScenarioFile={onNewScenarioFile}
          onDuplicateScenario={onDuplicateScenario}
          onDeleteScenario={onDeleteScenario}
          onRenameScenario={onRenameScenario}
          onNewFolder={onNewScenarioFolder}
          onDeleteScenarioFolder={onDeleteScenarioFolder}
          onRenameScenarioFolder={onRenameScenarioFolder}
          onRefresh={onRefresh}
          onMoveScenario={onMoveScenario}
          clipboard={clipboard}
          onCopyScenario={onCopyScenario}
          onCutScenario={onCutScenario}
          onPasteToScenarioFolder={onPasteToScenarioFolder}
          runLogTrigger={runLogTrigger}
          onRunFolder={onRunScenarioFolder}
        />
      )}

      {tab === 'coverage' && (
        <CoverageSidebarPanel
          workspacePath={workspacePath}
          selectedSpecFile={selectedCoverageSpecFile ?? null}
          onSelectSpec={sf => onSelectCoverageSpec?.(sf)}
        />
      )}

      {tab === 'history' && (
        <div className="flex flex-col flex-1 overflow-hidden">
          {history.length > 0 && (
            <div className="flex items-center justify-end px-2 py-1 border-b border-[var(--color-border)] shrink-0 gap-2">
              {confirmingClear ? (
                <>
                  <span className="text-[10px] text-[var(--color-muted-foreground)]">Clear all history?</span>
                  <button
                    type="button"
                    onClick={() => { setConfirmingClear(false); onClearHistory() }}
                    className="text-[10px] text-red-400 hover:text-red-300 transition-colors font-semibold"
                  >
                    Clear
                  </button>
                  <button
                    type="button"
                    onClick={() => setConfirmingClear(false)}
                    className="text-[10px] text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors"
                  >
                    Cancel
                  </button>
                </>
              ) : (
                <button
                  type="button"
                  onClick={() => setConfirmingClear(true)}
                  className="text-[10px] text-[var(--color-muted-foreground)] hover:text-red-400 transition-colors"
                >
                  Clear history
                </button>
              )}
            </div>
          )}
          <div className="flex-1 overflow-auto">
            {history.length === 0 ? (
              <div className="flex flex-col items-center gap-2 px-4 py-8 text-center">
                <Clock size={20} className="text-[var(--color-muted-foreground)] opacity-30" />
                <p className="text-xs text-[var(--color-muted-foreground)]">No requests yet</p>
                <p className="text-[11px] text-[var(--color-muted-foreground)] opacity-60">Send a request to see it here</p>
              </div>
            ) : (
              <p className="px-3 py-1.5 text-[10px] text-[var(--color-muted-foreground)] italic border-b border-[var(--color-border)]">
                Restores method, URL, headers, and body · auth not stored
              </p>
            )}
            {history.length > 0 && (
              <ul>
                {history.map(entry => (
                  <li key={entry.id}>
                    <button
                      type="button"
                      onClick={() => onSelectHistory(entry)}
                      className={cn(
                        'w-full text-left px-3 py-2 flex flex-col gap-0.5',
                        'hover:bg-[var(--color-muted)] transition-colors border-b border-[var(--color-border)]',
                      )}
                    >
                      <div className="flex items-center gap-1.5">
                        <span className={cn('text-[10px] font-mono font-bold shrink-0', METHOD_BADGE[entry.method] ?? 'text-[var(--color-foreground)]')}>
                          {entry.method}
                        </span>
                        <span className={cn('text-[10px] font-mono', statusColor(entry.statusCode))}>
                          {entry.statusCode || '—'}
                        </span>
                        <span className="ml-auto text-[10px] text-[var(--color-muted-foreground)] shrink-0">
                          {entry.durationMs}ms
                        </span>
                      </div>
                      <span className="text-[10px] text-[var(--color-muted-foreground)] truncate font-mono w-full">
                        {entry.url}
                      </span>
                    </button>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </div>
      )}
    </div>
  )
}
