import { useState, useRef, useEffect } from 'react'
import {
  ChevronRight, ChevronDown, FileText, Folder, FolderOpen,
  FolderSymlink, FolderPlus, FilePlus, Upload, Trash2, CopyPlus, Search, X, RefreshCw, Play, Plus, PlayCircle, CheckCircle2, XCircle, Loader2,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import type { FileNode, ScenarioInfo } from '@/lib/bridge'
import type { RunAllState } from '@/components/Sidebar'
import { openInExplorer } from '@/lib/bridge'
import { ContextMenu, type ContextMenuAction } from '@/components/ContextMenu'

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

function MethodBadge({ method }: { method: string }) {
  const color = METHOD_COLORS[method] ?? 'text-[var(--color-muted-foreground)]'
  return (
    <span className={cn('shrink-0 font-mono text-[9px] font-semibold leading-none', color)}>
      {method}
    </span>
  )
}

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

function TreeNode({
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
          {node.method && <MethodBadge method={node.method} />}
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
  scenarios?: ScenarioInfo[]
  onOpenWorkspace: () => void
  onNewWorkspace: () => void
  onSelectFile: (path: string) => void
  onSelectScenario?: (path: string, name: string) => void
  onNewScenario?: (name: string) => void
  onDuplicateScenario?: (path: string) => void
  onDeleteScenario?: (path: string) => void
  onRenameScenario?: (path: string, newName: string) => void
  runAllState?: RunAllState
  onRunAllScenarios?: () => void
  onDismissRunAll?: () => void
  onDeleteFile?: (path: string) => void
  onDuplicateFile?: (path: string) => void
  onDeleteFolder?: (path: string) => void
  onNewRequestInFolder?: (folderPath: string) => void
  onCreateFolder?: (parentPath: string) => void
  onRenameFile?: (path: string, newName: string) => void
  onRenameFolder?: (path: string, newName: string) => void
  onMoveFile?: (srcPath: string, destDir: string) => void
  onImportOpenAPI: () => void
  onRefresh?: () => void
  importing?: boolean
  activeFilePath?: string
  dirtyFilePaths?: Set<string>
  scenarioError?: string | null
}

export function WorkspacePanel({
  workspacePath,
  fileTree,
  scenarios = [],
  onOpenWorkspace,
  onNewWorkspace,
  onSelectFile,
  onSelectScenario,
  onNewScenario,
  onDuplicateScenario,
  onDeleteScenario,
  onRenameScenario,
  runAllState,
  onRunAllScenarios,
  onDismissRunAll,
  onDeleteFile,
  onDuplicateFile,
  onDeleteFolder,
  onNewRequestInFolder,
  onCreateFolder,
  onRenameFile,
  onRenameFolder,
  onMoveFile,
  onImportOpenAPI,
  onRefresh,
  importing,
  activeFilePath,
  dirtyFilePaths,
  scenarioError,
}: WorkspacePanelProps) {
  const folderName = workspacePath
    ? workspacePath.replace(/\\/g, '/').split('/').filter(Boolean).pop()
    : null
  const [searchQuery, setSearchQuery] = useState('')
  const filteredTree = filterTree(fileTree, searchQuery)
  const [collapsedPaths, setCollapsedPaths] = useState<Set<string>>(new Set())
  const [creatingScenario, setCreatingScenario] = useState(false)
  const [renamingScenarioPath, setRenamingScenarioPath] = useState<string | null>(null)
  const [scenarioQuery, setScenarioQuery] = useState('')
  const filteredScenarios = scenarioQuery
    ? scenarios.filter(s => s.title.toLowerCase().includes(scenarioQuery.toLowerCase()))
    : scenarios

  function handleToggleCollapse(path: string) {
    setCollapsedPaths(prev => {
      const next = new Set(prev)
      if (next.has(path)) { next.delete(path) } else { next.add(path) }
      return next
    })
  }

  // ---- Context menu ----
  const [ctxMenu, setCtxMenu] = useState<{ x: number; y: number; node: FileNode } | null>(null)
  const [scenarioCtxMenu, setScenarioCtxMenu] = useState<{ x: number; y: number; path: string; name: string } | null>(null)
  const [renamingPath, setRenamingPath] = useState<string | null>(null)
  const [explorerError, setExplorerError] = useState<string | null>(null)

  async function handleOpenInExplorer(path: string) {
    const err = await openInExplorer(path)
    if (err) {
      setExplorerError(err)
      setTimeout(() => setExplorerError(null), 4000)
    }
  }

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
        { kind: 'action', label: 'Open in Explorer', onClick: () => handleOpenInExplorer(node.path) },
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
          {explorerError && (
            <div className="px-2 py-1 text-[10px] text-red-400 bg-red-500/10 border-b border-red-500/20 truncate" title={explorerError}>
              {explorerError}
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
                  onMoveFile={onMoveFile}
                  activeFilePath={activeFilePath}
                  dirtyFilePaths={dirtyFilePaths}
                  renamingPath={renamingPath}
                  onRenameEnd={() => setRenamingPath(null)}
                  onContextMenu={handleContextMenu}
                  collapsedPaths={collapsedPaths}
                  onToggleCollapse={handleToggleCollapse}
                />
              ))
            )}
            {scenarioError && (
              <div className="mx-2 mt-2 px-2 py-1 text-[10px] text-red-400 bg-red-500/10 border border-red-500/20 rounded truncate" title={scenarioError}>
                {scenarioError}
              </div>
            )}
            {(scenarios.length > 0 || onNewScenario) && (
              <>
                <div className="px-2 pt-3 pb-1 flex items-center justify-between">
                  <span className="text-[10px] font-semibold uppercase tracking-widest text-[var(--color-muted-foreground)]">
                    Scenarios
                  </span>
                  <div className="flex items-center gap-1">
                    {onRunAllScenarios && scenarios.length > 0 && (
                      <button
                        type="button"
                        onClick={onRunAllScenarios}
                        disabled={runAllState?.running}
                        title="Run all scenarios"
                        className="text-[var(--color-muted-foreground)] hover:text-violet-400 transition-colors disabled:opacity-40"
                      >
                        {runAllState?.running
                          ? <Loader2 size={11} className="animate-spin" />
                          : <PlayCircle size={11} />}
                      </button>
                    )}
                    {onNewScenario && (
                      <button
                        type="button"
                        onClick={() => setCreatingScenario(true)}
                        title="New scenario"
                        className="text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors"
                      >
                        <Plus size={11} />
                      </button>
                    )}
                  </div>
                </div>
                {scenarios.length > 0 && (
                  <div className="flex items-center gap-1.5 px-2 py-1 border-t border-[var(--color-border)]">
                    <Search size={10} className="shrink-0 text-[var(--color-muted-foreground)]" />
                    <input
                      type="text"
                      value={scenarioQuery}
                      onChange={e => setScenarioQuery(e.target.value)}
                      placeholder="Filter scenarios…"
                      className="flex-1 min-w-0 bg-transparent text-xs outline-none text-[var(--color-foreground)] placeholder:text-[var(--color-muted-foreground)]"
                    />
                    {scenarioQuery && (
                      <button type="button" onClick={() => setScenarioQuery('')} className="shrink-0 text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)]">
                        <X size={10} />
                      </button>
                    )}
                  </div>
                )}
                {creatingScenario && (
                  <div className="flex items-center gap-1.5 px-2 py-0.5">
                    <Play size={10} className="shrink-0 text-violet-400" />
                    <RenameInput
                      initialValue=""
                      onConfirm={name => {
                        setCreatingScenario(false)
                        onNewScenario?.(name)
                      }}
                      onCancel={() => setCreatingScenario(false)}
                    />
                  </div>
                )}
                {filteredScenarios.length === 0 && scenarioQuery && (
                  <p className="px-3 py-1 text-xs text-[var(--color-muted-foreground)]">No matches.</p>
                )}
                {filteredScenarios.map(s => (
                  <div
                    key={s.path}
                    className={cn(
                      'group flex items-center gap-1.5 px-2 py-0.5 text-xs transition-colors',
                      'text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-muted)]',
                    )}
                  >
                    <Play size={10} className="shrink-0 text-violet-400" />
                    {renamingScenarioPath === s.path ? (
                      <RenameInput
                        initialValue={s.title}
                        onConfirm={newName => {
                          setRenamingScenarioPath(null)
                          onRenameScenario?.(s.path, newName)
                        }}
                        onCancel={() => setRenamingScenarioPath(null)}
                      />
                    ) : (
                      <button
                        type="button"
                        className="flex-1 min-w-0 flex items-center gap-1.5 text-left py-0.5 min-w-0"
                        onClick={() => onSelectScenario?.(s.path, s.title)}
                        onDoubleClick={e => { e.stopPropagation(); setRenamingScenarioPath(s.path) }}
                        onContextMenu={e => { e.preventDefault(); e.stopPropagation(); setScenarioCtxMenu({ x: e.clientX, y: e.clientY, path: s.path, name: s.title }) }}
                        title="Click to open · Double-click to rename"
                      >
                        <span className="truncate flex-1">{s.title}</span>
                        {!!s.stepCount && (
                          <span className="shrink-0 text-[10px] tabular-nums text-[var(--color-muted-foreground)]">
                            {s.stepCount}
                          </span>
                        )}
                      </button>
                    )}
                    {!renamingScenarioPath && (onDuplicateScenario || onDeleteScenario) && (
                      <div className="flex items-center shrink-0 opacity-0 group-hover:opacity-100 transition-opacity">
                        {onDuplicateScenario && (
                          <button
                            type="button"
                            onClick={e => { e.stopPropagation(); onDuplicateScenario(s.path) }}
                            title="Duplicate scenario"
                            className="p-0.5 rounded text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)]"
                          >
                            <CopyPlus size={10} />
                          </button>
                        )}
                        {onDeleteScenario && (
                          <button
                            type="button"
                            onClick={e => { e.stopPropagation(); onDeleteScenario(s.path) }}
                            title="Delete scenario"
                            className="p-0.5 rounded text-[var(--color-muted-foreground)] hover:text-red-400"
                          >
                            <Trash2 size={10} />
                          </button>
                        )}
                      </div>
                    )}
                  </div>
                ))}
              </>
            )}

            {runAllState && (
              <div className="mx-2 mt-2 mb-1 rounded border border-[var(--color-border)] bg-[var(--color-muted)] text-xs overflow-hidden">
                <div className="flex items-center justify-between px-2 py-1.5 border-b border-[var(--color-border)]">
                  <span className="font-semibold text-[var(--color-foreground)]">
                    {runAllState.running
                      ? <span className="flex items-center gap-1.5"><Loader2 size={10} className="animate-spin" /> Running {runAllState.done + 1}/{runAllState.total}…</span>
                      : `${runAllState.passed}/${runAllState.total} passed`}
                  </span>
                  {!runAllState.running && onDismissRunAll && (
                    <button type="button" onClick={onDismissRunAll} className="text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)]">
                      <X size={10} />
                    </button>
                  )}
                </div>
                {runAllState.results.map(r => (
                  <button
                    key={r.path}
                    type="button"
                    onClick={() => onSelectScenario?.(r.path, r.name)}
                    className="w-full flex items-center gap-1.5 px-2 py-1 text-left hover:bg-[var(--color-border)] transition-colors"
                  >
                    {r.passed
                      ? <CheckCircle2 size={10} className="shrink-0 text-emerald-400" />
                      : <XCircle size={10} className="shrink-0 text-red-400" />}
                    <span className="flex-1 truncate text-[var(--color-foreground)]">{r.name}</span>
                    <span className="shrink-0 text-[var(--color-muted-foreground)] text-[10px]">{r.durationMs}ms</span>
                  </button>
                ))}
              </div>
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

      {scenarioCtxMenu && (
        <ContextMenu
          x={scenarioCtxMenu.x}
          y={scenarioCtxMenu.y}
          items={[
            { kind: 'action', label: 'Open', onClick: () => { onSelectScenario?.(scenarioCtxMenu.path, scenarioCtxMenu.name); setScenarioCtxMenu(null) } },
            ...(onDuplicateScenario ? [{ kind: 'action' as const, label: 'Duplicate', onClick: () => { onDuplicateScenario(scenarioCtxMenu.path); setScenarioCtxMenu(null) } }] : []),
            { kind: 'separator' as const },
            { kind: 'action', label: 'Rename', onClick: () => { setRenamingScenarioPath(scenarioCtxMenu.path); setScenarioCtxMenu(null) } },
            { kind: 'action', label: 'Open in Explorer', onClick: () => { openInExplorer(scenarioCtxMenu.path); setScenarioCtxMenu(null) } },
            { kind: 'separator' as const },
            { kind: 'action', label: 'Delete', danger: true, disabled: !onDeleteScenario, onClick: () => { onDeleteScenario?.(scenarioCtxMenu.path); setScenarioCtxMenu(null) } },
          ]}
          onClose={() => setScenarioCtxMenu(null)}
        />
      )}
    </div>
  )
}
