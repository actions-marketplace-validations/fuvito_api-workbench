import { useState, useRef, useEffect } from 'react'
import {
  ChevronRight, ChevronDown, FileText, Folder, FolderOpen,
  FolderSymlink, FolderPlus, FilePlus, Upload, Trash2, CopyPlus, Search, X, RefreshCw,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import type { FileNode } from '@/lib/bridge'
import { openInExplorer } from '@/lib/bridge'
import { ContextMenu, type ContextMenuAction } from '@/components/ContextMenu'

// ---- Tree filter ----

function filterTree(nodes: FileNode[], query: string): FileNode[] {
  if (!query) return nodes
  const q = query.toLowerCase()
  return nodes.flatMap(node => {
    if (node.isDir) {
      const filteredChildren = filterTree(node.children ?? [], q)
      if (filteredChildren.length > 0) return [{ ...node, children: filteredChildren }]
      if (node.name.toLowerCase().includes(q)) return [{ ...node }]
      return []
    }
    const displayName = node.name.replace(/\.(yaml|yml|json)$/i, '')
    return displayName.toLowerCase().includes(q) ? [node] : []
  })
}

// ---- Inline rename input ----

function RenameInput({
  initialValue,
  onConfirm,
  onCancel,
}: {
  initialValue: string
  onConfirm: (v: string) => void
  onCancel: () => void
}) {
  const [value, setValue] = useState(initialValue)
  const ref = useRef<HTMLInputElement>(null)

  useEffect(() => {
    ref.current?.select()
  }, [])

  return (
    <input
      ref={ref}
      autoFocus
      value={value}
      onChange={e => setValue(e.target.value)}
      onKeyDown={e => {
        e.stopPropagation()
        if (e.key === 'Enter' && value.trim()) { onConfirm(value.trim()); return }
        if (e.key === 'Escape') onCancel()
      }}
      onBlur={() => {
        if (value.trim() && value.trim() !== initialValue) onConfirm(value.trim())
        else onCancel()
      }}
      onClick={e => e.stopPropagation()}
      className={cn(
        'flex-1 min-w-0 bg-transparent outline-none text-xs',
        'border-b border-[var(--color-accent)] text-[var(--color-foreground)]',
      )}
    />
  )
}

// ---- Tree node ----

type TreeNodeProps = {
  node: FileNode
  depth: number
  onSelect: (path: string) => void
  onDelete?: (path: string) => void
  onDuplicateFile?: (path: string) => void
  onDeleteFolder?: (path: string) => void
  onNewRequestInFolder?: (folderPath: string) => void
  onCreateFolder?: (parentPath: string) => void
  onRenameFile?: (path: string, newName: string) => void
  onRenameFolder?: (path: string, newName: string) => void
  activeFilePath?: string
  dirtyFilePaths?: Set<string>
  renamingPath?: string | null
  onRenameEnd?: () => void
  onContextMenu?: (e: React.MouseEvent, node: FileNode) => void
}

function TreeNode({
  node, depth,
  onSelect, onDelete, onDuplicateFile, onDeleteFolder, onNewRequestInFolder, onCreateFolder,
  onRenameFile, onRenameFolder,
  activeFilePath, dirtyFilePaths,
  renamingPath, onRenameEnd, onContextMenu,
}: TreeNodeProps) {
  const [localRenaming, setLocalRenaming] = useState(false)
  const [expanded, setExpanded] = useState(true)
  const indent = depth * 12

  const isRenaming = localRenaming || renamingPath === node.path

  function endRename() {
    setLocalRenaming(false)
    onRenameEnd?.()
  }

  if (node.isDir) {
    const isEmpty = !node.children?.some(c => !c.name.startsWith('.'))
    return (
      <div>
        <div
          className="group flex items-center hover:bg-[var(--color-muted)] transition-colors"
          style={{ paddingLeft: `${8 + indent}px` }}
          onContextMenu={e => { e.preventDefault(); e.stopPropagation(); onContextMenu?.(e, node) }}
        >
          <button
            type="button"
            onClick={() => setExpanded(e => !e)}
            className="flex items-center gap-1 py-0.5 shrink-0 text-[var(--color-muted-foreground)] group-hover:text-[var(--color-foreground)]"
          >
            <span className="shrink-0">
              {expanded ? <ChevronDown size={11} /> : <ChevronRight size={11} />}
            </span>
            <span className="shrink-0">
              {expanded ? <FolderOpen size={12} /> : <Folder size={12} />}
            </span>
          </button>
          <div className="flex-1 min-w-0 flex items-center py-0.5 mr-1">
            {isRenaming ? (
              <RenameInput
                initialValue={node.name}
                onConfirm={v => { onRenameFolder?.(node.path, v); endRename() }}
                onCancel={endRename}
              />
            ) : (
              <span
                className="truncate text-xs text-[var(--color-muted-foreground)] group-hover:text-[var(--color-foreground)] cursor-default select-none"
                onDoubleClick={() => setLocalRenaming(true)}
                title="Double-click to rename"
              >
                {node.name}
              </span>
            )}
          </div>
          {!isRenaming && (
            <div className="flex items-center gap-0.5 mr-1 opacity-0 group-hover:opacity-100 transition-opacity shrink-0">
              {onNewRequestInFolder && (
                <button
                  type="button"
                  onClick={e => { e.stopPropagation(); onNewRequestInFolder(node.path) }}
                  title="New request in this folder"
                  className="p-0.5 rounded text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)]"
                >
                  <FilePlus size={10} />
                </button>
              )}
              {onCreateFolder && (
                <button
                  type="button"
                  onClick={e => { e.stopPropagation(); onCreateFolder(node.path) }}
                  title="New subfolder"
                  className="p-0.5 rounded text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)]"
                >
                  <FolderPlus size={10} />
                </button>
              )}
              {isEmpty && onDeleteFolder && (
                <button
                  type="button"
                  onClick={e => { e.stopPropagation(); onDeleteFolder(node.path) }}
                  title="Delete folder"
                  className="p-0.5 rounded text-[var(--color-muted-foreground)] hover:text-red-400"
                >
                  <Trash2 size={10} />
                </button>
              )}
            </div>
          )}
        </div>
        {expanded && node.children?.map(child => (
          <TreeNode
            key={child.path}
            node={child}
            depth={depth + 1}
            onSelect={onSelect}
            onDelete={onDelete}
            onDuplicateFile={onDuplicateFile}
            onDeleteFolder={onDeleteFolder}
            onNewRequestInFolder={onNewRequestInFolder}
            onCreateFolder={onCreateFolder}
            onRenameFile={onRenameFile}
            onRenameFolder={onRenameFolder}
            activeFilePath={activeFilePath}
            dirtyFilePaths={dirtyFilePaths}
            renamingPath={renamingPath}
            onRenameEnd={onRenameEnd}
            onContextMenu={onContextMenu}
          />
        ))}
      </div>
    )
  }

  const isActive = node.path === activeFilePath
  const displayName = node.name.replace(/\.(yaml|yml|json)$/i, '')

  return (
    <div
      className={cn(
        'group flex items-center gap-1.5 py-0.5 text-xs transition-colors',
        isActive
          ? 'bg-[var(--color-muted)] text-[var(--color-foreground)] border-l-2 border-[var(--color-accent)]'
          : 'text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-muted)] border-l-2 border-transparent',
      )}
      style={{ paddingLeft: `${6 + indent + 15}px` }}
      onContextMenu={e => { e.preventDefault(); e.stopPropagation(); onContextMenu?.(e, node) }}
    >
      <FileText size={11} className="shrink-0" />
      {isRenaming ? (
        <RenameInput
          initialValue={displayName}
          onConfirm={v => { onRenameFile?.(node.path, v); endRename() }}
          onCancel={endRename}
        />
      ) : (
        <button
          type="button"
          onClick={() => onSelect(node.path)}
          className="flex items-center gap-1.5 flex-1 min-w-0 text-left"
        >
          <span
            className="truncate flex-1"
            title={node.name}
            onDoubleClick={e => { e.stopPropagation(); setLocalRenaming(true) }}
          >
            {displayName}
          </span>
          {dirtyFilePaths?.has(node.path) && (
            <span className="shrink-0 w-1.5 h-1.5 rounded-full bg-amber-400" title="Unsaved changes" />
          )}
        </button>
      )}
      {!isRenaming && (onDuplicateFile || onDelete) && (
        <div className="flex items-center mr-1 opacity-0 group-hover:opacity-100 transition-opacity shrink-0">
          {onDuplicateFile && (
            <button
              type="button"
              onClick={e => { e.stopPropagation(); onDuplicateFile(node.path) }}
              title="Duplicate request"
              className="p-0.5 rounded text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)]"
            >
              <CopyPlus size={10} />
            </button>
          )}
          {onDelete && (
            <button
              type="button"
              onClick={e => { e.stopPropagation(); onDelete(node.path) }}
              title="Delete request"
              className="p-0.5 rounded text-[var(--color-muted-foreground)] hover:text-red-400"
            >
              <Trash2 size={10} />
            </button>
          )}
        </div>
      )}
    </div>
  )
}

// ---- Panel ----

type WorkspacePanelProps = {
  workspacePath: string
  fileTree: FileNode[]
  onOpenWorkspace: () => void
  onNewWorkspace: () => void
  onSelectFile: (path: string) => void
  onDeleteFile?: (path: string) => void
  onDuplicateFile?: (path: string) => void
  onDeleteFolder?: (path: string) => void
  onNewRequestInFolder?: (folderPath: string) => void
  onCreateFolder?: (parentPath: string) => void
  onRenameFile?: (path: string, newName: string) => void
  onRenameFolder?: (path: string, newName: string) => void
  onImportOpenAPI: () => void
  onRefresh?: () => void
  importing?: boolean
  activeFilePath?: string
  dirtyFilePaths?: Set<string>
}

export function WorkspacePanel({
  workspacePath,
  fileTree,
  onOpenWorkspace,
  onNewWorkspace,
  onSelectFile,
  onDeleteFile,
  onDuplicateFile,
  onDeleteFolder,
  onNewRequestInFolder,
  onCreateFolder,
  onRenameFile,
  onRenameFolder,
  onImportOpenAPI,
  onRefresh,
  importing,
  activeFilePath,
  dirtyFilePaths,
}: WorkspacePanelProps) {
  const folderName = workspacePath
    ? workspacePath.replace(/\\/g, '/').split('/').filter(Boolean).pop()
    : null
  const [searchQuery, setSearchQuery] = useState('')
  const filteredTree = filterTree(fileTree, searchQuery)

  // ---- Context menu ----
  const [ctxMenu, setCtxMenu] = useState<{ x: number; y: number; node: FileNode } | null>(null)
  const [renamingPath, setRenamingPath] = useState<string | null>(null)

  function handleContextMenu(e: React.MouseEvent, node: FileNode) {
    e.preventDefault()
    e.stopPropagation()
    setCtxMenu({ x: e.clientX, y: e.clientY, node })
  }

  function buildCtxItems(node: FileNode): ContextMenuAction[] {
    if (node.isDir) {
      const isEmpty = !node.children?.some(c => !c.name.startsWith('.'))
      return [
        ...(onNewRequestInFolder ? [{ kind: 'action' as const, label: 'New Request in Folder', onClick: () => onNewRequestInFolder(node.path) }] : []),
        ...(onCreateFolder ? [{ kind: 'action' as const, label: 'New Subfolder', onClick: () => onCreateFolder(node.path) }] : []),
        { kind: 'separator' },
        { kind: 'action', label: 'Rename', onClick: () => setRenamingPath(node.path) },
        { kind: 'action', label: 'Open in Explorer', onClick: () => openInExplorer(node.path) },
        { kind: 'separator' },
        {
          kind: 'action',
          label: isEmpty ? 'Delete' : 'Delete — folder not empty',
          danger: true,
          disabled: !isEmpty || !onDeleteFolder,
          onClick: () => onDeleteFolder?.(node.path),
        },
      ]
    }
    return [
      { kind: 'action', label: 'Open', onClick: () => onSelectFile(node.path) },
      ...(onDuplicateFile ? [{ kind: 'action' as const, label: 'Duplicate', onClick: () => onDuplicateFile(node.path) }] : []),
      { kind: 'separator' },
      { kind: 'action', label: 'Rename', onClick: () => setRenamingPath(node.path) },
      { kind: 'action', label: 'Open in Explorer', onClick: () => openInExplorer(node.path) },
      { kind: 'separator' },
      {
        kind: 'action',
        label: 'Delete',
        danger: true,
        disabled: !onDeleteFile,
        onClick: () => onDeleteFile?.(node.path),
      },
    ]
  }

  return (
    <div className="flex flex-col h-full">
      {workspacePath ? (
        <>
          <div className="px-2 py-1.5 border-b border-[var(--color-border)] flex items-center gap-1 justify-between">
            <span className="text-label font-mono text-[var(--color-muted-foreground)] truncate" title={workspacePath}>
              {folderName}
            </span>
            <div className="flex items-center gap-1 shrink-0">
              {onCreateFolder && (
                <button
                  type="button"
                  onClick={() => onCreateFolder('')}
                  className="text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors"
                  title="New folder"
                >
                  <FolderPlus size={12} />
                </button>
              )}
              <button
                type="button"
                onClick={onImportOpenAPI}
                disabled={importing}
                className="text-[var(--color-muted-foreground)] hover:text-[var(--color-accent)] transition-colors disabled:opacity-40"
                title="Import OpenAPI spec"
              >
                {importing ? <span className="text-label">…</span> : <Upload size={12} />}
              </button>
              {onRefresh && (
                <button
                  type="button"
                  onClick={onRefresh}
                  className="text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors"
                  title="Refresh file tree"
                >
                  <RefreshCw size={12} />
                </button>
              )}
              <button
                type="button"
                onClick={onOpenWorkspace}
                className="text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors"
                title="Switch workspace folder"
              >
                <FolderSymlink size={12} />
              </button>
            </div>
          </div>
          {fileTree.length > 0 && (
            <div className="flex items-center gap-1.5 px-2 py-1 border-b border-[var(--color-border)]">
              <Search size={10} className="shrink-0 text-[var(--color-muted-foreground)]" />
              <input
                type="text"
                value={searchQuery}
                onChange={e => setSearchQuery(e.target.value)}
                placeholder="Filter requests…"
                className="flex-1 min-w-0 bg-transparent text-xs outline-none text-[var(--color-foreground)] placeholder:text-[var(--color-muted-foreground)]"
              />
              {searchQuery && (
                <button type="button" onClick={() => setSearchQuery('')} className="shrink-0 text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)]">
                  <X size={10} />
                </button>
              )}
            </div>
          )}
          <div className="flex-1 overflow-auto py-1">
            {fileTree.length === 0 ? (
              <p className="px-3 py-2 text-xs text-[var(--color-muted-foreground)]">
                No request files found.
              </p>
            ) : filteredTree.length === 0 ? (
              <p className="px-3 py-2 text-xs text-[var(--color-muted-foreground)]">No matches.</p>
            ) : (
              filteredTree.map(node => (
                <TreeNode
                  key={node.path}
                  node={node}
                  depth={0}
                  onSelect={onSelectFile}
                  onDelete={onDeleteFile}
                  onDuplicateFile={onDuplicateFile}
                  onDeleteFolder={onDeleteFolder}
                  onNewRequestInFolder={onNewRequestInFolder}
                  onCreateFolder={onCreateFolder}
                  onRenameFile={onRenameFile}
                  onRenameFolder={onRenameFolder}
                  activeFilePath={activeFilePath}
                  dirtyFilePaths={dirtyFilePaths}
                  renamingPath={renamingPath}
                  onRenameEnd={() => setRenamingPath(null)}
                  onContextMenu={handleContextMenu}
                />
              ))
            )}
          </div>
        </>
      ) : (
        <div className="flex flex-col gap-2 p-3">
          <p className="text-xs text-[var(--color-muted-foreground)]">
            Open a workspace folder to manage your request collections and environments.
          </p>
          <button
            type="button"
            onClick={onNewWorkspace}
            className={cn(
              'text-xs rounded px-3 py-1.5 font-semibold',
              'bg-[var(--color-accent)] text-white hover:opacity-90 transition-opacity',
            )}
          >
            New Workspace
          </button>
          <button
            type="button"
            onClick={onOpenWorkspace}
            className={cn(
              'text-xs rounded px-3 py-1.5',
              'bg-[var(--color-muted)] text-[var(--color-foreground)] border border-[var(--color-border)]',
              'hover:bg-[var(--color-border)] transition-colors',
            )}
          >
            Open Existing
          </button>
        </div>
      )}

      {ctxMenu && (
        <ContextMenu
          x={ctxMenu.x}
          y={ctxMenu.y}
          items={buildCtxItems(ctxMenu.node)}
          onClose={() => setCtxMenu(null)}
        />
      )}
    </div>
  )
}
