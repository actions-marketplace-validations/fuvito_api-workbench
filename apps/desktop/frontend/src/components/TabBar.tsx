import { useState, useRef, useEffect, useCallback } from 'react'
import { X, Plus } from 'lucide-react'
import { cn } from '@/lib/utils'
import { ContextMenu } from '@/components/ContextMenu'

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
  onSave?: (id: string) => void
  onCloseOthers?: (id: string) => void
  onCloseToRight?: (id: string) => void
}

export function TabBar({ tabs, activeTabId, onSelect, onClose, onNew, onSave, onCloseOthers, onCloseToRight }: TabBarProps) {
  const [ctxMenu, setCtxMenu] = useState<{ x: number; y: number; tabId: string } | null>(null)
  const containerRef = useRef<HTMLDivElement>(null)
  const [canScrollLeft, setCanScrollLeft] = useState(false)
  const [canScrollRight, setCanScrollRight] = useState(false)

  const updateScrollState = useCallback(() => {
    const el = containerRef.current
    if (!el) return
    setCanScrollLeft(el.scrollLeft > 2)
    setCanScrollRight(el.scrollLeft + el.clientWidth < el.scrollWidth - 2)
  }, [])

  useEffect(() => {
    const el = containerRef.current
    if (!el) return
    el.addEventListener('scroll', updateScrollState, { passive: true })
    const ro = new ResizeObserver(updateScrollState)
    ro.observe(el)
    updateScrollState()
    return () => { el.removeEventListener('scroll', updateScrollState); ro.disconnect() }
  }, [updateScrollState])

  // Scroll active tab into view
  useEffect(() => {
    const el = containerRef.current
    if (!el) return
    const activeEl = el.querySelector<HTMLElement>('[aria-selected="true"]')
    if (!activeEl) return
    const { offsetLeft, offsetWidth } = activeEl
    const { scrollLeft, clientWidth } = el
    if (offsetLeft < scrollLeft) {
      el.scrollTo({ left: offsetLeft - 8, behavior: 'smooth' })
    } else if (offsetLeft + offsetWidth > scrollLeft + clientWidth) {
      el.scrollTo({ left: offsetLeft + offsetWidth - clientWidth + 8, behavior: 'smooth' })
    }
    setTimeout(updateScrollState, 300)
  }, [activeTabId, updateScrollState])

  function handleContextMenu(e: React.MouseEvent, tabId: string) {
    e.preventDefault()
    e.stopPropagation()
    setCtxMenu({ x: e.clientX, y: e.clientY, tabId })
  }

  const ctxTab = ctxMenu ? tabs.find(t => t.id === ctxMenu.tabId) : null
  const ctxTabIndex = ctxMenu ? tabs.findIndex(t => t.id === ctxMenu.tabId) : -1

  return (
    <div className="relative flex items-stretch shrink-0 bg-[var(--color-muted)] border-b border-[var(--color-border)]">
      {/* Left fade when scrolled */}
      {canScrollLeft && (
        <div className="absolute left-0 top-0 bottom-0 w-6 bg-gradient-to-r from-[var(--color-muted)] to-transparent pointer-events-none z-10" />
      )}

      {/* Scrollable tab strip */}
      <div
        ref={containerRef}
        className="flex flex-1 items-stretch overflow-x-auto"
        style={{ scrollbarWidth: 'none' }}
      >
        {tabs.map(tab => {
          const isActive = tab.id === activeTabId
          return (
            <div
              key={tab.id}
              role="tab"
              aria-selected={isActive}
              onClick={() => onSelect(tab.id)}
              onContextMenu={e => handleContextMenu(e, tab.id)}
              className={cn(
                'group flex items-center gap-1.5 px-3 h-9 text-xs shrink-0 max-w-[200px] min-w-[80px]',
                'border-t-[3px] cursor-pointer select-none transition-colors',
                isActive
                  ? 'bg-[var(--color-card)] border-t-[var(--color-accent)] border-l border-r border-[var(--color-border)] text-[var(--color-foreground)]'
                  : 'bg-transparent border-t-transparent text-[var(--color-muted-foreground)] hover:bg-[var(--color-border)] hover:text-[var(--color-foreground)]',
              )}
            >
              {tab.isDirty && (
                <span className="shrink-0 w-1.5 h-1.5 rounded-full bg-amber-400" title="Unsaved changes" />
              )}
              {tab.kind === 'scenario' ? (
                <span className="shrink-0 font-mono font-bold text-label text-violet-400">SCN</span>
              ) : (
                <span className={cn('shrink-0 font-mono font-bold text-label', TAB_METHOD_COLOR[tab.method] ?? '')}>
                  {TAB_METHOD_ABBR[tab.method] ?? tab.method.slice(0, 3)}
                </span>
              )}
              <span className="truncate flex-1 text-xs">
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
      </div>

      {/* Right fade when more tabs exist */}
      {canScrollRight && (
        <div className="absolute right-9 top-0 bottom-0 w-8 bg-gradient-to-l from-[var(--color-muted)] to-transparent pointer-events-none z-10" />
      )}

      {/* New tab button — always visible outside scroll area */}
      <button
        type="button"
        onClick={onNew}
        title="New request tab (Ctrl+N)"
        className="h-9 px-3 shrink-0 text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-border)] transition-colors border-l border-[var(--color-border)]"
      >
        <Plus size={12} />
      </button>

      {ctxMenu && ctxTab && (
        <ContextMenu
          x={ctxMenu.x}
          y={ctxMenu.y}
          onClose={() => setCtxMenu(null)}
          items={[
            {
              kind: 'action',
              label: 'Save',
              disabled: ctxTab.kind === 'scenario' || !onSave,
              onClick: () => onSave?.(ctxMenu.tabId),
            },
            { kind: 'separator' },
            {
              kind: 'action',
              label: 'Close Tab',
              onClick: () => onClose(ctxMenu.tabId),
            },
            {
              kind: 'action',
              label: 'Close Others',
              disabled: tabs.length <= 1 || !onCloseOthers,
              onClick: () => onCloseOthers?.(ctxMenu.tabId),
            },
            {
              kind: 'action',
              label: 'Close to the Right',
              disabled: ctxTabIndex >= tabs.length - 1 || !onCloseToRight,
              onClick: () => onCloseToRight?.(ctxMenu.tabId),
            },
          ]}
        />
      )}
    </div>
  )
}
