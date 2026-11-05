import { FolderOpen, Save, FilePlus, Settings, Pencil, CopyPlus } from 'lucide-react'
import { cn } from '@/lib/utils'
import { SnippetButton } from '@/components/SnippetButton'
import { isTabDirty } from '@/lib/tabState'
import type { TabState } from '@/lib/tabState'
import { METHOD_COLOR } from '@/lib/httpUtils'
import type { EnvInfo } from '@/lib/bridge'

type AppToolbarProps = {
  activeTab: TabState
  activeTabId: string
  updateTab: (id: string, patch: Partial<TabState>) => void
  environments: EnvInfo[]
  activeEnv: EnvInfo | null
  workspacePath: string
  demoRunning: boolean
  demoPort: number
  onOpenRequest: () => void
  onSaveRequest: () => void
  onNewRequest: () => void
  onDuplicateTab: () => void
  onShowSettings: () => void
  onEnvChange: (path: string) => void
  onEditEnv: () => void
  onStopDemo: () => void
}

export function AppToolbar({
  activeTab, activeTabId, updateTab,
  environments, activeEnv, workspacePath,
  demoRunning, demoPort,
  onOpenRequest, onSaveRequest, onNewRequest, onDuplicateTab,
  onShowSettings, onEnvChange, onEditEnv, onStopDemo,
}: AppToolbarProps) {
  return (
    <header className="flex items-center gap-2 px-3 py-1.5 border-b border-[var(--color-border)] bg-[var(--color-card)] shrink-0">
      <div className="flex items-center gap-1.5 min-w-0 flex-1">
        {isTabDirty(activeTab) && (
          <span className="shrink-0 w-2 h-2 rounded-full bg-amber-400" title="Unsaved changes" />
        )}
        {activeTab.name !== null && (
          <span className={cn('text-xs font-mono font-bold shrink-0', METHOD_COLOR[activeTab.method])}>
            {activeTab.method}
          </span>
        )}
        {activeTab.editingName ? (
          <input
            autoFocus
            type="text"
            value={activeTab.name ?? ''}
            onChange={e => updateTab(activeTabId, { name: e.target.value })}
            onBlur={() => {
              if (!activeTab.name?.trim()) updateTab(activeTabId, { name: null, editingName: false })
              else updateTab(activeTabId, { editingName: false })
            }}
            onKeyDown={e => {
              if (e.key === 'Enter' || e.key === 'Escape') {
                if (!activeTab.name?.trim()) updateTab(activeTabId, { name: null, editingName: false })
                else updateTab(activeTabId, { editingName: false })
              }
            }}
            className={cn(
              'text-xs font-medium bg-transparent outline-none',
              'border-b border-[var(--color-accent)]',
              'text-[var(--color-foreground)] w-40',
            )}
          />
        ) : activeTab.name !== null ? (
          <button
            type="button"
            onClick={() => updateTab(activeTabId, { editingName: true, saveError: null })}
            title="Click to rename"
            className="text-xs font-medium text-[var(--color-foreground)] hover:text-[var(--color-accent)] truncate transition-colors text-left"
          >
            {activeTab.name}
          </button>
        ) : (
          <button
            type="button"
            onClick={() => updateTab(activeTabId, { name: '', editingName: true })}
            title="Click to name this request"
            className="text-xs text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors"
          >
            New Request
          </button>
        )}
        {activeTab.saveError && (
          <span className="text-label text-[var(--color-destructive)] truncate ml-1">
            {activeTab.saveError}
          </span>
        )}
      </div>

      <div className="ml-auto flex items-center gap-2 shrink-0">
        <button type="button" onClick={onOpenRequest} title="Open request file"
          className="flex items-center gap-1 text-xs text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors">
          <FolderOpen size={13} />
        </button>
        <button type="button" onClick={onSaveRequest} title="Save request (Ctrl+S)"
          className="flex items-center gap-1 text-xs text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors">
          <Save size={13} />
        </button>
        <button type="button" onClick={onNewRequest} title="New request tab (Ctrl+N)"
          className="flex items-center gap-1 text-xs text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors">
          <FilePlus size={13} />
        </button>
        <button type="button" onClick={onDuplicateTab} title="Duplicate tab"
          className="flex items-center gap-1 text-xs text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors">
          <CopyPlus size={13} />
        </button>
        <SnippetButton tab={activeTab} />
        <button type="button" onClick={onShowSettings} title="Settings"
          className="flex items-center gap-1 text-xs text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors">
          <Settings size={13} />
        </button>

        {environments.length > 0 && (
          <div className="w-px h-4 bg-[var(--color-border)]" />
        )}

        {environments.length > 0 && (
          <select
            value={activeEnv?.path ?? ''}
            onChange={e => onEnvChange(e.target.value)}
            className={cn(
              'bg-[var(--color-muted)] text-xs rounded px-2 py-0.5',
              'border border-[var(--color-border)] outline-none cursor-pointer',
              activeEnv ? 'text-[var(--color-accent)]' : 'text-[var(--color-muted-foreground)]',
            )}
          >
            <option value="">No environment</option>
            {environments.map(env => (
              <option key={env.path} value={env.path}>{env.name}</option>
            ))}
          </select>
        )}

        {workspacePath && (
          <button type="button" onClick={onEditEnv}
            title={activeEnv ? `Edit environment: ${activeEnv.name}` : 'Create environment'}
            className="flex items-center gap-1 text-xs text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors">
            <Pencil size={13} />
          </button>
        )}

        {demoRunning && (
          <>
            <div className="w-px h-4 bg-[var(--color-border)]" />
            <div className="flex items-center gap-1.5 text-xs">
              <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse shrink-0" />
              <button
                type="button"
                onClick={() => navigator.clipboard.writeText(`http://localhost:${demoPort}`).catch(() => {})}
                title={`Copy base URL: http://localhost:${demoPort}`}
                className="text-emerald-400 font-mono hover:text-emerald-300 transition-colors"
              >
                Demo :{demoPort}
              </button>
              <button type="button" onClick={onStopDemo} title="Stop demo server"
                className="text-[var(--color-muted-foreground)] hover:text-red-400 transition-colors leading-none">
                ✕
              </button>
            </div>
          </>
        )}
      </div>
    </header>
  )
}
