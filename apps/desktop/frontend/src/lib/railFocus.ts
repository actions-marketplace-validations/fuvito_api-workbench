import type { RailTab } from '@/components/ActivityRail'

export type RailPage = 'workspace' | 'conformance' | 'cli'

// Which rail icon belongs to a tab. A tab either belongs to a sidebar panel (Requests, Scenarios,
// Contracts), or is a full-width page with its own icon and no sidebar.
export function railTargetForTab(kind: string | undefined): { panel?: RailTab; page?: RailPage } {
  switch (kind) {
    case 'scenario':
    case 'runner':
      return { panel: 'scenarios' }
    case 'coverage':
      return { panel: 'coverage' }
    case 'workspace':
      return { page: 'workspace' }
    case 'conformance':
      return { page: 'conformance' }
    case 'cli':
      return { page: 'cli' }
    default:
      return { panel: 'explorer' }
  }
}
