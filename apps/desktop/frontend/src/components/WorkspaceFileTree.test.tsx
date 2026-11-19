import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { MethodBadge, RenameInput, TreeNode } from './WorkspaceFileTree'
import type { FileNode } from '@/lib/bridge'

const file = (name: string, method?: string, dir = '/ws/requests'): FileNode => ({
  name, path: `${dir}/${name}`, isDir: false, method,
})

const tree: FileNode = {
  name: 'requests', path: '/ws/requests', isDir: true,
  children: [
    { name: 'users', path: '/ws/requests/users', isDir: true, children: [file('get-user.yaml', 'GET', '/ws/requests/users')] },
    { name: 'empty', path: '/ws/requests/empty', isDir: true, children: [] },
    file('health.yml', 'GET'),
    file('notes.json'),
  ],
}

function renderTree(over: Partial<React.ComponentProps<typeof TreeNode>> = {}, node: FileNode = tree) {
  const props = {
    node,
    depth: 0,
    onSelect: vi.fn(),
    collapsedPaths: new Set<string>(),
    onToggleCollapse: vi.fn(),
    ...over,
  } as React.ComponentProps<typeof TreeNode>
  const utils = render(<TreeNode {...props} />)
  return { props, ...utils }
}

describe('MethodBadge', () => {
  it('renders the method with its colour class', () => {
    render(<MethodBadge method="GET" />)
    expect(screen.getByText('GET').className).toContain('text-emerald-400')
  })

  it('falls back to the muted colour for unknown methods', () => {
    render(<MethodBadge method="TRACE" />)
    expect(screen.getByText('TRACE').className).toContain('text-[var(--color-muted-foreground)]')
  })
})

describe('RenameInput', () => {
  function setup(initial = 'old') {
    const onConfirm = vi.fn(), onCancel = vi.fn()
    render(<RenameInput initialValue={initial} onConfirm={onConfirm} onCancel={onCancel} />)
    return { onConfirm, onCancel, input: screen.getByRole('textbox') as HTMLInputElement }
  }

  it('starts with the initial value', () => {
    expect(setup('abc').input.value).toBe('abc')
  })

  it('confirms the trimmed value on Enter', () => {
    const { input, onConfirm } = setup()
    fireEvent.change(input, { target: { value: '  fresh  ' } })
    fireEvent.keyDown(input, { key: 'Enter' })
    expect(onConfirm).toHaveBeenCalledWith('fresh')
  })

  it('does not confirm a blank value on Enter', () => {
    const { input, onConfirm } = setup()
    fireEvent.change(input, { target: { value: '   ' } })
    fireEvent.keyDown(input, { key: 'Enter' })
    expect(onConfirm).not.toHaveBeenCalled()
  })

  it('cancels on Escape', () => {
    const { input, onCancel } = setup()
    fireEvent.keyDown(input, { key: 'Escape' })
    expect(onCancel).toHaveBeenCalledTimes(1)
  })

  it('confirms on blur when changed, cancels when unchanged or blank', () => {
    const changed = setup('a')
    fireEvent.change(changed.input, { target: { value: 'b' } })
    fireEvent.blur(changed.input)
    expect(changed.onConfirm).toHaveBeenCalledWith('b')
  })

  it('cancels on blur when unchanged', () => {
    const { input, onCancel, onConfirm } = setup('same')
    fireEvent.blur(input)
    expect(onCancel).toHaveBeenCalledTimes(1)
    expect(onConfirm).not.toHaveBeenCalled()
  })

  it('stops key and click events from reaching parents', () => {
    const parentKey = vi.fn(), parentClick = vi.fn()
    render(
      <div onKeyDown={parentKey} onClick={parentClick}>
        <RenameInput initialValue="x" onConfirm={vi.fn()} onCancel={vi.fn()} />
      </div>,
    )
    const input = screen.getByRole('textbox')
    fireEvent.keyDown(input, { key: 'a' })
    fireEvent.click(input)
    expect(parentKey).not.toHaveBeenCalled()
    expect(parentClick).not.toHaveBeenCalled()
  })
})

describe('TreeNode — directories', () => {
  it('renders folder and nested children, hiding extensions on files', () => {
    renderTree()
    expect(screen.getByText('requests')).toBeInTheDocument()
    expect(screen.getByText('users')).toBeInTheDocument()
    expect(screen.getByText('get-user')).toBeInTheDocument()
    expect(screen.getByText('health')).toBeInTheDocument()
    expect(screen.getByText('notes')).toBeInTheDocument()
  })

  it('hides children of collapsed folders', () => {
    renderTree({ collapsedPaths: new Set(['/ws/requests/users']) })
    expect(screen.queryByText('get-user')).not.toBeInTheDocument()
    expect(screen.getByText('health')).toBeInTheDocument()
  })

  it('toggles a folder when its name or the empty part of its row is clicked, like scenario folders', () => {
    const { props } = renderTree()
    fireEvent.click(screen.getByText('users'))
    expect(props.onToggleCollapse).toHaveBeenCalledTimes(1)
    expect(props.onToggleCollapse).toHaveBeenLastCalledWith('/ws/requests/users')
    fireEvent.click(screen.getByText('users').closest('div[class*="group"]') as HTMLElement)
    expect(props.onToggleCollapse).toHaveBeenCalledTimes(2)
  })

  it('does not make folder names bold', () => {
    renderTree()
    expect(screen.getByText('users').className).not.toMatch(/font-(medium|semibold|bold)/)
  })

  it('toggles collapse when the chevron button is clicked', () => {
    const { props } = renderTree()
    fireEvent.click(screen.getAllByRole('button')[0])
    expect(props.onToggleCollapse).toHaveBeenCalledWith('/ws/requests')
  })

  it('shows a dash for files without a method and a 4-char badge otherwise', () => {
    renderTree({}, { name: 'r', path: '/r', isDir: true, children: [file('a.yaml', 'DELETE'), file('b.yaml')] })
    expect(screen.getByText('DELE')).toBeInTheDocument()
    expect(screen.getByText('—')).toBeInTheDocument()
  })

  it('only allows renaming a non-root folder by double-click', () => {
    renderTree()
    expect(screen.getByText('requests')).not.toHaveAttribute('title')
    expect(screen.getByText('users')).toHaveAttribute('title', 'Double-click to rename')
  })

  it('renames a folder inline and reports the new name', () => {
    const onRenameFolder = vi.fn()
    renderTree({ onRenameFolder })
    fireEvent.doubleClick(screen.getByText('users'))
    const input = screen.getByRole('textbox')
    fireEvent.change(input, { target: { value: 'people' } })
    fireEvent.keyDown(input, { key: 'Enter' })
    expect(onRenameFolder).toHaveBeenCalledWith('/ws/requests/users', 'people')
    expect(screen.queryByRole('textbox')).not.toBeInTheDocument()
  })

  it('cancelling a rename restores the label and calls onRenameEnd', () => {
    const onRenameEnd = vi.fn()
    renderTree({ onRenameEnd })
    fireEvent.doubleClick(screen.getByText('users'))
    fireEvent.keyDown(screen.getByRole('textbox'), { key: 'Escape' })
    expect(onRenameEnd).toHaveBeenCalledTimes(1)
    expect(screen.getByText('users')).toBeInTheDocument()
  })

  it('starts renaming when renamingPath matches externally', () => {
    renderTree({ renamingPath: '/ws/requests/users' })
    expect(screen.getByRole('textbox')).toHaveValue('users')
  })

  it('shows folder action buttons and invokes their callbacks', () => {
    const onNewRequestInFolder = vi.fn(), onCreateFolder = vi.fn()
    renderTree({ onNewRequestInFolder, onCreateFolder })
    fireEvent.click(screen.getAllByTitle('New request in this folder')[0])
    fireEvent.click(screen.getAllByTitle('New subfolder')[0])
    expect(onNewRequestInFolder).toHaveBeenCalledWith('/ws/requests')
    expect(onCreateFolder).toHaveBeenCalledWith('/ws/requests')
  })

  it('offers delete only for empty folders', () => {
    const onDeleteFolder = vi.fn()
    renderTree({ onDeleteFolder })
    const buttons = screen.getAllByTitle('Delete folder')
    expect(buttons).toHaveLength(1)
    fireEvent.click(buttons[0])
    expect(onDeleteFolder).toHaveBeenCalledWith('/ws/requests/empty')
  })

  it('treats a folder with only dotfiles as empty', () => {
    const onDeleteFolder = vi.fn()
    renderTree({ onDeleteFolder }, {
      name: 'f', path: '/f', isDir: true,
      children: [{ name: '.gitkeep', path: '/f/.gitkeep', isDir: false }],
    })
    expect(screen.getAllByTitle('Delete folder')).toHaveLength(1)
  })

  it('omits optional action buttons when callbacks are not provided', () => {
    renderTree()
    expect(screen.queryByTitle('New request in this folder')).not.toBeInTheDocument()
    expect(screen.queryByTitle('New subfolder')).not.toBeInTheDocument()
    expect(screen.queryByTitle('Delete folder')).not.toBeInTheDocument()
  })

  it('forwards the context menu with the node', () => {
    const onContextMenu = vi.fn()
    renderTree({ onContextMenu })
    fireEvent.contextMenu(screen.getByText('users'))
    expect(onContextMenu).toHaveBeenCalledTimes(1)
    expect(onContextMenu.mock.calls[0][1].path).toBe('/ws/requests/users')
  })
})

describe('TreeNode — drag and drop', () => {
  function dropData(path: string) {
    return { dataTransfer: { getData: () => path, setData: vi.fn(), dropEffect: '', effectAllowed: '' } }
  }

  it('moves a dropped request into the target folder', () => {
    const onMoveFile = vi.fn()
    renderTree({ onMoveFile })
    const folder = screen.getByText('users').closest('div[class*="group"]')!.parentElement!
    fireEvent.dragOver(folder, dropData(''))
    fireEvent.drop(folder, dropData('/ws/requests/health.yml'))
    expect(onMoveFile).toHaveBeenCalledWith('/ws/requests/health.yml', '/ws/requests/users')
  })

  it('ignores a drop onto itself or an empty payload', () => {
    const onMoveFile = vi.fn()
    renderTree({ onMoveFile })
    const folder = screen.getByText('users').closest('div[class*="group"]')!.parentElement!
    fireEvent.drop(folder, dropData('/ws/requests/users'))
    fireEvent.drop(folder, dropData(''))
    expect(onMoveFile).not.toHaveBeenCalled()
  })

  it('highlights on drag over and clears on drag leave', () => {
    renderTree()
    const row = screen.getByText('users').closest('div[class*="group"]') as HTMLElement
    const folder = row.parentElement!
    fireEvent.dragOver(folder, dropData(''))
    expect(row.className).toContain('bg-[var(--color-accent)]/10')
    fireEvent.dragLeave(folder, { relatedTarget: document.body })
    expect(row.className).not.toContain('bg-[var(--color-accent)]/10')
  })

  it('sets the drag payload when dragging a file', () => {
    renderTree()
    const row = screen.getByText('health').closest('[draggable="true"]') as HTMLElement
    const setData = vi.fn()
    fireEvent.dragStart(row, { dataTransfer: { setData, effectAllowed: '' } })
    expect(setData).toHaveBeenCalledWith('application/x-clrsigrequest', '/ws/requests/health.yml')
  })
})

describe('TreeNode — files', () => {
  const fileOnly = file('health.yml', 'GET')

  it('selects a file on click', () => {
    const { props } = renderTree({}, fileOnly)
    fireEvent.click(screen.getByText('health'))
    expect(props.onSelect).toHaveBeenCalledWith('/ws/requests/health.yml')
  })

  it('marks the active file', () => {
    renderTree({ activeFilePath: fileOnly.path }, fileOnly)
    expect(screen.getByText('health').closest('[draggable]')!.className).toContain('border-[var(--color-accent)]')
  })

  it('dims a file that is on the clipboard as cut, but not for copy', () => {
    const { unmount } = renderTree({ clipboard: { op: 'cut', kind: 'request', srcPath: fileOnly.path } }, fileOnly)
    expect(screen.getByText('health').closest('[draggable]')!.className).toContain('opacity-40')
    unmount()
    renderTree({ clipboard: { op: 'copy', kind: 'request', srcPath: fileOnly.path } }, fileOnly)
    expect(screen.getByText('health').closest('[draggable]')!.className).not.toContain('opacity-40')
  })

  it('shows an unsaved-changes dot for dirty files', () => {
    renderTree({ dirtyFilePaths: new Set([fileOnly.path]) }, fileOnly)
    expect(screen.getByTitle('Unsaved changes')).toBeInTheDocument()
  })

  it('shows no dirty dot for clean files', () => {
    renderTree({ dirtyFilePaths: new Set() }, fileOnly)
    expect(screen.queryByTitle('Unsaved changes')).not.toBeInTheDocument()
  })

  it('duplicates and deletes without selecting the file', () => {
    const onDuplicateFile = vi.fn(), onDelete = vi.fn()
    const { props } = renderTree({ onDuplicateFile, onDelete }, fileOnly)
    fireEvent.click(screen.getByTitle('Duplicate request'))
    fireEvent.click(screen.getByTitle('Delete request'))
    expect(onDuplicateFile).toHaveBeenCalledWith(fileOnly.path)
    expect(onDelete).toHaveBeenCalledWith(fileOnly.path)
    expect(props.onSelect).not.toHaveBeenCalled()
  })

  it('hides file action buttons when no handlers are supplied', () => {
    renderTree({}, fileOnly)
    expect(screen.queryByTitle('Duplicate request')).not.toBeInTheDocument()
    expect(screen.queryByTitle('Delete request')).not.toBeInTheDocument()
  })

  it('renames a file inline using the extensionless name', () => {
    const onRenameFile = vi.fn()
    renderTree({ onRenameFile }, fileOnly)
    fireEvent.doubleClick(screen.getByText('health'))
    const input = screen.getByRole('textbox')
    expect(input).toHaveValue('health')
    fireEvent.change(input, { target: { value: 'status' } })
    fireEvent.keyDown(input, { key: 'Enter' })
    expect(onRenameFile).toHaveBeenCalledWith(fileOnly.path, 'status')
  })

  it('hides action buttons while renaming', () => {
    renderTree({ onDelete: vi.fn(), renamingPath: fileOnly.path }, fileOnly)
    expect(screen.queryByTitle('Delete request')).not.toBeInTheDocument()
  })

  it('forwards the context menu with the file node', () => {
    const onContextMenu = vi.fn()
    renderTree({ onContextMenu }, fileOnly)
    fireEvent.contextMenu(screen.getByText('health'))
    expect(onContextMenu.mock.calls[0][1].path).toBe(fileOnly.path)
  })
})
