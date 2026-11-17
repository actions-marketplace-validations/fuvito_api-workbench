import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent, within } from '@testing-library/react'
import { WorkspaceScenarioSection } from './WorkspaceScenarioSection'
import type { ScenarioInfo } from '@/lib/bridge'

const scenarios: ScenarioInfo[] = [
  { title: 'Root smoke', path: '/ws/scenarios/smoke.yaml', stepCount: 3 },
  { title: 'Login', path: '/ws/scenarios/auth/login.yaml', folder: 'auth', stepCount: 2 },
  { title: 'Refresh', path: '/ws/scenarios/auth/tokens/refresh.yaml', folder: 'auth/tokens' },
  { title: '', path: '', folder: 'auth', isFolder: true },
  { title: '', path: '', folder: 'empty', isFolder: true },
]

function setup(over: Partial<React.ComponentProps<typeof WorkspaceScenarioSection>> = {}) {
  const props: React.ComponentProps<typeof WorkspaceScenarioSection> = {
    scenarios,
    onOpenTemplateGallery: vi.fn(),
    renamingScenarioPath: null,
    onRenamingChange: vi.fn(),
    renamingFolderPath: null,
    onRenamingFolderChange: vi.fn(),
    onScenarioContextMenu: vi.fn(),
    ...over,
  }
  const utils = render(<WorkspaceScenarioSection {...props} />)
  return { props, ...utils }
}

function inlineInput() {
  return screen.getAllByRole('textbox').find(i => !i.getAttribute('placeholder')) as HTMLInputElement
}

function queryInline() {
  return screen.queryAllByRole('textbox').find(i => !i.getAttribute('placeholder')) ?? null
}

function folderRow(name: string) {
  return screen.getByText(name).closest('div[class*="group"]') as HTMLElement
}

function dnd(data: string) {
  return { dataTransfer: { getData: () => data, setData: vi.fn(), dropEffect: '', effectAllowed: '' } }
}

describe('WorkspaceScenarioSection — tree', () => {
  it('builds nested folders from scenario paths and folder sentinels', () => {
    setup()
    expect(screen.getByText('scenarios')).toBeInTheDocument()
    expect(screen.getByText('auth')).toBeInTheDocument()
    expect(screen.getByText('tokens')).toBeInTheDocument()
    expect(screen.getByText('empty')).toBeInTheDocument()
    expect(screen.getByText('Root smoke')).toBeInTheDocument()
    expect(screen.getByText('Login')).toBeInTheDocument()
    expect(screen.getByText('Refresh')).toBeInTheDocument()
  })

  it('creates parent folders when only a nested scenario implies them', () => {
    setup({ scenarios: [{ title: 'Deep', path: '/p', folder: 'a/b/c' }] })
    expect(screen.getByText('a')).toBeInTheDocument()
    expect(screen.getByText('b')).toBeInTheDocument()
    expect(screen.getByText('c')).toBeInTheDocument()
  })

  it('shows step counts only when non-zero', () => {
    setup()
    expect(screen.getByText('3')).toBeInTheDocument()
    expect(screen.getByText('2')).toBeInTheDocument()
    expect(screen.queryByText('0')).not.toBeInTheDocument()
  })

  it('collapses and expands the root and individual folders', () => {
    setup()
    fireEvent.click(folderRow('auth'))
    expect(screen.queryByText('Login')).not.toBeInTheDocument()
    expect(screen.getByText('Root smoke')).toBeInTheDocument()
    fireEvent.click(folderRow('auth'))
    expect(screen.getByText('Login')).toBeInTheDocument()

    fireEvent.click(folderRow('scenarios'))
    expect(screen.queryByText('Root smoke')).not.toBeInTheDocument()
    fireEvent.click(folderRow('scenarios'))
    expect(screen.getByText('Root smoke')).toBeInTheDocument()
  })

  it('shows an empty state with actions when there are no scenarios', () => {
    const onNewScenario = vi.fn()
    const { props } = setup({ scenarios: [], onNewScenario })
    expect(screen.getByText('No scenarios yet')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Browse templates' }))
    expect(props.onOpenTemplateGallery).toHaveBeenCalledTimes(1)
    fireEvent.click(screen.getByText('New scenario'))
    expect(screen.queryByText('No scenarios yet')).not.toBeInTheDocument()
  })

  it('renders nothing when there are no scenarios and no create handler', () => {
    const { container } = setup({ scenarios: [] })
    expect(container).toBeEmptyDOMElement()
  })

  it('shows the scenario error with a tooltip', () => {
    setup({ scenarioError: 'cannot read dir' })
    expect(screen.getByText('cannot read dir')).toHaveAttribute('title', 'cannot read dir')
  })
})

describe('WorkspaceScenarioSection — selecting and row actions', () => {
  it('opens a scenario on click', () => {
    const onSelectScenario = vi.fn()
    setup({ onSelectScenario })
    fireEvent.click(screen.getByText('Login'))
    expect(onSelectScenario).toHaveBeenCalledWith('/ws/scenarios/auth/login.yaml', 'Login')
  })

  it('opens the scenario context menu without selecting', () => {
    const onSelectScenario = vi.fn()
    const { props } = setup({ onSelectScenario })
    fireEvent.contextMenu(screen.getByText('Login'))
    expect(props.onScenarioContextMenu).toHaveBeenCalledWith(expect.anything(), '/ws/scenarios/auth/login.yaml', 'Login')
    expect(onSelectScenario).not.toHaveBeenCalled()
  })

  it('starts a rename by double-click', () => {
    const { props } = setup()
    fireEvent.doubleClick(screen.getByText('Login'))
    expect(props.onRenamingChange).toHaveBeenCalledWith('/ws/scenarios/auth/login.yaml')
  })

  it('renders an inline rename input and commits the new name', () => {
    const onRenameScenario = vi.fn()
    const { props } = setup({ renamingScenarioPath: '/ws/scenarios/smoke.yaml', onRenameScenario })
    const input = inlineInput()
    expect(input).toHaveValue('Root smoke')
    fireEvent.change(input, { target: { value: 'Renamed' } })
    fireEvent.keyDown(input, { key: 'Enter' })
    expect(props.onRenamingChange).toHaveBeenCalledWith(null)
    expect(onRenameScenario).toHaveBeenCalledWith('/ws/scenarios/smoke.yaml', 'Renamed')
  })

  it('cancelling a rename clears the renaming state', () => {
    const { props } = setup({ renamingScenarioPath: '/ws/scenarios/smoke.yaml' })
    fireEvent.keyDown(inlineInput(), { key: 'Escape' })
    expect(props.onRenamingChange).toHaveBeenCalledWith(null)
  })

  it('duplicates and deletes without selecting', () => {
    const onDuplicateScenario = vi.fn(), onDeleteScenario = vi.fn(), onSelectScenario = vi.fn()
    setup({ onDuplicateScenario, onDeleteScenario, onSelectScenario })
    fireEvent.click(screen.getAllByTitle('Duplicate scenario')[0])
    fireEvent.click(screen.getAllByTitle('Delete scenario')[0])
    expect(onDuplicateScenario).toHaveBeenCalledTimes(1)
    expect(onDeleteScenario).toHaveBeenCalledTimes(1)
    expect(onSelectScenario).not.toHaveBeenCalled()
  })

  it('omits row actions without handlers and hides them while renaming', () => {
    const { unmount } = setup()
    expect(screen.queryByTitle('Duplicate scenario')).not.toBeInTheDocument()
    unmount()
    setup({ onDeleteScenario: vi.fn(), renamingScenarioPath: '/ws/scenarios/smoke.yaml' })
    expect(screen.queryByTitle('Delete scenario')).not.toBeInTheDocument()
  })

  it('dims a scenario that has been cut', () => {
    setup({ clipboard: { op: 'cut', kind: 'scenario', srcPath: '/ws/scenarios/smoke.yaml' } })
    expect(screen.getByText('Root smoke').closest('[draggable]')!.className).toContain('opacity-40')
  })

  it('does not dim copied scenarios', () => {
    setup({ clipboard: { op: 'copy', kind: 'scenario', srcPath: '/ws/scenarios/smoke.yaml' } })
    expect(screen.getByText('Root smoke').closest('[draggable]')!.className).not.toContain('opacity-40')
  })

  it('sets a drag payload when a scenario is dragged', () => {
    setup()
    const setData = vi.fn()
    fireEvent.dragStart(screen.getByText('Root smoke').closest('[draggable]')!, { dataTransfer: { setData, effectAllowed: '' } })
    expect(setData).toHaveBeenCalledWith('application/x-clrsigscenario', '/ws/scenarios/smoke.yaml')
  })
})

describe('WorkspaceScenarioSection — filtering', () => {
  it('only shows the filter box when real scenarios exist', () => {
    const { unmount } = setup({ scenarios: [{ title: '', path: '', folder: 'x', isFolder: true }], onNewScenario: vi.fn() })
    expect(screen.queryByPlaceholderText('Filter scenarios…')).not.toBeInTheDocument()
    unmount()
    setup()
    expect(screen.getByPlaceholderText('Filter scenarios…')).toBeInTheDocument()
  })

  it('filters case-insensitively and flattens the tree', () => {
    setup()
    fireEvent.change(screen.getByPlaceholderText('Filter scenarios…'), { target: { value: 'LOG' } })
    expect(screen.getByText('Login')).toBeInTheDocument()
    expect(screen.queryByText('Root smoke')).not.toBeInTheDocument()
    expect(screen.queryByText('auth')).not.toBeInTheDocument()
  })

  it('reports no matches and clears the filter', () => {
    setup()
    const input = screen.getByPlaceholderText('Filter scenarios…')
    fireEvent.change(input, { target: { value: 'zzz' } })
    expect(screen.getByText('No matches.')).toBeInTheDocument()
    fireEvent.click(input.parentElement!.querySelector('button')!)
    expect(screen.queryByText('No matches.')).not.toBeInTheDocument()
    expect(screen.getByText('Root smoke')).toBeInTheDocument()
  })
})

describe('WorkspaceScenarioSection — root toolbar', () => {
  it('runs all scenarios and disables the button while running', () => {
    const onRunAllScenarios = vi.fn()
    const { unmount } = setup({ onRunAllScenarios })
    fireEvent.click(screen.getByTitle('Run all scenarios'))
    expect(onRunAllScenarios).toHaveBeenCalledTimes(1)
    unmount()
    setup({ onRunAllScenarios, runAllState: { running: true } as never })
    expect(screen.getByTitle('Run all scenarios')).toBeDisabled()
  })

  it('hides run-all when there are no real scenarios', () => {
    setup({ scenarios: [{ title: '', path: '', folder: 'x', isFolder: true }], onRunAllScenarios: vi.fn(), onNewScenario: vi.fn() })
    expect(screen.queryByTitle('Run all scenarios')).not.toBeInTheDocument()
  })

  it('refreshes and opens the template gallery', () => {
    const onRefresh = vi.fn()
    const { props } = setup({ onRefresh })
    fireEvent.click(screen.getByTitle('Refresh scenarios'))
    fireEvent.click(screen.getByTitle('New scenario from template'))
    expect(onRefresh).toHaveBeenCalledTimes(1)
    expect(props.onOpenTemplateGallery).toHaveBeenCalledTimes(1)
  })

  it('creates a root scenario inline', () => {
    const onNewScenario = vi.fn()
    setup({ onNewScenario })
    fireEvent.click(screen.getByTitle('New scenario'))
    const input = inlineInput()
    fireEvent.change(input, { target: { value: 'Fresh' } })
    fireEvent.keyDown(input, { key: 'Enter' })
    expect(onNewScenario).toHaveBeenCalledWith('Fresh')
    expect(queryInline()).not.toBeInTheDocument()
  })

  it('creates a root folder inline', () => {
    const onNewFolder = vi.fn()
    setup({ onNewFolder })
    fireEvent.click(screen.getByTitle('New folder'))
    const input = inlineInput()
    fireEvent.change(input, { target: { value: 'group' } })
    fireEvent.keyDown(input, { key: 'Enter' })
    expect(onNewFolder).toHaveBeenCalledWith('', 'group')
  })

  it('cancelling inline creation removes the input', () => {
    setup({ onNewScenario: vi.fn() })
    fireEvent.click(screen.getByTitle('New scenario'))
    fireEvent.keyDown(inlineInput(), { key: 'Escape' })
    expect(queryInline()).not.toBeInTheDocument()
  })

  it('re-opens a collapsed root when creating', () => {
    setup({ onNewScenario: vi.fn(), onNewFolder: vi.fn() })
    fireEvent.click(folderRow('scenarios'))
    expect(screen.queryByText('Root smoke')).not.toBeInTheDocument()
    fireEvent.click(screen.getByTitle('New scenario'))
    expect(screen.getByText('Root smoke')).toBeInTheDocument()
    fireEvent.keyDown(inlineInput(), { key: 'Escape' })

    fireEvent.click(folderRow('scenarios'))
    fireEvent.click(screen.getByTitle('New folder'))
    expect(screen.getByText('Root smoke')).toBeInTheDocument()
  })

  it('opens the folder context menu for the root with an empty path', () => {
    const onFolderContextMenu = vi.fn()
    setup({ onFolderContextMenu })
    fireEvent.contextMenu(screen.getByText('scenarios'))
    expect(onFolderContextMenu).toHaveBeenCalledWith(expect.anything(), '', 'scenarios')
  })

  it('tolerates a root context menu without a handler', () => {
    setup()
    expect(() => fireEvent.contextMenu(screen.getByText('scenarios'))).not.toThrow()
  })
})

describe('WorkspaceScenarioSection — folder actions', () => {
  it('runs, deletes and opens the context menu for a folder', () => {
    const onRunFolder = vi.fn(), onDeleteFolder = vi.fn(), onFolderContextMenu = vi.fn()
    setup({ onRunFolder, onDeleteFolder, onFolderContextMenu })
    const row = folderRow('auth')
    fireEvent.click(within(row).getByTitle('Run all scenarios in folder'))
    fireEvent.click(within(row).getByTitle('Delete folder'))
    fireEvent.contextMenu(screen.getByText('auth'))
    expect(onRunFolder).toHaveBeenCalledWith('auth')
    expect(onDeleteFolder).toHaveBeenCalledWith('auth')
    expect(onFolderContextMenu).toHaveBeenCalledWith(expect.anything(), 'auth', 'auth')
    expect(screen.getByText('Login')).toBeInTheDocument()
  })

  it('starts renaming a folder by double-click and commits', () => {
    const onRenameFolder = vi.fn()
    const { props, rerender } = setup({ onRenameFolder })
    fireEvent.doubleClick(screen.getByText('auth'))
    expect(props.onRenamingFolderChange).toHaveBeenCalledWith('auth')
    rerender(<WorkspaceScenarioSection {...props} renamingFolderPath="auth" onRenameFolder={onRenameFolder} />)
    const input = inlineInput()
    fireEvent.change(input, { target: { value: 'security' } })
    fireEvent.keyDown(input, { key: 'Enter' })
    expect(onRenameFolder).toHaveBeenCalledWith('auth', 'security')
  })

  it('cancels a folder rename', () => {
    const { props } = setup({ renamingFolderPath: 'auth' })
    fireEvent.keyDown(inlineInput(), { key: 'Escape' })
    expect(props.onRenamingFolderChange).toHaveBeenCalledWith(null)
  })

  it('creates a scenario inside a folder', () => {
    const onNewScenario = vi.fn()
    setup({ onNewScenario })
    fireEvent.click(within(folderRow('empty')).getByTitle('New scenario in folder'))
    const input = inlineInput()
    fireEvent.change(input, { target: { value: 'In empty' } })
    fireEvent.keyDown(input, { key: 'Enter' })
    expect(onNewScenario).toHaveBeenCalledWith('In empty', 'empty')
  })

  it('creates a subfolder inside a folder', () => {
    const onNewFolder = vi.fn()
    setup({ onNewFolder })
    fireEvent.click(within(folderRow('empty')).getByTitle('New subfolder'))
    const input = inlineInput()
    fireEvent.change(input, { target: { value: 'child' } })
    fireEvent.keyDown(input, { key: 'Enter' })
    expect(onNewFolder).toHaveBeenCalledWith('empty', 'child')
  })

  it('cancelling folder-scoped creation hides the inputs', () => {
    setup()
    fireEvent.click(within(folderRow('empty')).getByTitle('New scenario in folder'))
    fireEvent.keyDown(inlineInput(), { key: 'Escape' })
    fireEvent.click(within(folderRow('empty')).getByTitle('New subfolder'))
    fireEvent.keyDown(inlineInput(), { key: 'Escape' })
    expect(queryInline()).not.toBeInTheDocument()
  })

  it('expands a collapsed folder when creating inside it', () => {
    setup({ onNewScenario: vi.fn() })
    fireEvent.click(folderRow('auth'))
    expect(screen.queryByText('Login')).not.toBeInTheDocument()
    fireEvent.click(within(folderRow('auth')).getByTitle('New scenario in folder'))
    expect(screen.getByText('Login')).toBeInTheDocument()
  })

  it('hides optional folder actions without handlers', () => {
    setup()
    expect(screen.queryByTitle('Run all scenarios in folder')).not.toBeInTheDocument()
    expect(screen.queryByTitle('Delete folder')).not.toBeInTheDocument()
  })
})

describe('WorkspaceScenarioSection — drag and drop', () => {
  it('moves a dropped scenario into a folder', () => {
    const onMoveScenario = vi.fn()
    setup({ onMoveScenario })
    const row = folderRow('empty')
    fireEvent.dragOver(row, dnd(''))
    expect(row.className).toContain('bg-[var(--color-accent)]/10')
    fireEvent.drop(row, dnd('/ws/scenarios/smoke.yaml'))
    expect(onMoveScenario).toHaveBeenCalledWith('/ws/scenarios/smoke.yaml', 'empty')
    expect(row.className).not.toContain('bg-[var(--color-accent)]/10')
  })

  it('moves a dropped scenario to the root', () => {
    const onMoveScenario = vi.fn()
    setup({ onMoveScenario })
    const row = folderRow('scenarios')
    fireEvent.dragOver(row, dnd(''))
    fireEvent.drop(row, dnd('/ws/scenarios/auth/login.yaml'))
    expect(onMoveScenario).toHaveBeenCalledWith('/ws/scenarios/auth/login.yaml', '')
  })

  it('ignores drops without a payload and clears highlight on drag leave', () => {
    const onMoveScenario = vi.fn()
    setup({ onMoveScenario })
    const row = folderRow('empty')
    fireEvent.drop(row, dnd(''))
    expect(onMoveScenario).not.toHaveBeenCalled()
    fireEvent.dragOver(row, dnd(''))
    fireEvent.dragLeave(row, { relatedTarget: document.body })
    expect(row.className).not.toContain('bg-[var(--color-accent)]/10')

    const root = folderRow('scenarios')
    fireEvent.dragOver(root, dnd(''))
    expect(root.className).toContain('bg-[var(--color-accent)]/10')
    fireEvent.dragLeave(root, { relatedTarget: document.body })
    fireEvent.drop(root, dnd(''))
    expect(onMoveScenario).not.toHaveBeenCalled()
  })

  it('tolerates drops without a move handler', () => {
    setup()
    expect(() => fireEvent.drop(folderRow('empty'), dnd('/x'))).not.toThrow()
  })
})
