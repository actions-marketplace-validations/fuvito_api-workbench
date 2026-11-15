import type { FileNode } from '@/lib/bridge'
import { openInExplorer } from '@/lib/bridge'
import type { ContextMenuAction } from '@/components/ContextMenu'
import type { FileClipboard } from '@/types'

export type FileMenuHandlers = {
  onSelectFile: (path: string) => void
  onDuplicateFile?: (path: string) => void
  onDeleteFile?: (path: string) => void
  onDeleteFolder?: (path: string) => void
  onNewRequestInFolder?: (folderPath: string) => void
  onCreateFolder?: (parentPath: string) => void
  onRunFolder?: (folderPath: string) => void
  onRename: (path: string) => void
  onOpenInExplorer: (path: string) => void
  onCopy?: (path: string) => void
  onCut?: (path: string) => void
  onPasteToFolder?: (folderPath: string) => void
  clipboard?: FileClipboard | null
  isRootFolder?: boolean
}

export function buildFileMenuItems(node: FileNode, h: FileMenuHandlers): ContextMenuAction[] {
  if (node.isDir) {
    const isEmpty = !node.children?.some(c => !c.name.startsWith('.'))
    const hasRequests = node.children?.some(c => !c.isDir) ?? false
    const canPaste = !!(h.clipboard?.kind === 'request' && h.onPasteToFolder)
    return [
      ...(h.onNewRequestInFolder ? [{ kind: 'action' as const, label: 'New Request in Folder', onClick: () => h.onNewRequestInFolder!(node.path) }] : []),
      ...(h.onCreateFolder ? [{ kind: 'action' as const, label: 'New Subfolder', onClick: () => h.onCreateFolder!(node.path) }] : []),
      ...(h.onRunFolder && hasRequests ? [
        { kind: 'separator' as const },
        { kind: 'action' as const, label: 'Run all requests', onClick: () => h.onRunFolder!(node.path) },
      ] : []),
      ...(canPaste ? [
        { kind: 'separator' as const },
        { kind: 'action' as const, label: h.clipboard!.op === 'cut' ? 'Move here' : 'Paste copy here', onClick: () => h.onPasteToFolder!(node.path) },
      ] : []),
      { kind: 'separator' },
      ...(!h.isRootFolder ? [{ kind: 'action' as const, label: 'Rename', onClick: () => h.onRename(node.path) }] : []),
      { kind: 'action', label: 'Open in Explorer', onClick: () => h.onOpenInExplorer(node.path) },
      { kind: 'separator' },
      {
        kind: 'action',
        label: isEmpty ? 'Delete' : 'Delete — folder not empty',
        danger: true,
        disabled: !isEmpty || !h.onDeleteFolder,
        onClick: () => h.onDeleteFolder?.(node.path),
      },
    ]
  }
  return [
    { kind: 'action', label: 'Open', onClick: () => h.onSelectFile(node.path) },
    ...(h.onDuplicateFile ? [{ kind: 'action' as const, label: 'Duplicate', onClick: () => h.onDuplicateFile!(node.path) }] : []),
    ...(h.onCopy || h.onCut ? [
      { kind: 'separator' as const },
      ...(h.onCopy ? [{ kind: 'action' as const, label: 'Copy', onClick: () => h.onCopy!(node.path) }] : []),
      ...(h.onCut ? [{ kind: 'action' as const, label: 'Cut', onClick: () => h.onCut!(node.path) }] : []),
    ] : []),
    { kind: 'separator' },
    { kind: 'action', label: 'Rename', onClick: () => h.onRename(node.path) },
    { kind: 'action', label: 'Open in Explorer', onClick: () => { openInExplorer(node.path) } },
    { kind: 'separator' },
    {
      kind: 'action',
      label: 'Delete',
      danger: true,
      disabled: !h.onDeleteFile,
      onClick: () => h.onDeleteFile?.(node.path),
    },
  ]
}

export type ScenarioMenuHandlers = {
  onOpen: (path: string, name: string) => void
  onDuplicate?: (path: string) => void
  onRename: (path: string) => void
  onDelete?: (path: string) => void
  onCopy?: (path: string) => void
  onCut?: (path: string) => void
}

export function buildScenarioMenuItems(
  target: { path: string; name: string },
  h: ScenarioMenuHandlers,
): ContextMenuAction[] {
  return [
    { kind: 'action', label: 'Open', onClick: () => h.onOpen(target.path, target.name) },
    ...(h.onDuplicate ? [{ kind: 'action' as const, label: 'Duplicate', onClick: () => h.onDuplicate!(target.path) }] : []),
    ...(h.onCopy || h.onCut ? [
      { kind: 'separator' as const },
      ...(h.onCopy ? [{ kind: 'action' as const, label: 'Copy', onClick: () => h.onCopy!(target.path) }] : []),
      ...(h.onCut ? [{ kind: 'action' as const, label: 'Cut', onClick: () => h.onCut!(target.path) }] : []),
    ] : []),
    { kind: 'separator' as const },
    { kind: 'action', label: 'Rename', onClick: () => h.onRename(target.path) },
    { kind: 'action', label: 'Open in Explorer', onClick: () => { openInExplorer(target.path) } },
    { kind: 'separator' as const },
    { kind: 'action', label: 'Delete', danger: true, disabled: !h.onDelete, onClick: () => h.onDelete?.(target.path) },
  ]
}

export type ScenarioFolderMenuHandlers = {
  onRename: (folderPath: string) => void
  onDelete: (folderPath: string) => void
  onRunFolder?: (folderPath: string) => void
  onPasteToFolder?: (relFolderPath: string) => void
  clipboard?: FileClipboard | null
}

export function buildScenarioFolderMenuItems(
  folderPath: string,
  h: ScenarioFolderMenuHandlers,
): ContextMenuAction[] {
  const isRoot = folderPath === ''
  const canPaste = !!(h.clipboard?.kind === 'scenario' && h.onPasteToFolder)
  return [
    ...(h.onRunFolder ? [
      { kind: 'action' as const, label: isRoot ? 'Run all scenarios' : 'Run all in folder', onClick: () => h.onRunFolder!(folderPath) },
      { kind: 'separator' as const },
    ] : []),
    ...(!isRoot ? [{ kind: 'action' as const, label: 'Rename', onClick: () => h.onRename(folderPath) }] : []),
    ...(canPaste ? [
      ...(!isRoot ? [{ kind: 'separator' as const }] : []),
      { kind: 'action' as const, label: h.clipboard!.op === 'cut' ? 'Move here' : 'Paste copy here', onClick: () => h.onPasteToFolder!(folderPath) },
    ] : []),
    ...(!isRoot ? [
      { kind: 'separator' as const },
      { kind: 'action' as const, label: 'Delete Folder', danger: true, onClick: () => h.onDelete(folderPath) },
    ] : []),
  ]
}
