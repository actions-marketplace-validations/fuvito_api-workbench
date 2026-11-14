import { FolderOpen, Play, Clock, Settings, HelpCircle, BarChart2, Server, PanelLeftClose } from 'lucide-react'
import { cn } from '@/lib/utils'

export type RailTab = 'explorer' | 'scenarios' | 'history'

type NavItem = {
  tab: RailTab
  icon: React.ComponentType<{ size?: number; className?: string }>
  title: string
  activeColor: string
}

const NAV_ITEMS: NavItem[] = [
  { tab: 'explorer',  icon: FolderOpen, title: 'Explorer (Ctrl+B)', activeColor: 'text-sky-400' },
  { tab: 'scenarios', icon: Play,       title: 'Scenarios',         activeColor: 'text-violet-400' },
  { tab: 'history',   icon: Clock,      title: 'History',           activeColor: 'text-amber-400' },
]

type ActivityRailProps = {
  activeTab: RailTab
  sidebarOpen: boolean
  onTabClick: (tab: RailTab) => void
  onCollapse: () => void
  onShowSettings: () => void
  onShowHelp: () => void
  demoRunning?: boolean
}

export function ActivityRail({
  activeTab, sidebarOpen, onTabClick, onCollapse, onShowSettings, onShowHelp, demoRunning,
}: ActivityRailProps) {
  return (
    <div className="shrink-0 w-11 flex flex-col border-r border-[var(--color-border)] bg-[var(--color-card)]">

      {/* Collapse rail button */}
      <div className="flex justify-center pt-1.5 pb-0.5">
        <button
          type="button"
          title="Collapse sidebar (Ctrl+B)"
          onClick={onCollapse}
          className="flex items-center justify-center w-8 h-7 rounded text-[var(--color-border)] hover:text-[var(--color-muted-foreground)] hover:bg-[var(--color-muted)] transition-colors"
        >
          <PanelLeftClose size={14} />
        </button>
      </div>

      {/* Primary nav */}
      <nav className="flex flex-col py-1">
        {NAV_ITEMS.map(({ tab, icon: Icon, title, activeColor }) => {
          const active = sidebarOpen && activeTab === tab
          return (
            <button
              key={tab}
              type="button"
              title={title}
              onClick={() => onTabClick(tab)}
              className={cn(
                'relative flex items-center justify-center w-11 h-9 transition-colors',
                active
                  ? activeColor
                  : 'text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-muted)]',
              )}
            >
              {active && (
                <span className="absolute left-0 inset-y-1.5 w-[2px] rounded-r-full bg-current opacity-80" />
              )}
              <Icon size={16} />
            </button>
          )
        })}
      </nav>

      {/* Backlog — coming soon */}
      <div className="flex flex-col py-1 border-t border-[var(--color-border)]">
        <button
          type="button"
          title="Simulations — coming soon"
          disabled
          className="flex items-center justify-center w-11 h-9 text-[var(--color-border)] cursor-default"
        >
          <Server size={16} />
        </button>
        <button
          type="button"
          title="Analytics — coming soon"
          disabled
          className="flex items-center justify-center w-11 h-9 text-[var(--color-border)] cursor-default"
        >
          <BarChart2 size={16} />
        </button>
      </div>

      {/* Bottom: settings + help */}
      <div className="mt-auto flex flex-col py-1 border-t border-[var(--color-border)]">
        {demoRunning && (
          <div className="flex justify-center py-1">
            <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse" />
          </div>
        )}
        <button
          type="button"
          title="Settings"
          onClick={onShowSettings}
          className="flex items-center justify-center w-11 h-9 text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-muted)] transition-colors"
        >
          <Settings size={16} />
        </button>
        <button
          type="button"
          title="Help"
          onClick={onShowHelp}
          className="flex items-center justify-center w-11 h-9 text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-muted)] transition-colors"
        >
          <HelpCircle size={16} />
        </button>
      </div>
    </div>
  )
}
