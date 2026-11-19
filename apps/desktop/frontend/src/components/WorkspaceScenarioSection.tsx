import { useState } from 'react'
import { Search, X, Play, FilePlus, FolderPlus, PlayCircle, Loader2, LayoutTemplate, CopyPlus, Trash2, ChevronDown, ChevronRight, Folder, FolderOpen, RefreshCw } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { ScenarioInfo } from '@/lib/bridge'
import type { RunAllState } from '@/types'
import type { FileClipboard } from '@/types'
import { RenameInput } from './WorkspaceFileTree'

// ── Tree data model ───────────────────────────────────────────────────────────

type FolderNode = {
  kind: 'folder'
  name: string
  relPath: string   // relative to {workspace}/scenarios/, e.g. "auth" or "auth/sub"
  children: TreeNode[]
}

type ScenarioNode = {
  kind: 'scenario'
  info: ScenarioInfo
}

type TreeNode = FolderNode | ScenarioNode

function buildTree(scenarios: ScenarioInfo[]): TreeNode[] {
  const root: TreeNode[] = []
  const folderMap = new Map<string, FolderNode>()

  function getOrCreate(relPath: string): FolderNode {
    if (folderMap.has(relPath)) return folderMap.get(relPath)!
    const parts = relPath.split('/')
    const name = parts[parts.length - 1]
    const parentPath = parts.slice(0, -1).join('/')
    const node: FolderNode = { kind: 'folder', name, relPath, children: [] }
    folderMap.set(relPath, node)
    if (parentPath === '') {
      root.push(node)
    } else {
      getOrCreate(parentPath).children.push(node)
    }
    return node
  }

  // Pre-seed folders from sentinels (sorted by depth so parents are created first).
  const folderSentinels = scenarios.filter(s => s.isFolder && s.folder)
  folderSentinels.sort((a, b) => (a.folder!.split('/').length) - (b.folder!.split('/').length))
  for (const f of folderSentinels) {
    getOrCreate(f.folder!)
  }

  for (const s of scenarios) {
    if (s.isFolder) continue
    const folder = s.folder ?? ''
    const target = folder === '' ? root : getOrCreate(folder).children
    target.push({ kind: 'scenario', info: s })
  }

  return root
}

// ── Props ─────────────────────────────────────────────────────────────────────

type Props = {
  scenarios: ScenarioInfo[]
  scenarioError?: string | null
  runAllState?: RunAllState
  onRunAllScenarios?: () => void
  onNewScenario?: (name: string, folderPath?: string) => void
  onNewFolder?: (parentFolderPath: string, name: string) => void
  onSelectScenario?: (path: string, name: string) => void
  onDuplicateScenario?: (path: string) => void
  onDeleteScenario?: (path: string) => void
  onRenameScenario?: (path: string, newName: string) => void
  onDeleteFolder?: (folderPath: string) => void
  onRenameFolder?: (folderPath: string, newName: string) => void
  onRunFolder?: (folderPath: string) => void
  onOpenTemplateGallery: () => void
  onRefresh?: () => void
  onMoveScenario?: (srcPath: string, destFolderRelPath: string) => void
  clipboard?: FileClipboard | null
  // the scenario open in the active tab, and scenarios with unsaved edits, like the requests tree
  activeScenarioPath?: string | null
  dirtyScenarioPaths?: Set<string>
  renamingScenarioPath: string | null
  onRenamingChange: (path: string | null) => void
  renamingFolderPath: string | null
  onRenamingFolderChange: (path: string | null) => void
  onScenarioContextMenu: (e: React.MouseEvent, path: string, name: string) => void
  onFolderContextMenu?: (e: React.MouseEvent, folderPath: string, name: string) => void
}

// ── Scenario row ──────────────────────────────────────────────────────────────

function ScenarioRow({
  s, indent,
  renamingScenarioPath, onRenamingChange,
  onSelectScenario, onDuplicateScenario, onDeleteScenario, onRenameScenario,
  onScenarioContextMenu, clipboard, activeScenarioPath, dirtyScenarioPaths,
}: {
  s: ScenarioInfo
  indent: number
  renamingScenarioPath: string | null
  onRenamingChange: (path: string | null) => void
  onSelectScenario?: (path: string, name: string) => void
  onDuplicateScenario?: (path: string) => void
  onDeleteScenario?: (path: string) => void
  onRenameScenario?: (path: string, newName: string) => void
  onScenarioContextMenu: (e: React.MouseEvent, path: string, name: string) => void
  clipboard?: FileClipboard | null
  activeScenarioPath?: string | null
  dirtyScenarioPaths?: Set<string>
}) {
  const isCut = clipboard?.op === 'cut' && clipboard.srcPath === s.path
  const isActive = !!activeScenarioPath && activeScenarioPath === s.path
  return (
    <div
      draggable
      onDragStart={e => {
        e.dataTransfer.setData('application/x-clrsigscenario', s.path)
        e.dataTransfer.effectAllowed = 'move'
      }}
      className={cn(
        'group flex items-center gap-1.5 pr-2 py-0.5 text-xs transition-colors cursor-grab active:cursor-grabbing border-l-2',
        isActive
          ? 'bg-[var(--color-muted)] text-[var(--color-foreground)] border-[var(--color-accent)]'
          : 'text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-muted)] border-transparent',
        isCut && 'opacity-40 italic',
      )}
      style={{ paddingLeft: `${6 + indent * 12 + 12}px` }}
    >
      <Play size={12} className="ui-icon shrink-0 text-violet-400" />
      {renamingScenarioPath === s.path ? (
        <RenameInput
          initialValue={s.title}
          onConfirm={newName => { onRenamingChange(null); onRenameScenario?.(s.path, newName) }}
          onCancel={() => onRenamingChange(null)}
        />
      ) : (
        <button
          type="button"
          className="flex-1 min-w-0 flex items-center gap-1.5 text-left"
          onClick={() => onSelectScenario?.(s.path, s.title)}
          onDoubleClick={e => { e.stopPropagation(); onRenamingChange(s.path) }}
          onContextMenu={e => { e.preventDefault(); e.stopPropagation(); onScenarioContextMenu(e, s.path, s.title) }}
          title="Click to open · Double-click to rename"
        >
          <span className="truncate flex-1">{s.title}</span>
          {dirtyScenarioPaths?.has(s.path) && (
            <span className="shrink-0 w-1.5 h-1.5 rounded-full bg-amber-400" title="Unsaved changes" />
          )}
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
  )
}

// ── Folder row ────────────────────────────────────────────────────────────────

function FolderRow({
  node, indent, open, onToggle,
  renamingFolderPath, onRenamingFolderChange,
  onNewScenario, onNewFolder, onDeleteFolder, onRenameFolder, onRunFolder,
  onFolderContextMenu,
  renamingScenarioPath, onRenamingChange,
  onSelectScenario, onDuplicateScenario, onDeleteScenario, onRenameScenario,
  onScenarioContextMenu, onMoveScenario, clipboard, activeScenarioPath, dirtyScenarioPaths,
}: {
  node: FolderNode
  indent: number
  open: boolean
  onToggle: () => void
  renamingFolderPath: string | null
  onRenamingFolderChange: (path: string | null) => void
  onNewScenario?: (name: string, folderPath?: string) => void
  onNewFolder?: (parentFolderPath: string, name: string) => void
  onDeleteFolder?: (folderPath: string) => void
  onRenameFolder?: (folderPath: string, newName: string) => void
  onRunFolder?: (folderPath: string) => void
  onFolderContextMenu?: (e: React.MouseEvent, folderPath: string, name: string) => void
  renamingScenarioPath: string | null
  onRenamingChange: (path: string | null) => void
  onSelectScenario?: (path: string, name: string) => void
  onDuplicateScenario?: (path: string) => void
  onDeleteScenario?: (path: string) => void
  onRenameScenario?: (path: string, newName: string) => void
  onScenarioContextMenu: (e: React.MouseEvent, path: string, name: string) => void
  onMoveScenario?: (srcPath: string, destFolderRelPath: string) => void
  clipboard?: FileClipboard | null
  activeScenarioPath?: string | null
  dirtyScenarioPaths?: Set<string>
}) {
  const [creatingInFolder, setCreatingInFolder] = useState(false)
  const [creatingSubfolder, setCreatingSubfolder] = useState(false)
  const [isDragOver, setIsDragOver] = useState(false)

  return (
    <>
      <div
        onDragOver={e => { e.preventDefault(); e.stopPropagation(); e.dataTransfer.dropEffect = 'move'; setIsDragOver(true) }}
        onDragLeave={e => { if (!e.currentTarget.contains(e.relatedTarget as Node)) setIsDragOver(false) }}
        onDrop={e => {
          e.preventDefault(); e.stopPropagation()
          setIsDragOver(false)
          const src = e.dataTransfer.getData('application/x-clrsigscenario')
          if (src) onMoveScenario?.(src, node.relPath)
        }}
        className={cn(
          'group flex items-center gap-1.5 pr-2 py-0.5 hover:bg-[var(--color-muted)] transition-colors cursor-pointer',
          isDragOver && 'bg-[var(--color-accent)]/10 outline outline-1 outline-[var(--color-accent)]/40',
        )}
        style={{ paddingLeft: `${8 + indent * 12}px` }}
        onClick={onToggle}
        onContextMenu={e => { e.preventDefault(); e.stopPropagation(); onFolderContextMenu?.(e, node.relPath, node.name) }}
      >
        <span className="ui-icon shrink-0 text-[var(--color-muted-foreground)]">
          {open ? <ChevronDown size={11} /> : <ChevronRight size={11} />}
        </span>
        <span className="ui-icon shrink-0 text-amber-500">
          {open ? <FolderOpen size={12} /> : <Folder size={12} />}
        </span>

        {renamingFolderPath === node.relPath ? (
          <RenameInput
            initialValue={node.name}
            onConfirm={newName => { onRenamingFolderChange(null); onRenameFolder?.(node.relPath, newName) }}
            onCancel={() => onRenamingFolderChange(null)}
          />
        ) : (
          <span
            className="flex-1 min-w-0 text-xs text-[var(--color-muted-foreground)] group-hover:text-[var(--color-foreground)] truncate"
            onDoubleClick={e => { e.stopPropagation(); onRenamingFolderChange(node.relPath) }}
          >
            {node.name}
          </span>
        )}

        {!renamingFolderPath && (
          <div className="flex items-center shrink-0 gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">
            {onRunFolder && (
              <button
                type="button"
                onClick={e => { e.stopPropagation(); onRunFolder(node.relPath) }}
                title="Run all scenarios in folder"
                className="p-0.5 rounded text-[var(--color-muted-foreground)] hover:text-violet-400 transition-colors"
              >
                <PlayCircle size={10} />
              </button>
            )}
            <button
              type="button"
              onClick={e => { e.stopPropagation(); setCreatingInFolder(true); if (!open) onToggle() }}
              title="New scenario in folder"
              className="p-0.5 rounded text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)]"
            >
              <FilePlus size={10} />
            </button>
            <button
              type="button"
              onClick={e => { e.stopPropagation(); setCreatingSubfolder(true); if (!open) onToggle() }}
              title="New subfolder"
              className="p-0.5 rounded text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)]"
            >
              <FolderPlus size={10} />
            </button>
            {onDeleteFolder && (
              <button
                type="button"
                onClick={e => { e.stopPropagation(); onDeleteFolder(node.relPath) }}
                title="Delete folder"
                className="p-0.5 rounded text-[var(--color-muted-foreground)] hover:text-red-400"
              >
                <Trash2 size={10} />
              </button>
            )}
          </div>
        )}
      </div>

      {open && (
        <>
          {creatingSubfolder && (
            <div className="flex items-center gap-1.5 py-0.5" style={{ paddingLeft: `${8 + (indent + 1) * 12}px` }}>
              <Folder size={12} className="ui-icon shrink-0 text-[var(--color-muted-foreground)]" />
              <RenameInput
                initialValue=""
                onConfirm={name => { setCreatingSubfolder(false); onNewFolder?.(node.relPath, name) }}
                onCancel={() => setCreatingSubfolder(false)}
              />
            </div>
          )}
          {creatingInFolder && (
            <div className="flex items-center gap-1.5 py-0.5" style={{ paddingLeft: `${8 + (indent + 1) * 12 + 12}px` }}>
              <Play size={12} className="ui-icon shrink-0 text-violet-400" />
              <RenameInput
                initialValue=""
                onConfirm={name => { setCreatingInFolder(false); onNewScenario?.(name, node.relPath) }}
                onCancel={() => setCreatingInFolder(false)}
              />
            </div>
          )}
          {node.children.map(child =>
            child.kind === 'folder' ? (
              <FolderRowStateful
                key={child.relPath}
                node={child}
                indent={indent + 1}
                renamingFolderPath={renamingFolderPath}
                onRenamingFolderChange={onRenamingFolderChange}
                onNewScenario={onNewScenario}
                onNewFolder={onNewFolder}
                onDeleteFolder={onDeleteFolder}
                onRenameFolder={onRenameFolder}
                onRunFolder={onRunFolder}
                onFolderContextMenu={onFolderContextMenu}
                renamingScenarioPath={renamingScenarioPath}
                onRenamingChange={onRenamingChange}
                onSelectScenario={onSelectScenario}
                onDuplicateScenario={onDuplicateScenario}
                onDeleteScenario={onDeleteScenario}
                onRenameScenario={onRenameScenario}
                onScenarioContextMenu={onScenarioContextMenu}
                onMoveScenario={onMoveScenario}
                clipboard={clipboard}
                activeScenarioPath={activeScenarioPath}
                dirtyScenarioPaths={dirtyScenarioPaths}
              />
            ) : (
              <ScenarioRow
                key={child.info.path}
                s={child.info}
                indent={indent + 1}
                renamingScenarioPath={renamingScenarioPath}
                onRenamingChange={onRenamingChange}
                onSelectScenario={onSelectScenario}
                onDuplicateScenario={onDuplicateScenario}
                onDeleteScenario={onDeleteScenario}
                onRenameScenario={onRenameScenario}
                onScenarioContextMenu={onScenarioContextMenu}
                clipboard={clipboard}
                activeScenarioPath={activeScenarioPath}
                dirtyScenarioPaths={dirtyScenarioPaths}
              />
            )
          )}
        </>
      )}
    </>
  )
}

// Wrapper that owns the open/closed state so each folder is independently collapsible
function FolderRowStateful(props: Omit<React.ComponentProps<typeof FolderRow>, 'open' | 'onToggle'>) {
  const [open, setOpen] = useState(true)
  return <FolderRow {...props} open={open} onToggle={() => setOpen(o => !o)} />
}

// ── Main component ────────────────────────────────────────────────────────────

export function WorkspaceScenarioSection({
  scenarios, scenarioError, runAllState,
  onRunAllScenarios, onNewScenario, onNewFolder, onSelectScenario,
  onDuplicateScenario, onDeleteScenario, onRenameScenario,
  onDeleteFolder, onRenameFolder, onRunFolder,
  onOpenTemplateGallery, onRefresh,
  onMoveScenario, clipboard,
  renamingScenarioPath, onRenamingChange,
  renamingFolderPath, onRenamingFolderChange,
  onScenarioContextMenu, onFolderContextMenu, activeScenarioPath, dirtyScenarioPaths,
}: Props) {
  const [creatingScenario, setCreatingScenario] = useState(false)
  const [creatingFolder, setCreatingFolder] = useState(false)
  const [scenarioQuery, setScenarioQuery] = useState('')
  const [rootDragOver, setRootDragOver] = useState(false)

  const filteredScenarios = scenarioQuery
    ? scenarios.filter(s => !s.isFolder && s.title.toLowerCase().includes(scenarioQuery.toLowerCase()))
    : null  // null = show tree

  const tree = filteredScenarios === null ? buildTree(scenarios) : null

  return (
    <>
      {scenarioError && (
        <div className="mx-2 mt-2 px-2 py-1 text-[10px] text-red-400 bg-red-500/10 border border-red-500/20 rounded truncate" title={scenarioError}>
          {scenarioError}
        </div>
      )}
      {(scenarios.some(s => !s.isFolder) || onNewScenario) && (
        <>
          {/* Toolbar, like the requests panel: no root folder node, items start at the top level */}
          <div className="px-2 py-1.5 border-b border-[var(--color-border)] flex items-center gap-1.5 justify-end">
            {onRunAllScenarios && scenarios.filter(s => !s.isFolder).length > 0 && (
              <button
                type="button"
                onClick={() => onRunAllScenarios()}
                disabled={runAllState?.running}
                title="Run all scenarios"
                className="text-[var(--color-muted-foreground)] hover:text-violet-400 transition-colors disabled:opacity-40"
              >
                {runAllState?.running ? <Loader2 size={12} className="animate-spin" /> : <PlayCircle size={12} />}
              </button>
            )}
            {onRefresh && (
              <button type="button" onClick={() => onRefresh()} title="Refresh scenarios"
                className="text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors">
                <RefreshCw size={12} />
              </button>
            )}
            <button type="button" onClick={() => onOpenTemplateGallery()} title="New scenario from template"
              className="text-[var(--color-muted-foreground)] hover:text-[var(--color-accent)] transition-colors">
              <LayoutTemplate size={12} />
            </button>
            {onNewFolder && (
              <button type="button" onClick={() => setCreatingFolder(true)} title="New folder"
                className="text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors">
                <FolderPlus size={12} />
              </button>
            )}
            {onNewScenario && (
              <button type="button" onClick={() => setCreatingScenario(true)} title="New scenario"
                className="text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors">
                <FilePlus size={12} />
              </button>
            )}
          </div>

          {scenarios.some(s => !s.isFolder) && (
            <div className="flex items-center gap-1.5 px-2 py-1 border-b border-[var(--color-border)]">
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

          {/* Empty space below the items is the root: drop a scenario here to move it out of a folder */}
          <div
            data-testid="scenario-root"
            className={cn('min-h-full', rootDragOver && 'bg-[var(--color-accent)]/10 outline outline-1 outline-[var(--color-accent)]/40')}
            onDragOver={e => { e.preventDefault(); e.stopPropagation(); e.dataTransfer.dropEffect = 'move'; setRootDragOver(true) }}
            onDragLeave={e => { if (!e.currentTarget.contains(e.relatedTarget as Node)) setRootDragOver(false) }}
            onDrop={e => {
              e.preventDefault(); e.stopPropagation()
              setRootDragOver(false)
              const src = e.dataTransfer.getData('application/x-clrsigscenario')
              if (src) onMoveScenario?.(src, '')
            }}
            onContextMenu={e => { e.preventDefault(); e.stopPropagation(); onFolderContextMenu?.(e, '', 'scenarios') }}
          >
            <>

              {creatingFolder && (
                <div className="flex items-center gap-1.5 py-0.5" style={{ paddingLeft: '8px' }}>
                  <Folder size={12} className="ui-icon shrink-0 text-amber-500" />
                  <RenameInput
                    initialValue=""
                    onConfirm={name => { setCreatingFolder(false); onNewFolder?.('', name) }}
                    onCancel={() => setCreatingFolder(false)}
                  />
                </div>
              )}
              {creatingScenario && (
                <div className="flex items-center gap-1.5 py-0.5" style={{ paddingLeft: '20px' }}>
                  <Play size={12} className="ui-icon shrink-0 text-violet-400" />
                  <RenameInput
                    initialValue=""
                    onConfirm={name => { setCreatingScenario(false); onNewScenario?.(name) }}
                    onCancel={() => setCreatingScenario(false)}
                  />
                </div>
              )}

              {filteredScenarios !== null && (
                <>
                  {filteredScenarios.length === 0 && (
                    <p className="px-3 py-1 text-xs text-[var(--color-muted-foreground)]">No matches.</p>
                  )}
                  {filteredScenarios.map(s => (
                    <ScenarioRow
                      key={s.path}
                      s={s}
                      indent={0}
                      renamingScenarioPath={renamingScenarioPath}
                      onRenamingChange={onRenamingChange}
                      onSelectScenario={onSelectScenario}
                      onDuplicateScenario={onDuplicateScenario}
                      onDeleteScenario={onDeleteScenario}
                      onRenameScenario={onRenameScenario}
                      onScenarioContextMenu={onScenarioContextMenu}
                      clipboard={clipboard}
                      activeScenarioPath={activeScenarioPath}
                      dirtyScenarioPaths={dirtyScenarioPaths}
                    />
                  ))}
                </>
              )}

              {tree !== null && (
                <>
                  {tree.length === 0 && !creatingScenario && !creatingFolder && (
                    <div className="flex flex-col items-center gap-3 px-4 py-6 text-center">
                      <Play size={20} className="text-[var(--color-muted-foreground)] opacity-30" />
                      <p className="text-xs text-[var(--color-muted-foreground)]">No scenarios yet</p>
                      <div className="flex flex-col gap-1.5 w-full">
                        <button type="button" onClick={() => setCreatingScenario(true)}
                          className="w-full text-xs py-1.5 px-3 rounded border border-[var(--color-border)] text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-muted)] transition-colors">
                          New scenario
                        </button>
                        <button type="button" onClick={onOpenTemplateGallery}
                          className="w-full text-xs py-1.5 px-3 rounded border border-[var(--color-border)] text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-muted)] transition-colors">
                          Browse templates
                        </button>
                      </div>
                    </div>
                  )}
                  {tree.map(node =>
                    node.kind === 'folder' ? (
                      <FolderRowStateful
                        key={node.relPath}
                        node={node}
                        indent={0}
                        renamingFolderPath={renamingFolderPath}
                        onRenamingFolderChange={onRenamingFolderChange}
                        onNewScenario={onNewScenario}
                        onNewFolder={onNewFolder}
                        onDeleteFolder={onDeleteFolder}
                        onRenameFolder={onRenameFolder}
                        onRunFolder={onRunFolder}
                        onFolderContextMenu={onFolderContextMenu}
                        renamingScenarioPath={renamingScenarioPath}
                        onRenamingChange={onRenamingChange}
                        onSelectScenario={onSelectScenario}
                        onDuplicateScenario={onDuplicateScenario}
                        onDeleteScenario={onDeleteScenario}
                        onRenameScenario={onRenameScenario}
                        onScenarioContextMenu={onScenarioContextMenu}
                        onMoveScenario={onMoveScenario}
                        clipboard={clipboard}
                        activeScenarioPath={activeScenarioPath}
                        dirtyScenarioPaths={dirtyScenarioPaths}
                      />
                    ) : (
                      <ScenarioRow
                        key={node.info.path}
                        s={node.info}
                        indent={0}
                        renamingScenarioPath={renamingScenarioPath}
                        onRenamingChange={onRenamingChange}
                        onSelectScenario={onSelectScenario}
                        onDuplicateScenario={onDuplicateScenario}
                        onDeleteScenario={onDeleteScenario}
                        onRenameScenario={onRenameScenario}
                        onScenarioContextMenu={onScenarioContextMenu}
                        clipboard={clipboard}
                        activeScenarioPath={activeScenarioPath}
                        dirtyScenarioPaths={dirtyScenarioPaths}
                      />
                    )
                  )}
                </>
              )}
            </>
          </div>
        </>
      )}
    </>
  )
}
