import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { AppToolbar } from './AppToolbar'
import { createBlankTab, type TabState } from '@/lib/tabState'

vi.mock('@/components/SnippetButton', () => ({ SnippetButton: () => <button>snippet</button> }))

function tab(over: Partial<TabState> = {}): TabState {
  return { ...createBlankTab(), id: 't1', ...over }
}

function setup(activeTab: TabState) {
  const props = {
    activeTab,
    activeTabId: activeTab.id,
    updateTab: vi.fn(),
    onOpenRequest: vi.fn(),
    onSaveRequest: vi.fn(),
    onNewRequest: vi.fn(),
    onDuplicateTab: vi.fn(),
  }
  render(<AppToolbar {...props} />)
  return props
}

describe('AppToolbar — request tab', () => {
  it('invites naming an unnamed request and starts editing on click', () => {
    const p = setup(tab({ name: null }))
    fireEvent.click(screen.getByTitle('Click to name this request'))
    expect(p.updateTab).toHaveBeenCalledWith('t1', { name: '', editingName: true })
  })

  it('shows the name and starts renaming on click', () => {
    const p = setup(tab({ name: 'Get users', method: 'GET' }))
    expect(screen.getByText('GET')).toBeInTheDocument()
    fireEvent.click(screen.getByTitle('Click to rename'))
    expect(p.updateTab).toHaveBeenCalledWith('t1', { editingName: true, saveError: null })
  })

  it('hides the method for an unnamed request', () => {
    setup(tab({ name: null, method: 'POST' }))
    expect(screen.queryByText('POST')).not.toBeInTheDocument()
  })

  it('edits the name inline', () => {
    const p = setup(tab({ name: 'abc', editingName: true }))
    fireEvent.change(screen.getByDisplayValue('abc'), { target: { value: 'abcd' } })
    expect(p.updateTab).toHaveBeenCalledWith('t1', { name: 'abcd' })
  })

  it.each(['Enter', 'Escape'])('commits a non-empty name on %s', key => {
    const p = setup(tab({ name: 'abc', editingName: true }))
    fireEvent.keyDown(screen.getByDisplayValue('abc'), { key })
    expect(p.updateTab).toHaveBeenCalledWith('t1', { editingName: false })
  })

  it('clears a blank name on Enter and blur', () => {
    const p = setup(tab({ name: '   ', editingName: true }))
    const input = screen.getByDisplayValue('')
    fireEvent.keyDown(input, { key: 'Enter' })
    fireEvent.blur(input)
    expect(p.updateTab).toHaveBeenNthCalledWith(1, 't1', { name: null, editingName: false })
    expect(p.updateTab).toHaveBeenNthCalledWith(2, 't1', { name: null, editingName: false })
  })

  it('commits a non-empty name on blur and ignores other keys', () => {
    const p = setup(tab({ name: 'abc', editingName: true }))
    const input = screen.getByDisplayValue('abc')
    fireEvent.keyDown(input, { key: 'a' })
    expect(p.updateTab).not.toHaveBeenCalled()
    fireEvent.blur(input)
    expect(p.updateTab).toHaveBeenCalledWith('t1', { editingName: false })
  })

  it('shows a save error', () => {
    setup(tab({ name: 'x', saveError: 'disk full' }))
    expect(screen.getByText('disk full')).toBeInTheDocument()
  })

  it('shows the unsaved dot only for dirty tabs', () => {
    const clean = createBlankTab()
    const { unmount } = render(<AppToolbar activeTab={clean} activeTabId={clean.id} updateTab={vi.fn()} onOpenRequest={vi.fn()} onSaveRequest={vi.fn()} onNewRequest={vi.fn()} onDuplicateTab={vi.fn()} />)
    expect(screen.queryByTitle('Unsaved changes')).not.toBeInTheDocument()
    unmount()
    setup(tab({ url: 'https://changed.example', savedKey: 'different' }))
    expect(screen.getByTitle('Unsaved changes')).toBeInTheDocument()
  })

  it('wires the file actions', () => {
    const p = setup(tab())
    fireEvent.click(screen.getByTitle('Open request file'))
    fireEvent.click(screen.getByTitle('Save request (Ctrl+S)'))
    fireEvent.click(screen.getByTitle('New request tab (Ctrl+N)'))
    fireEvent.click(screen.getByTitle('Duplicate tab'))
    expect(p.onOpenRequest).toHaveBeenCalledTimes(1)
    expect(p.onSaveRequest).toHaveBeenCalledTimes(1)
    expect(p.onNewRequest).toHaveBeenCalledTimes(1)
    expect(p.onDuplicateTab).toHaveBeenCalledTimes(1)
    expect(screen.getByText('snippet')).toBeInTheDocument()
  })
})

describe('AppToolbar — non-request tabs', () => {
  it('shows only the scenario name and no file actions', () => {
    setup(tab({ kind: 'scenario', name: 'Login flow' }))
    expect(screen.getByText('Login flow')).toBeInTheDocument()
    expect(screen.queryByTitle('Save request (Ctrl+S)')).not.toBeInTheDocument()
    expect(screen.queryByText('snippet')).not.toBeInTheDocument()
  })

  it('shows the scenario file path, not the name the tab and title already show', () => {
    setup(tab({ kind: 'scenario', name: 'Login flow', scenarioPath: 'C:\\ws\\scenarios\\login.yaml' }))
    expect(screen.getByText('C:/ws/scenarios/login.yaml')).toBeInTheDocument()
    expect(screen.queryByText('Login flow')).not.toBeInTheDocument()
  })

  it('falls back to a generic scenario label', () => {
    setup(tab({ kind: 'scenario', name: null }))
    expect(screen.getByText('Scenario')).toBeInTheDocument()
  })

  it('hides file actions on the runner tab', () => {
    setup(tab({ kind: 'runner' }))
    expect(screen.queryByTitle('Open request file')).not.toBeInTheDocument()
    expect(screen.queryByTitle('Duplicate tab')).not.toBeInTheDocument()
    expect(screen.queryByText('snippet')).not.toBeInTheDocument()
  })
})
