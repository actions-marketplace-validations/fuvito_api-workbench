import { describe, it, expect } from 'vitest'
import { railTargetForTab } from './railFocus'

describe('railTargetForTab', () => {
  it('maps tabs to the sidebar panel they belong to', () => {
    expect(railTargetForTab('request')).toEqual({ panel: 'explorer' })
    expect(railTargetForTab(undefined)).toEqual({ panel: 'explorer' })
    expect(railTargetForTab('scenario')).toEqual({ panel: 'scenarios' })
    expect(railTargetForTab('runner')).toEqual({ panel: 'scenarios' })
    expect(railTargetForTab('coverage')).toEqual({ panel: 'coverage' })
  })

  it('maps full-width page tabs to their own icon and no panel', () => {
    expect(railTargetForTab('workspace')).toEqual({ page: 'workspace' })
    expect(railTargetForTab('conformance')).toEqual({ page: 'conformance' })
    expect(railTargetForTab('cli')).toEqual({ page: 'cli' })
  })
})
