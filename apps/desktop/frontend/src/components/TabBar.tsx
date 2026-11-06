import { X, Plus } from 'lucide-react'
import { cn } from '@/lib/utils'

export type TabInfo = {
  id: string
  name: string | null
  method: string
  isDirty: boolean
  kind?: 'request' | 'scenario'
}

const TAB_METHOD_COLOR: Record<string, string> = {
  GET:     'text-emerald-400',
  POST:    'text-blue-400',
  PUT:     'text-amber-400',
  PATCH:   'text-orange-400',
  DELETE:  'text-red-400',
  HEAD:    'text-purple-400',
  OPTIONS: 'text-cyan-400',
}

const TAB_METHOD_ABBR: Record<string, string> = {
  GET: 'GET', POST: 'PST', PUT: 'PUT', PATCH: 'PCH',
  DELETE: 'DEL', HEAD: 'HED', OPTIONS: 'OPT',
}

type TabBarProps = {
  tabs: TabInfo[]
  activeTabId: string
  onSelect: (id: string) => void
  onClose: (id: string) => void
  onNew: () => void
}

export function TabBar({ tabs, activeTabId, onSelect, onClose, onNew }: TabBarProps) {
  return (
    <div className="flex items-stretch shrink-0 bg-[var(--color-background)] border-b border-[var(--color-border)] overflow-x-auto">
      {tabs.map(tab => {
        const isActive = tab.id === activeTabId
        return (
          <div
            key={tab.id}
            role="tab"
            aria-selected={isActive}
            onClick={() => onSelect(tab.id)}
            className={cn(
              'group flex items-center gap-1.5 px-3 h-8 text-xs shrink-0 max-w-[200px] min-w-[80px]',
              'border-t-2 cursor-pointer select-none transition-colors',
              isActive
                ? 'bg-[var(--color-card)] border-t-[var(--color-accent)] text-[var(--color-foreground)]'
                : 'bg-transparent border-t-transparent text-[var(--color-muted-foreground)] hover:bg-[var(--color-muted)] hover:text-[var(--color-foreground)]',
            )}
          >
            {tab.isDirty && (
              <span className="shrink-0 w-1.5 h-1.5 rounded-full bg-amber-400" title="Unsaved changes" />
            )}
            {tab.kind === 'scenario' ? (
              <span className="shrink-0 font-mono font-bold text-[10px] text-violet-400">SCN</span>
            ) : (
              <span className={cn('shrink-0 font-mono font-bold text-[10px]', TAB_METHOD_COLOR[tab.method] ?? '')}>
                {TAB_METHOD_ABBR[tab.method] ?? tab.method.slice(0, 3)}
              </span>
            )}
            <span className="truncate flex-1 text-[11px]">
              {tab.kind === 'scenario' ? (tab.name ?? 'Scenario') : (tab.name ?? 'New Request')}
            </span>
            <button
              type="button"
              onClick={e => { e.stopPropagation(); onClose(tab.id) }}
              title="Close tab"
              className={cn(
                'shrink-0 rounded p-0.5 transition-opacity',
                tab.isDirty ? 'opacity-100' : 'opacity-0 group-hover:opacity-100',
                'hover:bg-[var(--color-muted)]',
              )}
            >
              <X size={10} />
            </button>
          </div>
        )
      })}
      <button
        type="button"
        onClick={onNew}
        title="New request tab"
        className="h-8 px-3 shrink-0 text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-muted)] transition-colors"
      >
        <Plus size={12} />
      </button>
    </div>
  )
}
