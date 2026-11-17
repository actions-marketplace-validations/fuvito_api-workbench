import { describe, it, expect, vi, beforeAll } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { TabBar } from './TabBar'
import type { TabInfo } from './TabBar'

beforeAll(() => {
  globalThis.ResizeObserver = class { observe() {} unobserve() {} disconnect() {} } as never
  Element.prototype.scrollTo = vi.fn() as never
})

const tabs: TabInfo[] = [
  { id: 'a', name: 'Get users', method: 'GET', isDirty: false, filePath: '/ws/a.yaml' },
  { id: 'b', name: null, method: 'DELETE', isDirty: true },
  { id: 'c', name: 'S', method: 'GET', isDirty: false, kind: 'scenario' },
]

function setup(over: Partial<React.ComponentProps<typeof TabBar>> = {}) {
  const h = {
    onSelect: vi.fn(), onClose: vi.fn(), onNew: vi.fn(), onSave: vi.fn(), onCloseOthers: vi.fn(),
    onCloseToRight: vi.fn(), onCloseAll: vi.fn(), onCopyPath: vi.fn(), onRevealInExplorer: vi.fn(),
  }
  render(<TabBar tabs={tabs} activeTabId="a" {...h} {...over} />)
  return h
}

describe('TabBar rendering', () => {
  it('shows names, abbreviated methods and fallbacks', () => {
    setup()
    expect(screen.getByText('Get users')).toBeInTheDocument()
    expect(screen.getByText('New Request')).toBeInTheDocument()
    expect(screen.getByText('GET')).toBeInTheDocument()
    expect(screen.getByText('DEL')).toBeInTheDocument()
    expect(screen.getByText('SCN')).toBeInTheDocument()
  })

  it.each([
    ['coverage', 'COV', 'Coverage'],
    ['runner', 'RUN', 'Run Results'],
    ['scenario', 'SCN', 'Scenario'],
  ] as const)('labels %s tabs with %s and a default name', (kind, badge, name) => {
    setup({ tabs: [{ id: 'x', name: null, method: 'GET', isDirty: false, kind }], activeTabId: 'x' })
    expect(screen.getByText(badge)).toBeInTheDocument()
    expect(screen.getByText(name)).toBeInTheDocument()
  })

  it('abbreviates unknown methods to three letters', () => {
    setup({ tabs: [{ id: 'x', name: 'n', method: 'PROPFIND', isDirty: false }], activeTabId: 'x' })
    expect(screen.getByText('PRO')).toBeInTheDocument()
  })

  it('marks only the active tab selected and focusable', () => {
    setup()
    const [a, b] = screen.getAllByRole('tab')
    expect(a).toHaveAttribute('aria-selected', 'true')
    expect(a).toHaveAttribute('tabindex', '0')
    expect(b).toHaveAttribute('aria-selected', 'false')
    expect(b).toHaveAttribute('tabindex', '-1')
  })

  it('flags unsaved tabs', () => {
    setup()
    expect(screen.getAllByTitle('Unsaved changes')).toHaveLength(1)
  })
})

describe('TabBar interaction', () => {
  it('selects a tab on click', () => {
    const h = setup()
    fireEvent.click(screen.getByText('New Request'))
    expect(h.onSelect).toHaveBeenCalledWith('b')
  })

  it('closes with the X button without selecting the tab', () => {
    const h = setup()
    fireEvent.click(screen.getAllByTitle('Close tab')[1])
    expect(h.onClose).toHaveBeenCalledWith('b')
    expect(h.onSelect).not.toHaveBeenCalled()
  })

  it('creates a new tab', () => {
    const h = setup()
    fireEvent.click(screen.getByTitle('New request tab (Ctrl+N)'))
    expect(h.onNew).toHaveBeenCalled()
  })

  it('arrow keys cycle through tabs and wrap around', () => {
    const h = setup()
    const list = screen.getByRole('tablist')
    fireEvent.keyDown(list, { key: 'ArrowRight' })
    expect(h.onSelect).toHaveBeenLastCalledWith('b')
    fireEvent.keyDown(list, { key: 'ArrowLeft' })
    expect(h.onSelect).toHaveBeenLastCalledWith('c')
  })

  it('wraps forward from the last tab', () => {
    const h = setup({ activeTabId: 'c' })
    fireEvent.keyDown(screen.getByRole('tablist'), { key: 'ArrowRight' })
    expect(h.onSelect).toHaveBeenLastCalledWith('a')
  })

  it('Delete and Backspace close the active tab; other keys do nothing', () => {
    const h = setup()
    const list = screen.getByRole('tablist')
    fireEvent.keyDown(list, { key: 'Delete' })
    fireEvent.keyDown(list, { key: 'Backspace' })
    fireEvent.keyDown(list, { key: 'x' })
    expect(h.onClose).toHaveBeenCalledTimes(2)
    expect(h.onClose).toHaveBeenCalledWith('a')
  })
})

describe('TabBar context menu', () => {
  const open = (label: string) => fireEvent.contextMenu(screen.getByText(label), { clientX: 10, clientY: 20 })
  const item = (label: string) => screen.getByText(label) as HTMLButtonElement

  it('opens on right click with every action', () => {
    setup()
    open('Get users')
    for (const l of ['Save', 'Copy Path', 'Reveal in Explorer', 'Close Tab', 'Close Others', 'Close to the Right', 'Close All Tabs']) {
      expect(item(l)).toBeInTheDocument()
    }
  })

  it('routes each action with the right-clicked tab id or path', () => {
    const h = setup()
    open('Get users')
    fireEvent.click(item('Save'))
    expect(h.onSave).toHaveBeenCalledWith('a')

    open('Get users')
    fireEvent.click(item('Copy Path'))
    expect(h.onCopyPath).toHaveBeenCalledWith('/ws/a.yaml')

    open('Get users')
    fireEvent.click(item('Reveal in Explorer'))
    expect(h.onRevealInExplorer).toHaveBeenCalledWith('/ws/a.yaml')

    open('Get users')
    fireEvent.click(item('Close Others'))
    expect(h.onCloseOthers).toHaveBeenCalledWith('a')

    open('Get users')
    fireEvent.click(item('Close to the Right'))
    expect(h.onCloseToRight).toHaveBeenCalledWith('a')

    open('Get users')
    fireEvent.click(item('Close All Tabs'))
    expect(h.onCloseAll).toHaveBeenCalled()

    open('Get users')
    fireEvent.click(item('Close Tab'))
    expect(h.onClose).toHaveBeenCalledWith('a')
  })

  it('disables path actions for unsaved tabs', () => {
    setup()
    open('New Request')
    expect(item('Copy Path')).toBeDisabled()
    expect(item('Reveal in Explorer')).toBeDisabled()
    expect(item('Save')).not.toBeDisabled()
  })

  it('disables Save for scenario and coverage tabs', () => {
    setup()
    open('S')
    expect(item('Save')).toBeDisabled()
  })

  it('disables "Close to the Right" on the last tab', () => {
    setup()
    open('S')
    expect(item('Close to the Right')).toBeDisabled()
  })

  it('disables Close Others when it is the only tab', () => {
    setup({ tabs: [tabs[0]] })
    open('Get users')
    expect(item('Close Others')).toBeDisabled()
  })

  it('disables bulk actions whose handlers are missing', () => {
    setup({ onSave: undefined, onCloseOthers: undefined, onCloseToRight: undefined, onCloseAll: undefined })
    open('Get users')
    for (const l of ['Save', 'Close Others', 'Close to the Right', 'Close All Tabs']) expect(item(l)).toBeDisabled()
  })

  it('closes the menu after choosing an action', () => {
    setup()
    open('Get users')
    fireEvent.click(item('Close Tab'))
    expect(screen.queryByText('Close All Tabs')).not.toBeInTheDocument()
  })
})
