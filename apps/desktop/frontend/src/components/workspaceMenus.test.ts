import { describe, it, expect, vi, beforeEach } from 'vitest'
import type { FileNode } from '@/lib/bridge'
import type { ContextMenuAction } from '@/components/ContextMenu'

vi.mock('@/lib/bridge', () => ({ openInExplorer: vi.fn() }))

import { openInExplorer } from '@/lib/bridge'
import { buildFileMenuItems, buildScenarioMenuItems, buildScenarioFolderMenuItems } from './workspaceMenus'

const labels = (items: ContextMenuAction[]) => items.flatMap(i => (i.kind === 'action' ? [i.label] : []))
const find = (items: ContextMenuAction[], label: string) => {
  const item = items.find(i => i.kind === 'action' && i.label === label)
  if (!item || item.kind !== 'action') throw new Error(`no action "${label}" in ${labels(items).join(', ')}`)
  return item
}

const file: FileNode = { name: 'a.yaml', path: '/ws/requests/a.yaml', isDir: false }
const dir = (children: FileNode[] = []): FileNode => ({ name: 'f', path: '/ws/requests/f', isDir: true, children })

beforeEach(() => vi.clearAllMocks())

describe('buildFileMenuItems — files', () => {
  it('always offers Open, Rename, Open in Explorer and Delete', () => {
    const items = buildFileMenuItems(file, { onSelectFile: vi.fn(), onRename: vi.fn(), onOpenInExplorer: vi.fn() })
    expect(labels(items)).toEqual(['Open', 'Rename', 'Open in Explorer', 'Delete'])
  })

  it('disables Delete when no delete handler is supplied', () => {
    const items = buildFileMenuItems(file, { onSelectFile: vi.fn(), onRename: vi.fn(), onOpenInExplorer: vi.fn() })
    expect(find(items, 'Delete')).toMatchObject({ disabled: true, danger: true })
  })

  it('wires handlers to the file path', () => {
    const h = {
      onSelectFile: vi.fn(), onRename: vi.fn(), onOpenInExplorer: vi.fn(),
      onDuplicateFile: vi.fn(), onDeleteFile: vi.fn(), onCopy: vi.fn(), onCut: vi.fn(),
    }
    const items = buildFileMenuItems(file, h)
    for (const [label, fn] of [
      ['Open', h.onSelectFile], ['Duplicate', h.onDuplicateFile], ['Copy', h.onCopy],
      ['Cut', h.onCut], ['Rename', h.onRename], ['Delete', h.onDeleteFile],
    ] as const) {
      find(items, label).onClick()
      expect(fn).toHaveBeenCalledWith(file.path)
    }
    expect(find(items, 'Delete').disabled).toBe(false)
  })

  it('opens files in the explorer through the bridge rather than a handler', () => {
    const h = { onSelectFile: vi.fn(), onRename: vi.fn(), onOpenInExplorer: vi.fn() }
    find(buildFileMenuItems(file, h), 'Open in Explorer').onClick()
    expect(openInExplorer).toHaveBeenCalledWith(file.path)
    expect(h.onOpenInExplorer).not.toHaveBeenCalled()
  })
})

describe('buildFileMenuItems — folders', () => {
  const base = { onSelectFile: vi.fn(), onRename: vi.fn(), onOpenInExplorer: vi.fn() }

  it('treats hidden files as empty and allows deletion', () => {
    const items = buildFileMenuItems(dir([{ name: '.DS_Store', path: 'x', isDir: false }]), { ...base, onDeleteFolder: vi.fn() })
    expect(find(items, 'Delete')).toMatchObject({ disabled: false })
  })

  it('blocks deletion of non-empty folders and says why', () => {
    const items = buildFileMenuItems(dir([file]), { ...base, onDeleteFolder: vi.fn() })
    expect(find(items, 'Delete — folder not empty').disabled).toBe(true)
  })

  it('blocks deletion when there is no handler', () => {
    expect(find(buildFileMenuItems(dir(), base), 'Delete').disabled).toBe(true)
  })

  it('only offers "Run all requests" when the folder directly contains requests', () => {
    const onRunFolder = vi.fn()
    expect(labels(buildFileMenuItems(dir(), { ...base, onRunFolder }))).not.toContain('Run all requests')
    expect(labels(buildFileMenuItems(dir([dir()]), { ...base, onRunFolder }))).not.toContain('Run all requests')
    find(buildFileMenuItems(dir([file]), { ...base, onRunFolder }), 'Run all requests').onClick()
    expect(onRunFolder).toHaveBeenCalledWith('/ws/requests/f')
  })

  it('hides Rename for the root folder', () => {
    expect(labels(buildFileMenuItems(dir(), { ...base, isRootFolder: true }))).not.toContain('Rename')
    expect(labels(buildFileMenuItems(dir(), base))).toContain('Rename')
  })

  it('offers paste only for request clipboards and labels cut vs copy', () => {
    const onPasteToFolder = vi.fn()
    const copy = { op: 'copy', kind: 'request', srcPath: '/a' } as const
    const cut = { op: 'cut', kind: 'request', srcPath: '/a' } as const
    const scenario = { op: 'copy', kind: 'scenario', srcPath: '/a' } as const

    expect(labels(buildFileMenuItems(dir(), { ...base, onPasteToFolder, clipboard: copy }))).toContain('Paste copy here')
    expect(labels(buildFileMenuItems(dir(), { ...base, onPasteToFolder, clipboard: cut }))).toContain('Move here')
    expect(labels(buildFileMenuItems(dir(), { ...base, onPasteToFolder, clipboard: scenario }))).not.toContain('Paste copy here')
    expect(labels(buildFileMenuItems(dir(), { ...base, clipboard: copy }))).not.toContain('Paste copy here')
  })

  it('includes creation actions only when handlers are provided', () => {
    const onNewRequestInFolder = vi.fn()
    const onCreateFolder = vi.fn()
    const items = buildFileMenuItems(dir(), { ...base, onNewRequestInFolder, onCreateFolder })
    find(items, 'New Request in Folder').onClick()
    find(items, 'New Subfolder').onClick()
    expect(onNewRequestInFolder).toHaveBeenCalledWith('/ws/requests/f')
    expect(onCreateFolder).toHaveBeenCalledWith('/ws/requests/f')
    expect(labels(buildFileMenuItems(dir(), base))).not.toContain('New Subfolder')
  })

  it('opens folders in the explorer via the handler', () => {
    find(buildFileMenuItems(dir(), base), 'Open in Explorer').onClick()
    expect(base.onOpenInExplorer).toHaveBeenCalledWith('/ws/requests/f')
  })
})

describe('buildScenarioMenuItems', () => {
  const target = { path: '/ws/scenarios/s.yaml', name: 'S' }

  it('passes both path and name to onOpen', () => {
    const onOpen = vi.fn()
    find(buildScenarioMenuItems(target, { onOpen, onRename: vi.fn() }), 'Open').onClick()
    expect(onOpen).toHaveBeenCalledWith(target.path, target.name)
  })

  it('builds the full menu with every handler and routes clicks', () => {
    const h = { onOpen: vi.fn(), onRename: vi.fn(), onDuplicate: vi.fn(), onDelete: vi.fn(), onCopy: vi.fn(), onCut: vi.fn() }
    const items = buildScenarioMenuItems(target, h)
    expect(labels(items)).toEqual(['Open', 'Duplicate', 'Copy', 'Cut', 'Rename', 'Open in Explorer', 'Delete'])
    for (const [label, fn] of [['Duplicate', h.onDuplicate], ['Copy', h.onCopy], ['Cut', h.onCut], ['Rename', h.onRename], ['Delete', h.onDelete]] as const) {
      find(items, label).onClick()
      expect(fn).toHaveBeenCalledWith(target.path)
    }
    find(items, 'Open in Explorer').onClick()
    expect(openInExplorer).toHaveBeenCalledWith(target.path)
  })

  it('omits optional actions and disables Delete without a handler', () => {
    const items = buildScenarioMenuItems(target, { onOpen: vi.fn(), onRename: vi.fn() })
    expect(labels(items)).toEqual(['Open', 'Rename', 'Open in Explorer', 'Delete'])
    expect(find(items, 'Delete').disabled).toBe(true)
  })
})

describe('buildScenarioFolderMenuItems', () => {
  const h = { onRename: vi.fn(), onDelete: vi.fn() }

  it('the root folder cannot be renamed or deleted', () => {
    expect(buildScenarioFolderMenuItems('', h)).toEqual([])
  })

  it('subfolders get Rename and Delete Folder', () => {
    const items = buildScenarioFolderMenuItems('auth', h)
    expect(labels(items)).toEqual(['Rename', 'Delete Folder'])
    find(items, 'Rename').onClick()
    find(items, 'Delete Folder').onClick()
    expect(h.onRename).toHaveBeenCalledWith('auth')
    expect(h.onDelete).toHaveBeenCalledWith('auth')
    expect(find(items, 'Delete Folder').danger).toBe(true)
  })

  it('labels the run action differently for the root', () => {
    const onRunFolder = vi.fn()
    expect(labels(buildScenarioFolderMenuItems('', { ...h, onRunFolder }))).toEqual(['Run all scenarios'])
    const sub = buildScenarioFolderMenuItems('auth', { ...h, onRunFolder })
    find(sub, 'Run all in folder').onClick()
    expect(onRunFolder).toHaveBeenCalledWith('auth')
  })

  it('offers paste only for scenario clipboards', () => {
    const onPasteToFolder = vi.fn()
    const scenarioCopy = { op: 'copy', kind: 'scenario', srcPath: '/a' } as const
    const scenarioCut = { op: 'cut', kind: 'scenario', srcPath: '/a' } as const
    const request = { op: 'copy', kind: 'request', srcPath: '/a' } as const

    expect(labels(buildScenarioFolderMenuItems('', { ...h, onPasteToFolder, clipboard: scenarioCopy }))).toEqual(['Paste copy here'])
    expect(labels(buildScenarioFolderMenuItems('x', { ...h, onPasteToFolder, clipboard: scenarioCut }))).toContain('Move here')
    expect(labels(buildScenarioFolderMenuItems('x', { ...h, onPasteToFolder, clipboard: request }))).not.toContain('Move here')
    find(buildScenarioFolderMenuItems('x', { ...h, onPasteToFolder, clipboard: scenarioCopy }), 'Paste copy here').onClick()
    expect(onPasteToFolder).toHaveBeenCalledWith('x')
  })
})
