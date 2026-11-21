import { describe, it, expect } from 'vitest'
import { iconForTab, isRailIcon, recordTab, tabToRestore } from './railIcons'

describe('iconForTab', () => {
  it('maps every tab kind to exactly one icon', () => {
    expect(iconForTab('request')).toBe('explorer')
    expect(iconForTab(undefined)).toBe('explorer')
    expect(iconForTab('scenario')).toBe('scenarios')
    expect(iconForTab('runner')).toBe('scenarios')
    expect(iconForTab('coverage')).toBe('coverage')
    expect(iconForTab('workspace')).toBe('workspace')
    expect(iconForTab('conformance')).toBe('conformance')
    expect(iconForTab('cli')).toBe('cli')
  })
  it('recognises icons', () => {
    expect(isRailIcon('cli')).toBe(true)
    expect(isRailIcon('history')).toBe(false)
  })
})

describe('recordTab', () => {
  it('stores the active tab under its icon and keeps the others', () => {
    const a = recordTab({}, { id: 't1', kind: 'request' })
    const b = recordTab(a, { id: 's1', kind: 'scenario' })
    expect(b).toEqual({ explorer: 't1', scenarios: 's1' })
    expect(recordTab(b, { id: 't2', kind: 'request' })).toEqual({ explorer: 't2', scenarios: 's1' })
  })
  it('returns the same object when nothing changes', () => {
    const a = { explorer: 't1' }
    expect(recordTab(a, { id: 't1', kind: 'request' })).toBe(a)
    expect(recordTab(a, undefined)).toBe(a)
  })
})

describe('tabToRestore', () => {
  const tabs = [
    { id: 'w', kind: 'workspace' }, { id: 't1', kind: 'request' }, { id: 't2', kind: 'request' }, { id: 's1', kind: 'scenario' },
  ]
  it('restores the remembered tab while it is open', () => {
    expect(tabToRestore({ explorer: 't1' }, 'explorer', tabs)).toBe('t1')
  })
  it('falls back to the last open tab of the icon when the remembered one was closed', () => {
    expect(tabToRestore({ explorer: 'gone' }, 'explorer', tabs)).toBe('t2')
    expect(tabToRestore({}, 'scenarios', tabs)).toBe('s1')
  })
  it('returns nothing for an icon with no open tab', () => {
    expect(tabToRestore({ coverage: 'c1' }, 'coverage', tabs)).toBeUndefined()
    expect(tabToRestore({}, 'cli', tabs)).toBeUndefined()
  })
})
