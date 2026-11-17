import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent, act } from '@testing-library/react'
import { WorkspacePanel } from './WorkspacePanel'
import * as bridge from '@/lib/bridge'
import type { FileNode } from '@/lib/bridge'

vi.mock('@/lib/bridge', () => ({ openInExplorer: vi.fn() }))

const b = vi.mocked(bridge)

const tree: FileNode[] = [
  {
    name: 'requests', path: '/ws/requests', isDir: true,
    children: [
      { name: 'users', path: '/ws/requests/users', isDir: true, children: [{ name: 'get-user.yaml', path: '/ws/requests/users/get-user.yaml', isDir: false, method: 'GET' }] },
      { name: 'empty', path: '/ws/requests/empty', isDir: true, children: [] },
      { name: 'health.yml', path: '/ws/requests/health.yml', isDir: false, method: 'GET' },
    ],
  },
]

function setup(over: Partial<React.ComponentProps<typeof WorkspacePanel>> = {}) {
  const props: React.ComponentProps<typeof WorkspacePanel> = {
    workspacePath: '/ws',
    fileTree: tree,
    onOpenWorkspace: vi.fn(),
    onNewWorkspace: vi.fn(),
    onSelectFile: vi.fn(),
    onImportOpenAPI: vi.fn(),
    ...over,
  }
  render(<WorkspacePanel {...props} />)
  return props
}

beforeEach(() => { vi.clearAllMocks(); b.openInExplorer.mockResolvedValue('' as never) })
afterEach(() => vi.useRealTimers())

describe('WorkspacePanel — no workspace', () => {
  it('shows the empty state and wires its actions', () => {
    const p = setup({ workspacePath: '', fileTree: [], onTryDemo: vi.fn() })
    fireEvent.click(screen.getByRole('button', { name: 'New Workspace' }))
    fireEvent.click(screen.getByRole('button', { name: 'Open Existing' }))
    fireEvent.click(screen.getByRole('button', { name: /Try the Demo/ }))
    expect(p.onNewWorkspace).toHaveBeenCalledTimes(1)
    expect(p.onOpenWorkspace).toHaveBeenCalledTimes(1)
    expect(p.onTryDemo).toHaveBeenCalledTimes(1)
    expect(screen.queryByPlaceholderText('Filter requests…')).not.toBeInTheDocument()
  })
})

describe('WorkspacePanel — empty workspace', () => {
  it('offers import and a new request when there are no files', () => {
    const onNewRequestInFolder = vi.fn()
    const p = setup({ fileTree: [], onNewRequestInFolder })
    expect(screen.getByText('No request files yet')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Import OpenAPI spec' }))
    fireEvent.click(screen.getByRole('button', { name: 'New request' }))
    expect(p.onImportOpenAPI).toHaveBeenCalledTimes(1)
    expect(onNewRequestInFolder).toHaveBeenCalledWith('')
    expect(screen.queryByPlaceholderText('Filter requests…')).not.toBeInTheDocument()
  })

  it('tolerates New request without a handler', () => {
    setup({ fileTree: [] })
    expect(() => fireEvent.click(screen.getByRole('button', { name: 'New request' }))).not.toThrow()
  })
})

describe('WorkspacePanel — tree', () => {
  it('renders the tree and selects a file', () => {
    const p = setup()
    fireEvent.click(screen.getByText('health'))
    expect(p.onSelectFile).toHaveBeenCalledWith('/ws/requests/health.yml')
  })

  it('collapses and expands folders', () => {
    setup()
    expect(screen.getByText('get-user')).toBeInTheDocument()
    const usersToggle = screen.getByText('users').closest('div[class*="group"]')!.querySelector('button')!
    fireEvent.click(usersToggle)
    expect(screen.queryByText('get-user')).not.toBeInTheDocument()
    fireEvent.click(usersToggle)
    expect(screen.getByText('get-user')).toBeInTheDocument()
  })

  it('shows import and folder-run results', () => {
    const onDismissFolderRun = vi.fn()
    setup({
      importError: 'bad spec',
      folderRunState: { folderPath: '/ws/requests', folderName: 'requests', running: false, results: [] },
      onDismissFolderRun,
    })
    expect(screen.getByText('Import failed: bad spec')).toBeInTheDocument()
    fireEvent.click(screen.getAllByRole('button').find(b => b.querySelector('svg.lucide-x'))!)
    expect(onDismissFolderRun).toHaveBeenCalledTimes(1)
  })

  it('paste-to-root is only offered when a request is on the clipboard', () => {
    const onPasteToRoot = vi.fn()
    setup({ onPasteToRoot, clipboard: { op: 'copy', kind: 'scenario', srcPath: '/x' } })
    expect(screen.queryByTitle('Paste here (workspace root)')).not.toBeInTheDocument()
  })

  it('pastes to the root for a request clipboard', () => {
    const onPasteToRoot = vi.fn()
    setup({ onPasteToRoot, clipboard: { op: 'copy', kind: 'request', srcPath: '/x' } })
    fireEvent.click(screen.getByTitle('Paste here (workspace root)'))
    expect(onPasteToRoot).toHaveBeenCalledTimes(1)
  })
})

describe('WorkspacePanel — filter', () => {
  const input = () => screen.getByPlaceholderText('Filter requests…')

  it('filters files ignoring extension and case, keeping ancestor folders', () => {
    setup()
    fireEvent.change(input(), { target: { value: 'GET-USER' } })
    expect(screen.getByText('get-user')).toBeInTheDocument()
    expect(screen.getByText('users')).toBeInTheDocument()
    expect(screen.queryByText('health')).not.toBeInTheDocument()
    expect(screen.queryByText('empty')).not.toBeInTheDocument()
  })

  it('keeps a folder whose own name matches even with no matching children', () => {
    setup()
    fireEvent.change(input(), { target: { value: 'empty' } })
    expect(screen.getByText('empty')).toBeInTheDocument()
  })

  it('does not match against the file extension', () => {
    setup()
    fireEvent.change(input(), { target: { value: 'yaml' } })
    expect(screen.getByText('No matches.')).toBeInTheDocument()
  })

  it('clears the filter', () => {
    setup()
    fireEvent.change(input(), { target: { value: 'zzz' } })
    expect(screen.getByText('No matches.')).toBeInTheDocument()
    fireEvent.click(input().parentElement!.querySelector('button')!)
    expect(screen.queryByText('No matches.')).not.toBeInTheDocument()
    expect(screen.getByText('health')).toBeInTheDocument()
  })
})

describe('WorkspacePanel — context menu', () => {
  function rightClick(text: string) {
    fireEvent.contextMenu(screen.getByText(text), { clientX: 40, clientY: 50 })
  }

  it('opens a file menu with the supplied actions and runs one', () => {
    const p = setup({ onDuplicateFile: vi.fn(), onDeleteFile: vi.fn(), onCopyFile: vi.fn(), onCutFile: vi.fn() })
    rightClick('health')
    for (const l of ['Open', 'Duplicate', 'Copy', 'Cut', 'Rename', 'Open in Explorer', 'Delete']) {
      expect(screen.getByRole('button', { name: l })).toBeInTheDocument()
    }
    fireEvent.click(screen.getByRole('button', { name: 'Duplicate' }))
    expect(p.onDuplicateFile).toHaveBeenCalledWith('/ws/requests/health.yml')
    expect(screen.queryByRole('button', { name: 'Duplicate' })).not.toBeInTheDocument()
  })

  it('opens a file from the menu', () => {
    const p = setup()
    rightClick('health')
    fireEvent.click(screen.getByRole('button', { name: 'Open' }))
    expect(p.onSelectFile).toHaveBeenCalledWith('/ws/requests/health.yml')
  })

  it('renames a file inline from the menu', () => {
    const onRenameFile = vi.fn()
    setup({ onRenameFile })
    rightClick('health')
    fireEvent.click(screen.getByRole('button', { name: 'Rename' }))
    const input = screen.getByDisplayValue('health')
    fireEvent.change(input, { target: { value: 'status' } })
    fireEvent.keyDown(input, { key: 'Enter' })
    expect(onRenameFile).toHaveBeenCalledWith('/ws/requests/health.yml', 'status')
  })

  it('reveals a file in the system explorer', () => {
    setup()
    rightClick('health')
    fireEvent.click(screen.getByRole('button', { name: 'Open in Explorer' }))
    expect(b.openInExplorer).toHaveBeenCalledWith('/ws/requests/health.yml')
  })

  it('offers folder actions, disabling delete for non-empty folders', () => {
    setup({ onNewRequestInFolder: vi.fn(), onCreateFolder: vi.fn(), onDeleteFolder: vi.fn(), onRunFolder: vi.fn() })
    rightClick('users')
    expect(screen.getByRole('button', { name: 'New Request in Folder' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Run all requests' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Delete — folder not empty' })).toBeDisabled()
  })

  it('runs and creates from a folder menu', () => {
    const onRunFolder = vi.fn(), onNewRequestInFolder = vi.fn()
    setup({ onRunFolder, onNewRequestInFolder })
    rightClick('users')
    fireEvent.click(screen.getByRole('button', { name: 'Run all requests' }))
    expect(onRunFolder).toHaveBeenCalledWith('/ws/requests/users')
    rightClick('users')
    fireEvent.click(screen.getByRole('button', { name: 'New Request in Folder' }))
    expect(onNewRequestInFolder).toHaveBeenCalledWith('/ws/requests/users')
  })

  it('allows deleting an empty folder', () => {
    const onDeleteFolder = vi.fn()
    setup({ onDeleteFolder })
    rightClick('empty')
    fireEvent.click(screen.getByRole('button', { name: 'Delete' }))
    expect(onDeleteFolder).toHaveBeenCalledWith('/ws/requests/empty')
  })

  it('hides Rename for a top-level (root) folder', () => {
    setup()
    rightClick('requests')
    expect(screen.queryByRole('button', { name: 'Rename' })).not.toBeInTheDocument()
    fireEvent.keyDown(document, { key: 'Escape' })
    rightClick('users')
    expect(screen.getByRole('button', { name: 'Rename' })).toBeInTheDocument()
  })

  it('offers paste into a folder for a request clipboard', () => {
    const onPasteToFolder = vi.fn()
    setup({ onPasteToFolder, clipboard: { op: 'cut', kind: 'request', srcPath: '/ws/requests/health.yml' } })
    rightClick('users')
    fireEvent.click(screen.getByRole('button', { name: 'Move here' }))
    expect(onPasteToFolder).toHaveBeenCalledWith('/ws/requests/users')
  })

  it('closes on Escape', () => {
    setup()
    rightClick('health')
    fireEvent.keyDown(document, { key: 'Escape' })
    expect(screen.queryByRole('button', { name: 'Open in Explorer' })).not.toBeInTheDocument()
  })

  it('shows an explorer error from a folder and clears it after a few seconds', async () => {
    vi.useFakeTimers()
    b.openInExplorer.mockResolvedValue('explorer missing' as never)
    setup()
    rightClick('users')
    await act(async () => { fireEvent.click(screen.getByRole('button', { name: 'Open in Explorer' })) })
    expect(screen.getByText('explorer missing')).toBeInTheDocument()
    act(() => { vi.advanceTimersByTime(4100) })
    expect(screen.queryByText('explorer missing')).not.toBeInTheDocument()
  })

  it('shows no error when the explorer opens cleanly', async () => {
    setup()
    rightClick('users')
    await act(async () => { fireEvent.click(screen.getByRole('button', { name: 'Open in Explorer' })) })
    expect(b.openInExplorer).toHaveBeenCalledWith('/ws/requests/users')
  })
})
