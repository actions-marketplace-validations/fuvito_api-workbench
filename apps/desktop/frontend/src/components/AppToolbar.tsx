import { FolderOpen, Save, FilePlus, CopyPlus } from 'lucide-react'
import { cn } from '@/lib/utils'
import { SnippetButton } from '@/components/SnippetButton'
import { isTabDirty } from '@/lib/tabState'
import type { TabState } from '@/lib/tabState'
import { METHOD_COLOR } from '@/lib/httpUtils'

type AppToolbarProps = {
  activeTab: TabState
  activeTabId: string
  updateTab: (id: string, patch: Partial<TabState>) => void
  onOpenRequest: () => void
  onSaveRequest: () => void
  onNewRequest: () => void
  onDuplicateTab: () => void
}

export function AppToolbar({
  activeTab, activeTabId, updateTab,
  onOpenRequest, onSaveRequest, onNewRequest, onDuplicateTab,
}: AppToolbarProps) {
  const hasDescription = activeTab.kind !== 'scenario' && !!activeTab.description

  return (
    <header className="flex items-center gap-2 px-3 py-1.5 border-b border-[var(--color-border)] bg-[var(--color-card)] shrink-0">
      <div className={cn('min-w-0 flex-1', hasDescription ? 'flex flex-col gap-0' : 'flex items-center gap-1.5')}>
        {activeTab.kind === 'scenario' ? (
          <span className="text-xs text-[var(--color-muted-foreground)] truncate">
            {activeTab.name ?? 'Scenario'}
          </span>
        ) : (
          <>
            <div className="flex items-center gap-1.5 min-w-0">
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
            {hasDescription && (
              <p className="text-[11px] text-[var(--color-muted-foreground)] truncate leading-tight">
                {activeTab.description}
              </p>
            )}
          </>
        )}
      </div>

      <div className="ml-auto flex items-center gap-1 shrink-0">
        <button type="button" onClick={onOpenRequest} title="Open request file"
          className="p-1.5 rounded text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-muted)] transition-colors">
          <FolderOpen size={13} />
        </button>
        {activeTab.kind !== 'scenario' && (
          <button type="button" onClick={onSaveRequest} title="Save request (Ctrl+S)"
            className="flex items-center gap-1 px-2 py-1 rounded text-xs text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-muted)] transition-colors">
            <Save size={13} />
            <span>Save</span>
          </button>
        )}
        <button type="button" onClick={onNewRequest} title="New request tab (Ctrl+N)"
          className="flex items-center gap-1 px-2 py-1 rounded text-xs text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-muted)] transition-colors">
          <FilePlus size={13} />
          <span>New</span>
        </button>
        {activeTab.kind !== 'scenario' && (
          <button type="button" onClick={onDuplicateTab} title="Duplicate tab"
            className="p-1.5 rounded text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-muted)] transition-colors">
            <CopyPlus size={13} />
          </button>
        )}
        {activeTab.kind !== 'scenario' && <SnippetButton tab={activeTab} />}
      </div>
    </header>
  )
}
