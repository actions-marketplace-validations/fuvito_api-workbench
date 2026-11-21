export type RailIcon = 'workspace' | 'explorer' | 'scenarios' | 'coverage' | 'conformance' | 'cli'

export const RAIL_ICONS: RailIcon[] = ['workspace', 'explorer', 'scenarios', 'coverage', 'conformance', 'cli']

// The icons that are one page: clicking one with no tab open opens the page. The others (Requests,
// Scenarios, Contracts) need the user to pick something in the sidebar.
export const SINGLETON_ICONS: RailIcon[] = ['workspace', 'conformance', 'cli']

export type LastTabs = Partial<Record<RailIcon, string>>

// Every tab belongs to exactly one rail icon.
export function iconForTab(kind: string | undefined): RailIcon {
  switch (kind) {
    case 'scenario':
    case 'runner':
      return 'scenarios'
    case 'coverage':
      return 'coverage'
    case 'workspace':
      return 'workspace'
    case 'conformance':
      return 'conformance'
    case 'cli':
      return 'cli'
    default:
      return 'explorer'
  }
}

export function isRailIcon(v: unknown): v is RailIcon {
  return typeof v === 'string' && (RAIL_ICONS as string[]).includes(v)
}

// Remember the tab that is now active as its icon's last tab.
export function recordTab(last: LastTabs, tab: { id: string; kind?: string } | undefined): LastTabs {
  if (!tab) return last
  const icon = iconForTab(tab.kind)
  return last[icon] === tab.id ? last : { ...last, [icon]: tab.id }
}

// The tab to show when an icon is clicked: its remembered tab if that is still open, else the
// last open tab of that icon (a closed last tab falls back to the nearest one left), else none.
export function tabToRestore(last: LastTabs, icon: RailIcon, tabs: { id: string; kind?: string }[]): string | undefined {
  const remembered = last[icon]
  if (remembered && tabs.some(t => t.id === remembered)) return remembered
  const ofIcon = tabs.filter(t => iconForTab(t.kind) === icon)
  return ofIcon.length > 0 ? ofIcon[ofIcon.length - 1].id : undefined
}
