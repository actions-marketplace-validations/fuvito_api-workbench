import { useState } from 'react'
import {
  ChevronRight, ChevronDown, FileText, Folder, FolderOpen,
  FolderSymlink, FolderPlus, FilePlus, Upload, Trash2,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import type { FileNode } from '@/lib/bridge'

type TreeNodeProps = {
  node: FileNode
  depth: number
  onSelect: (path: string) => void
  onDelete?: (path: string) => void
  onDeleteFolder?: (path: string) => void
  onNewRequestInFolder?: (folderPath: string) => void
  onCreateFolder?: (parentPath: string) => void
  activeFilePath?: string
  isDirty?: boolean
}

function TreeNode({
  node, depth,
  onSelect, onDelete, onDeleteFolder, onNewRequestInFolder, onCreateFolder,
  activeFilePath, isDirty,
}: TreeNodeProps) {
  const [expanded, setExpanded] = useState(true)
  const indent = depth * 12

  if (node.isDir) {
    const isEmpty = !node.children || node.children.length === 0
    return (
      <div>
        <div
          className="group flex items-center hover:bg-[var(--color-muted)] transition-colors"
          style={{ paddingLeft: `${8 + indent}px` }}
        >
          <button
            type="button"
            onClick={() => setExpanded(e => !e)}
            className="flex items-center gap-1 flex-1 min-w-0 py-0.5 text-left text-xs text-[var(--color-muted-foreground)] group-hover:text-[var(--color-foreground)]"
          >
            <span className="shrink-0">
              {expanded ? <ChevronDown size={11} /> : <ChevronRight size={11} />}
            </span>
            <span className="shrink-0">
              {expanded ? <FolderOpen size={12} /> : <Folder size={12} />}
            </span>
            <span className="truncate">{node.name}</span>
          </button>
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
        </div>
        {expanded && node.children?.map(child => (
          <TreeNode
            key={child.path}
            node={child}
            depth={depth + 1}
            onSelect={onSelect}
            onDelete={onDelete}
            onDeleteFolder={onDeleteFolder}
            onNewRequestInFolder={onNewRequestInFolder}
            onCreateFolder={onCreateFolder}
            activeFilePath={activeFilePath}
            isDirty={isDirty}
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
    >
      <button
        type="button"
        onClick={() => onSelect(node.path)}
        className="flex items-center gap-1.5 flex-1 min-w-0 text-left"
      >
        <FileText size={11} className="shrink-0" />
        <span className="truncate flex-1" title={node.name}>{displayName}</span>
        {isActive && isDirty && (
          <span className="shrink-0 w-1.5 h-1.5 rounded-full bg-amber-400" title="Unsaved changes" />
        )}
      </button>
      {onDelete && (
        <button
          type="button"
          onClick={e => { e.stopPropagation(); onDelete(node.path) }}
          title="Delete request"
          className="shrink-0 mr-1 p-0.5 rounded opacity-0 group-hover:opacity-100 transition-opacity text-[var(--color-muted-foreground)] hover:text-red-400"
        >
          <Trash2 size={10} />
        </button>
      )}
    </div>
  )
}

type WorkspacePanelProps = {
  workspacePath: string
  fileTree: FileNode[]
  onOpenWorkspace: () => void
  onNewWorkspace: () => void
  onSelectFile: (path: string) => void
  onDeleteFile?: (path: string) => void
  onDeleteFolder?: (path: string) => void
  onNewRequestInFolder?: (folderPath: string) => void
  onCreateFolder?: (parentPath: string) => void
  onImportOpenAPI: () => void
  importing?: boolean
  activeFilePath?: string
  isDirty?: boolean
}

export function WorkspacePanel({
  workspacePath,
  fileTree,
  onOpenWorkspace,
  onNewWorkspace,
  onSelectFile,
  onDeleteFile,
  onDeleteFolder,
  onNewRequestInFolder,
  onCreateFolder,
  onImportOpenAPI,
  importing,
  activeFilePath,
  isDirty,
}: WorkspacePanelProps) {
  const folderName = workspacePath
    ? workspacePath.replace(/\\/g, '/').split('/').filter(Boolean).pop()
    : null

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
          <div className="flex-1 overflow-auto py-1">
            {fileTree.length === 0 ? (
              <p className="px-3 py-2 text-xs text-[var(--color-muted-foreground)]">
                No request files found.
              </p>
            ) : (
              fileTree.map(node => (
                <TreeNode
                  key={node.path}
                  node={node}
                  depth={0}
                  onSelect={onSelectFile}
                  onDelete={onDeleteFile}
                  onDeleteFolder={onDeleteFolder}
                  onNewRequestInFolder={onNewRequestInFolder}
                  onCreateFolder={onCreateFolder}
                  activeFilePath={activeFilePath}
                  isDirty={isDirty}
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
    </div>
  )
}
