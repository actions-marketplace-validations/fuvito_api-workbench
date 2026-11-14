import { FolderOpen, Play, Clock, Settings, HelpCircle, ChevronsRight, ChevronsLeft } from 'lucide-react'
import { cn } from '@/lib/utils'

export type RailTab = 'explorer' | 'scenarios' | 'history'

type NavItem = {
  tab: RailTab
  icon: React.ComponentType<{ size?: number; className?: string }>
  label: string
  title: string
  activeColor: string
}

const NAV_ITEMS: NavItem[] = [
  { tab: 'explorer',  icon: FolderOpen, label: 'Explorer',  title: 'Explorer',  activeColor: 'text-sky-400' },
  { tab: 'scenarios', icon: Play,       label: 'Scenarios', title: 'Scenarios', activeColor: 'text-violet-400' },
  { tab: 'history',   icon: Clock,      label: 'History',   title: 'History',   activeColor: 'text-amber-400' },
]

type ActivityRailProps = {
  activeTab: RailTab
  sidebarOpen: boolean
  expanded: boolean
  onTabClick: (tab: RailTab) => void
  onToggleExpanded: () => void
  onShowSettings: () => void
  onShowHelp: () => void
  demoRunning?: boolean
}

export function ActivityRail({
  activeTab, sidebarOpen, expanded,
  onTabClick, onToggleExpanded, onShowSettings, onShowHelp, demoRunning,
}: ActivityRailProps) {
  return (
    <div
      className="shrink-0 flex flex-col border-r border-[var(--color-border)] bg-[var(--color-card)] overflow-hidden transition-[width] duration-150"
      style={{ width: expanded ? 160 : 44 }}
    >
      {/* Primary nav */}
      <nav className="flex flex-col py-1 mt-1">
        {NAV_ITEMS.map(({ tab, icon: Icon, label, title, activeColor }) => {
          const active = sidebarOpen && activeTab === tab
          return (
            <button
              key={tab}
              type="button"
              title={expanded ? undefined : title}
              onClick={() => onTabClick(tab)}
              className={cn(
                'relative flex items-center h-9 transition-colors',
                expanded ? 'gap-2.5 px-3' : 'justify-center w-11',
                active
                  ? cn(activeColor, 'bg-[var(--color-muted)]')
                  : 'text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-muted)]',
              )}
            >
              {active && (
                <span className="absolute left-0 inset-y-1 w-[3px] rounded-r-full bg-current opacity-70" />
              )}
              <Icon size={16} className="shrink-0" />
              {expanded && (
                <span className="text-xs font-medium truncate">{label}</span>
              )}
            </button>
          )
        })}
      </nav>

      {/* Bottom: settings + help + expand toggle */}
      <div className="mt-auto flex flex-col py-1 border-t border-[var(--color-border)]">
        {demoRunning && (
          <div className={cn('flex items-center py-1', expanded ? 'px-3 gap-2' : 'justify-center')}>
            <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse shrink-0" />
            {expanded && <span className="text-[10px] text-emerald-400">Demo running</span>}
          </div>
        )}

        {([
          { icon: Settings,   label: 'Settings', title: 'Settings', onClick: onShowSettings },
          { icon: HelpCircle, label: 'Help',     title: 'Help',     onClick: onShowHelp },
        ] as { icon: React.ComponentType<{ size?: number; className?: string }>; label: string; title: string; onClick: () => void }[]).map(({ icon: Icon, label, title, onClick }) => (
          <button
            key={label}
            type="button"
            title={expanded ? undefined : title}
            onClick={onClick}
            className={cn(
              'flex items-center h-9 text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-muted)] transition-colors',
              expanded ? 'gap-2.5 px-3' : 'justify-center w-11',
            )}
          >
            <Icon size={16} className="shrink-0" />
            {expanded && <span className="text-xs font-medium">{label}</span>}
          </button>
        ))}

        {/* Expand / collapse toggle */}
        <button
          type="button"
          title={expanded ? 'Collapse rail' : 'Expand rail'}
          onClick={onToggleExpanded}
          className={cn(
            'flex items-center h-8 text-[var(--color-border)] hover:text-[var(--color-muted-foreground)] hover:bg-[var(--color-muted)] transition-colors',
            expanded ? 'gap-2.5 px-3' : 'justify-center w-11',
          )}
        >
          {expanded
            ? <ChevronsLeft size={14} className="shrink-0" />
            : <ChevronsRight size={14} className="shrink-0" />
          }
          {expanded && <span className="text-[10px]">Collapse</span>}
        </button>
      </div>
    </div>
  )
}
