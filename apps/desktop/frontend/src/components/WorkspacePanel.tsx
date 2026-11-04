import { useState } from 'react'
import { ChevronRight, ChevronDown, FileText, Folder, FolderOpen } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { FileNode } from '@/lib/bridge'

type TreeNodeProps = {
  node: FileNode
  depth: number
  onSelect: (path: string) => void
}

function TreeNode({ node, depth, onSelect }: TreeNodeProps) {
  const [expanded, setExpanded] = useState(true)
  const indent = depth * 12

  if (node.isDir) {
    return (
      <div>
        <button
          type="button"
          onClick={() => setExpanded(e => !e)}
          className={cn(
            'w-full flex items-center gap-1 px-2 py-0.5 text-left',
            'text-xs text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)]',
            'hover:bg-[var(--color-muted)] transition-colors',
          )}
          style={{ paddingLeft: `${8 + indent}px` }}
        >
          <span className="shrink-0 text-[var(--color-muted-foreground)]">
            {expanded ? <ChevronDown size={11} /> : <ChevronRight size={11} />}
          </span>
          <span className="shrink-0 text-[var(--color-muted-foreground)]">
            {expanded ? <FolderOpen size={12} /> : <Folder size={12} />}
          </span>
          <span className="truncate">{node.name}</span>
        </button>
        {expanded && node.children?.map(child => (
          <TreeNode key={child.path} node={child} depth={depth + 1} onSelect={onSelect} />
        ))}
      </div>
    )
  }

  const displayName = node.name.replace(/\.(yaml|yml|json)$/i, '')

  return (
    <button
      type="button"
      onClick={() => onSelect(node.path)}
      className={cn(
        'w-full flex items-center gap-1.5 px-2 py-0.5 text-left',
        'text-xs text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)]',
        'hover:bg-[var(--color-muted)] transition-colors',
      )}
      style={{ paddingLeft: `${8 + indent + 15}px` }}
    >
      <FileText size={11} className="shrink-0" />
      <span className="truncate" title={node.name}>{displayName}</span>
    </button>
  )
}

type WorkspacePanelProps = {
  workspacePath: string
  fileTree: FileNode[]
  onOpenWorkspace: () => void
  onNewWorkspace: () => void
  onSelectFile: (path: string) => void
}

export function WorkspacePanel({
  workspacePath,
  fileTree,
  onOpenWorkspace,
  onNewWorkspace,
  onSelectFile,
}: WorkspacePanelProps) {
  const folderName = workspacePath
    ? workspacePath.replace(/\\/g, '/').split('/').filter(Boolean).pop()
    : null

  return (
    <div className="flex flex-col h-full">
      {workspacePath ? (
        <>
          <div className="px-2 py-1.5 border-b border-[var(--color-border)] flex items-center gap-1 justify-between">
            <span className="text-[10px] font-mono text-[var(--color-muted-foreground)] truncate" title={workspacePath}>
              {folderName}
            </span>
            <button
              type="button"
              onClick={onOpenWorkspace}
              className="shrink-0 text-[10px] text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors"
              title="Change workspace"
            >
              ⌁
            </button>
          </div>
          <div className="flex-1 overflow-auto py-1">
            {fileTree.length === 0 ? (
              <p className="px-3 py-2 text-xs text-[var(--color-muted-foreground)]">
                No request files found.
              </p>
            ) : (
              fileTree.map(node => (
                <TreeNode key={node.path} node={node} depth={0} onSelect={onSelectFile} />
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
