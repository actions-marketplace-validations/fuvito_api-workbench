import { cn } from '@/lib/utils'
import { METHOD_BADGE, statusColor } from '@/lib/httpUtils'
import { WorkspacePanel } from '@/components/WorkspacePanel'
import type { HistoryEntry, FileNode } from '@/lib/bridge'

export type SidebarTab = 'workspace' | 'history'

export type SidebarProps = {
  width: number
  tab: SidebarTab
  onTabChange: (t: SidebarTab) => void
  workspacePath: string
  fileTree: FileNode[]
  history: HistoryEntry[]
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
  onSelectHistory: (entry: HistoryEntry) => void
  onImportOpenAPI: () => void
  importing?: boolean
  activeFilePath?: string
  isDirty?: boolean
}

export function Sidebar({
  width,
  tab, onTabChange,
  workspacePath, fileTree, history,
  onOpenWorkspace, onNewWorkspace,
  onSelectFile, onDeleteFile, onDuplicateFile, onDeleteFolder, onNewRequestInFolder, onCreateFolder,
  onRenameFile, onRenameFolder,
  onSelectHistory,
  onImportOpenAPI, importing,
  activeFilePath, isDirty,
}: SidebarProps) {
  return (
    <aside
      className="shrink-0 flex flex-col border-r border-[var(--color-border)] bg-[var(--color-card)]"
      style={{ width }}
    >
      {/* Tab bar */}
      <div className="flex border-b border-[var(--color-border)]">
        {(['workspace', 'history'] as SidebarTab[]).map(t => (
          <button
            key={t}
            type="button"
            onClick={() => onTabChange(t)}
            className={cn(
              'flex-1 py-1.5 text-[10px] font-semibold uppercase tracking-widest transition-colors',
              tab === t
                ? 'text-[var(--color-foreground)] border-b-2 border-[var(--color-accent)]'
                : 'text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] border-b-2 border-transparent',
            )}
          >
            {t}
          </button>
        ))}
      </div>

      <div className="flex-1 overflow-hidden flex flex-col">
        {tab === 'workspace' && (
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
            onImportOpenAPI={onImportOpenAPI}
            importing={importing}
            activeFilePath={activeFilePath}
            isDirty={isDirty}
          />
        )}

        {tab === 'history' && (
          <div className="flex-1 overflow-auto">
            {history.length === 0 ? (
              <p className="p-3 text-xs text-[var(--color-muted-foreground)]">No requests yet.</p>
            ) : (
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
        )}
      </div>
    </aside>
  )
}
