import type { FileNode } from '@/lib/bridge'
import { openInExplorer } from '@/lib/bridge'
import type { ContextMenuAction } from '@/components/ContextMenu'

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
}

export function buildFileMenuItems(node: FileNode, h: FileMenuHandlers): ContextMenuAction[] {
  if (node.isDir) {
    const isEmpty = !node.children?.some(c => !c.name.startsWith('.'))
    const hasRequests = node.children?.some(c => !c.isDir) ?? false
    return [
      ...(h.onNewRequestInFolder ? [{ kind: 'action' as const, label: 'New Request in Folder', onClick: () => h.onNewRequestInFolder!(node.path) }] : []),
      ...(h.onCreateFolder ? [{ kind: 'action' as const, label: 'New Subfolder', onClick: () => h.onCreateFolder!(node.path) }] : []),
      ...(h.onRunFolder && hasRequests ? [
        { kind: 'separator' as const },
        { kind: 'action' as const, label: 'Run all requests', onClick: () => h.onRunFolder!(node.path) },
      ] : []),
      { kind: 'separator' },
      { kind: 'action', label: 'Rename', onClick: () => h.onRename(node.path) },
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
}

export function buildScenarioMenuItems(
  target: { path: string; name: string },
  h: ScenarioMenuHandlers,
): ContextMenuAction[] {
  return [
    { kind: 'action', label: 'Open', onClick: () => h.onOpen(target.path, target.name) },
    ...(h.onDuplicate ? [{ kind: 'action' as const, label: 'Duplicate', onClick: () => h.onDuplicate!(target.path) }] : []),
    { kind: 'separator' as const },
    { kind: 'action', label: 'Rename', onClick: () => h.onRename(target.path) },
    { kind: 'action', label: 'Open in Explorer', onClick: () => { openInExplorer(target.path) } },
    { kind: 'separator' as const },
    { kind: 'action', label: 'Delete', danger: true, disabled: !h.onDelete, onClick: () => h.onDelete?.(target.path) },
  ]
}
