import { useState } from 'react'
import { Search, X } from 'lucide-react'
import type { FileNode, ScenarioInfo, FolderRunItem, TemplateInfo } from '@/lib/bridge'
import { listTemplates, createFromTemplate, openInExplorer } from '@/lib/bridge'
import type { RunAllState } from '@/components/Sidebar'
import { ContextMenu } from '@/components/ContextMenu'
import { ScenarioTemplateGallery } from '@/components/ScenarioTemplateGallery'
import { TreeNode } from './WorkspaceFileTree'
import { buildFileMenuItems, buildScenarioMenuItems } from './workspaceMenus'
import { WorkspacePanelHeader } from './WorkspacePanelHeader'
import { WorkspaceEmptyState } from './WorkspaceEmptyState'
import { FolderRunResults, RunAllResults } from './WorkspaceRunResults'
import { WorkspaceScenarioSection } from './WorkspaceScenarioSection'

function filterTree(nodes: FileNode[], query: string): FileNode[] {
  if (!query) return nodes
  const q = query.toLowerCase()
  return nodes.flatMap(node => {
    if (node.isDir) {
      const filtered = filterTree(node.children ?? [], q)
      if (filtered.length > 0) return [{ ...node, children: filtered }]
      if (node.name.toLowerCase().includes(q)) return [{ ...node }]
      return []
    }
    return node.name.replace(/\.(yaml|yml|json)$/i, '').toLowerCase().includes(q) ? [node] : []
  })
}

export interface FolderRunState {
  folderPath: string
  folderName: string
  running: boolean
  results: FolderRunItem[]
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
  onNewScenarioFile?: (path: string, name: string) => void
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
  onImportPostman?: () => void
  onImportInsomnia?: () => void
  onTryDemo?: () => void
  demoStarting?: boolean
  onRefresh?: () => void
  importing?: boolean
  activeFilePath?: string
  dirtyFilePaths?: Set<string>
  scenarioError?: string | null
  onRunFolder?: (folderPath: string) => void
  folderRunState?: FolderRunState | null
  onDismissFolderRun?: () => void
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
  onNewScenarioFile,
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
  onImportPostman,
  onImportInsomnia,
  onTryDemo,
  demoStarting,
  onRefresh,
  importing,
  activeFilePath,
  dirtyFilePaths,
  scenarioError,
  onRunFolder,
  folderRunState,
  onDismissFolderRun,
}: WorkspacePanelProps) {
  const folderName = workspacePath
    ? workspacePath.replace(/\\/g, '/').split('/').filter(Boolean).pop()
    : null
  const [searchQuery, setSearchQuery] = useState('')
  const filteredTree = filterTree(fileTree, searchQuery)
  const [collapsedPaths, setCollapsedPaths] = useState<Set<string>>(new Set())
  const [showTemplates, setShowTemplates] = useState(false)
  const [templates, setTemplates] = useState<TemplateInfo[]>([])
  const [templatesLoading, setTemplatesLoading] = useState(false)

  async function openTemplateGallery() {
    setShowTemplates(true)
    if (templates.length === 0) {
      setTemplatesLoading(true)
      const list = await listTemplates()
      setTemplates(list)
      setTemplatesLoading(false)
    }
  }

  async function handleUseTemplate(templateID: string, name: string) {
    setShowTemplates(false)
    const result = await createFromTemplate(templateID, name)
    if (!result.error && result.path) {
      onNewScenarioFile?.(result.path, name)
    }
  }
  const [renamingScenarioPath, setRenamingScenarioPath] = useState<string | null>(null)

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

  return (
    <div className="flex flex-col h-full">
      {workspacePath ? (
        <>
          <WorkspacePanelHeader
            folderName={folderName ?? ''}
            workspacePath={workspacePath}
            importing={importing}
            onCreateFolder={onCreateFolder}
            onImportOpenAPI={onImportOpenAPI}
            onImportPostman={onImportPostman}
            onImportInsomnia={onImportInsomnia}
            onRefresh={onRefresh}
            onOpenWorkspace={onOpenWorkspace}
          />
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
            {folderRunState && (
              <FolderRunResults state={folderRunState} onSelectFile={onSelectFile} onDismiss={onDismissFolderRun} />
            )}

            <WorkspaceScenarioSection
              scenarios={scenarios}
              scenarioError={scenarioError}
              runAllState={runAllState}
              onRunAllScenarios={onRunAllScenarios}
              onNewScenario={onNewScenario}
              onSelectScenario={onSelectScenario}
              onDuplicateScenario={onDuplicateScenario}
              onDeleteScenario={onDeleteScenario}
              onRenameScenario={onRenameScenario}
              onOpenTemplateGallery={openTemplateGallery}
              renamingScenarioPath={renamingScenarioPath}
              onRenamingChange={setRenamingScenarioPath}
              onScenarioContextMenu={(e, path, name) => setScenarioCtxMenu({ x: e.clientX, y: e.clientY, path, name })}
            />

            {runAllState && (
              <RunAllResults state={runAllState} onSelectScenario={onSelectScenario} onDismiss={onDismissRunAll} />
            )}
          </div>
        </>
      ) : (
        <WorkspaceEmptyState
          onTryDemo={onTryDemo}
          demoStarting={demoStarting}
          onNewWorkspace={onNewWorkspace}
          onOpenWorkspace={onOpenWorkspace}
        />
      )}

      {ctxMenu && (
        <ContextMenu
          x={ctxMenu.x}
          y={ctxMenu.y}
          items={buildFileMenuItems(ctxMenu.node, {
            onSelectFile,
            onDuplicateFile,
            onDeleteFile,
            onDeleteFolder,
            onNewRequestInFolder,
            onCreateFolder,
            onRunFolder,
            onRename: setRenamingPath,
            onOpenInExplorer: handleOpenInExplorer,
          })}
          onClose={() => setCtxMenu(null)}
        />
      )}

      {scenarioCtxMenu && (
        <ContextMenu
          x={scenarioCtxMenu.x}
          y={scenarioCtxMenu.y}
          items={buildScenarioMenuItems(scenarioCtxMenu, {
            onOpen: (path, name) => onSelectScenario?.(path, name),
            onDuplicate: onDuplicateScenario,
            onRename: setRenamingScenarioPath,
            onDelete: onDeleteScenario,
          })}
          onClose={() => setScenarioCtxMenu(null)}
        />
      )}

      {showTemplates && (
        <ScenarioTemplateGallery
          templates={templates}
          loading={templatesLoading}
          onUse={handleUseTemplate}
          onClose={() => setShowTemplates(false)}
        />
      )}
    </div>
  )
}
