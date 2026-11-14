import { useState } from 'react'
import { Search, X, Play, Plus, PlayCircle, Loader2, LayoutTemplate, CopyPlus, Trash2 } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { ScenarioInfo } from '@/lib/bridge'
import type { RunAllState } from '@/components/Sidebar'
import { RenameInput } from './WorkspaceFileTree'

type Props = {
  scenarios: ScenarioInfo[]
  scenarioError?: string | null
  runAllState?: RunAllState
  onRunAllScenarios?: () => void
  onNewScenario?: (name: string) => void
  onSelectScenario?: (path: string, name: string) => void
  onDuplicateScenario?: (path: string) => void
  onDeleteScenario?: (path: string) => void
  onRenameScenario?: (path: string, newName: string) => void
  onOpenTemplateGallery: () => void
  renamingScenarioPath: string | null
  onRenamingChange: (path: string | null) => void
  onScenarioContextMenu: (e: React.MouseEvent, path: string, name: string) => void
}

export function WorkspaceScenarioSection({
  scenarios, scenarioError, runAllState,
  onRunAllScenarios, onNewScenario, onSelectScenario,
  onDuplicateScenario, onDeleteScenario, onRenameScenario,
  onOpenTemplateGallery,
  renamingScenarioPath, onRenamingChange, onScenarioContextMenu,
}: Props) {
  const [creatingScenario, setCreatingScenario] = useState(false)
  const [scenarioQuery, setScenarioQuery] = useState('')
  const filteredScenarios = scenarioQuery
    ? scenarios.filter(s => s.title.toLowerCase().includes(scenarioQuery.toLowerCase()))
    : scenarios

  return (
    <>
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
              <button
                type="button"
                onClick={onOpenTemplateGallery}
                title="New scenario from template"
                className="text-[var(--color-muted-foreground)] hover:text-[var(--color-accent)] transition-colors"
              >
                <LayoutTemplate size={11} />
              </button>
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
          {filteredScenarios.length === 0 && !scenarioQuery && !creatingScenario && (
            <div className="flex flex-col items-center gap-3 px-4 py-6 text-center">
              <Play size={20} className="text-[var(--color-muted-foreground)] opacity-30" />
              <p className="text-xs text-[var(--color-muted-foreground)]">No scenarios yet</p>
              <div className="flex flex-col gap-1.5 w-full">
                <button
                  type="button"
                  onClick={() => setCreatingScenario(true)}
                  className="w-full text-xs py-1.5 px-3 rounded border border-[var(--color-border)] text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-muted)] transition-colors"
                >
                  New scenario
                </button>
                <button
                  type="button"
                  onClick={onOpenTemplateGallery}
                  className="w-full text-xs py-1.5 px-3 rounded border border-[var(--color-border)] text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-muted)] transition-colors"
                >
                  Browse templates
                </button>
              </div>
            </div>
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
                    onRenamingChange(null)
                    onRenameScenario?.(s.path, newName)
                  }}
                  onCancel={() => onRenamingChange(null)}
                />
              ) : (
                <button
                  type="button"
                  className="flex-1 min-w-0 flex items-center gap-1.5 text-left py-0.5"
                  onClick={() => onSelectScenario?.(s.path, s.title)}
                  onDoubleClick={e => { e.stopPropagation(); onRenamingChange(s.path) }}
                  onContextMenu={e => { e.preventDefault(); e.stopPropagation(); onScenarioContextMenu(e, s.path, s.title) }}
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
    </>
  )
}
