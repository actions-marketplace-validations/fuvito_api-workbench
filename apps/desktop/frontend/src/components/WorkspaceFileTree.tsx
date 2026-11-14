import { useState, useRef, useEffect } from 'react'
import {
  ChevronRight, ChevronDown, Folder, FolderOpen,
  FolderPlus, FilePlus, Trash2, CopyPlus,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import type { FileNode } from '@/lib/bridge'

// ---- Method badge ----

const METHOD_COLORS: Record<string, string> = {
  GET:     'text-emerald-400',
  POST:    'text-blue-400',
  PUT:     'text-amber-400',
  PATCH:   'text-violet-400',
  DELETE:  'text-red-400',
  HEAD:    'text-cyan-400',
  OPTIONS: 'text-slate-400',
}

export function MethodBadge({ method }: { method: string }) {
  const color = METHOD_COLORS[method] ?? 'text-[var(--color-muted-foreground)]'
  return (
    <span className={cn('shrink-0 font-mono text-[9px] font-semibold leading-none', color)}>
      {method}
    </span>
  )
}


// ---- Inline rename input ----

export function RenameInput({
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
        'flex-1 min-w-0 outline-none text-xs rounded-sm px-1',
        'bg-[var(--color-input)] border border-[var(--color-accent)]',
        'text-[var(--color-foreground)]',
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
  onMoveFile?: (srcPath: string, destDir: string) => void
  activeFilePath?: string
  dirtyFilePaths?: Set<string>
  renamingPath?: string | null
  onRenameEnd?: () => void
  onContextMenu?: (e: React.MouseEvent, node: FileNode) => void
  collapsedPaths: Set<string>
  onToggleCollapse: (path: string) => void
}

export function TreeNode({
  node, depth,
  onSelect, onDelete, onDuplicateFile, onDeleteFolder, onNewRequestInFolder, onCreateFolder,
  onRenameFile, onRenameFolder, onMoveFile,
  activeFilePath, dirtyFilePaths,
  renamingPath, onRenameEnd, onContextMenu,
  collapsedPaths, onToggleCollapse,
}: TreeNodeProps) {
  const [localRenaming, setLocalRenaming] = useState(false)
  const [isDragOver, setIsDragOver] = useState(false)
  const expanded = !collapsedPaths.has(node.path)
  const indent = depth * 12

  const isRenaming = localRenaming || renamingPath === node.path

  function endRename() {
    setLocalRenaming(false)
    onRenameEnd?.()
  }

  if (node.isDir) {
    const isEmpty = !node.children?.some(c => !c.name.startsWith('.'))
    return (
      <div
        onDragOver={e => { e.preventDefault(); e.stopPropagation(); e.dataTransfer.dropEffect = 'move'; setIsDragOver(true) }}
        onDragLeave={e => { if (!e.currentTarget.contains(e.relatedTarget as Node)) setIsDragOver(false) }}
        onDrop={e => {
          e.preventDefault(); e.stopPropagation()
          setIsDragOver(false)
          const src = e.dataTransfer.getData('application/x-clrsigrequest')
          if (src && src !== node.path) onMoveFile?.(src, node.path)
        }}
      >
        <div
          className={cn(
            'group flex items-center hover:bg-[var(--color-muted)] transition-colors',
            isDragOver && 'bg-[var(--color-accent)]/10 outline outline-1 outline-[var(--color-accent)]/40',
          )}
          style={{ paddingLeft: `${8 + indent}px` }}
          onContextMenu={e => { e.preventDefault(); e.stopPropagation(); onContextMenu?.(e, node) }}
        >
          <button
            type="button"
            onClick={() => onToggleCollapse(node.path)}
            className="flex items-center gap-1 py-0.5 shrink-0 text-[var(--color-muted-foreground)] group-hover:text-[var(--color-foreground)]"
          >
            <span className="shrink-0">
              {expanded ? <ChevronDown size={11} /> : <ChevronRight size={11} />}
            </span>
            <span className="shrink-0 text-amber-500">
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
            onMoveFile={onMoveFile}
            activeFilePath={activeFilePath}
            dirtyFilePaths={dirtyFilePaths}
            renamingPath={renamingPath}
            onRenameEnd={onRenameEnd}
            onContextMenu={onContextMenu}
            collapsedPaths={collapsedPaths}
            onToggleCollapse={onToggleCollapse}
          />
        ))}
      </div>
    )
  }

  const isActive = node.path === activeFilePath
  const displayName = node.name.replace(/\.(yaml|yml|json)$/i, '')

  return (
    <div
      draggable
      onDragStart={e => {
        e.dataTransfer.setData('application/x-clrsigrequest', node.path)
        e.dataTransfer.effectAllowed = 'move'
      }}
      className={cn(
        'group flex items-center gap-1.5 py-0.5 text-xs transition-colors cursor-grab active:cursor-grabbing',
        isActive
          ? 'bg-[var(--color-muted)] text-[var(--color-foreground)] border-l-2 border-[var(--color-accent)]'
          : 'text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-muted)] border-l-2 border-transparent',
      )}
      style={{ paddingLeft: `${6 + indent + 15}px` }}
      onContextMenu={e => { e.preventDefault(); e.stopPropagation(); onContextMenu?.(e, node) }}
    >
      <span className="w-9 shrink-0 flex items-center">
        {node.method
          ? <span className={cn('font-mono text-[9px] font-bold', METHOD_COLORS[node.method] ?? 'text-[var(--color-muted-foreground)]')}>{node.method.slice(0, 4)}</span>
          : <span className="text-[var(--color-muted-foreground)] text-[9px] font-mono">—</span>
        }
      </span>
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
