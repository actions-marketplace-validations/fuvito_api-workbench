import { useState, useEffect, useRef, useMemo } from 'react'
import { Search, FileCode, FlaskConical } from 'lucide-react'
import { cn } from '@/lib/utils'
import { METHOD_COLOR } from '@/lib/httpUtils'
import type { FileNode, ScenarioInfo } from '@/lib/bridge'

function flattenFiles(nodes: FileNode[]): FileNode[] {
  return nodes.flatMap(n => n.isDir ? flattenFiles(n.children ?? []) : [n])
}

function displayName(node: FileNode): string {
  return node.name.replace(/\.(yaml|yml|json)$/i, '')
}

type ResultItem =
  | { kind: 'request'; node: FileNode }
  | { kind: 'scenario'; scenario: ScenarioInfo }

type Props = {
  fileTree: FileNode[]
  scenarios?: ScenarioInfo[]
  onSelect: (path: string) => void
  onSelectScenario?: (path: string, name: string) => void
  onClose: () => void
}

export function QuickOpen({ fileTree, scenarios = [], onSelect, onSelectScenario, onClose }: Props) {
  const [query, setQuery] = useState('')
  const [selectedIdx, setSelectedIdx] = useState(0)
  const inputRef = useRef<HTMLInputElement>(null)
  const selectedRef = useRef<HTMLLIElement>(null)

  const allFiles = useMemo(() => flattenFiles(fileTree), [fileTree])
  const allScenarios = useMemo(() => scenarios.filter(s => !s.isFolder), [scenarios])

  const results = useMemo((): ResultItem[] => {
    const q = query.trim().toLowerCase()
    const requests: ResultItem[] = (q
      ? allFiles.filter(f => displayName(f).toLowerCase().includes(q) || f.path.toLowerCase().includes(q))
      : allFiles
    ).slice(0, 15).map(node => ({ kind: 'request', node }))

    const scenResults: ResultItem[] = (q
      ? allScenarios.filter(s => s.title.toLowerCase().includes(q) || s.path.toLowerCase().includes(q))
      : allScenarios
    ).slice(0, 10).map(scenario => ({ kind: 'scenario', scenario }))

    return [...requests, ...scenResults].slice(0, 20)
  }, [allFiles, allScenarios, query])

  useEffect(() => { setTimeout(() => inputRef.current?.focus(), 0) }, [])
  useEffect(() => { setSelectedIdx(0) }, [query])
  useEffect(() => {
    selectedRef.current?.scrollIntoView({ block: 'nearest' })
  }, [selectedIdx])

  function commit(item: ResultItem) {
    if (item.kind === 'request') {
      onSelect(item.node.path)
    } else {
      onSelectScenario?.(item.scenario.path, item.scenario.title)
    }
    onClose()
  }

  function handleKeyDown(e: React.KeyboardEvent) {
    if (e.key === 'ArrowDown') { e.preventDefault(); setSelectedIdx(i => Math.min(i + 1, results.length - 1)) }
    else if (e.key === 'ArrowUp') { e.preventDefault(); setSelectedIdx(i => Math.max(i - 1, 0)) }
    else if (e.key === 'Enter' && results[selectedIdx]) commit(results[selectedIdx])
    else if (e.key === 'Escape') onClose()
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-start justify-center pt-[15vh] bg-black/50"
      onClick={onClose}
    >
      <div
        className="w-full max-w-md bg-[var(--color-card)] border border-[var(--color-border)] rounded-lg shadow-2xl overflow-hidden"
        onClick={e => e.stopPropagation()}
      >
        <div className="flex items-center gap-2 px-3 py-2.5 border-b border-[var(--color-border)]">
          <Search size={14} className="text-[var(--color-muted-foreground)] shrink-0" />
          <input
            ref={inputRef}
            value={query}
            onChange={e => setQuery(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Open request or scenario…"
            className="flex-1 bg-transparent outline-none text-sm text-[var(--color-foreground)] placeholder:text-[var(--color-muted-foreground)]"
          />
          <kbd className="text-[10px] text-[var(--color-muted-foreground)] border border-[var(--color-border)] rounded px-1 py-0.5 font-mono">Esc</kbd>
        </div>
        {results.length === 0 ? (
          <p className="px-4 py-6 text-xs text-center text-[var(--color-muted-foreground)]">No results found.</p>
        ) : (
          <ul className="max-h-80 overflow-auto py-1">
            {results.map((item, i) => {
              const isSelected = i === selectedIdx
              if (item.kind === 'request') {
                const file = item.node
                const folder = file.path.replace(/\\/g, '/').split('/').slice(-2, -1)[0] ?? ''
                return (
                  <li key={file.path} ref={isSelected ? selectedRef : undefined}>
                    <button
                      type="button"
                      onClick={() => commit(item)}
                      className={cn(
                        'w-full flex items-center gap-2.5 px-3 py-2 cursor-pointer transition-colors text-left',
                        isSelected ? 'bg-[var(--color-accent)]/15' : 'hover:bg-[var(--color-muted)]'
                      )}
                    >
                      {file.method ? (
                        <span className={cn('text-[10px] font-mono font-bold w-7 shrink-0', METHOD_COLOR[file.method as keyof typeof METHOD_COLOR] ?? 'text-[var(--color-muted-foreground)]')}>
                          {file.method.slice(0, 3)}
                        </span>
                      ) : (
                        <FileCode size={12} className="text-[var(--color-muted-foreground)] shrink-0 w-7" />
                      )}
                      <span className="text-xs text-[var(--color-foreground)] truncate flex-1">{displayName(file)}</span>
                      {folder && (
                        <span className="text-[10px] text-[var(--color-muted-foreground)] shrink-0">{folder}</span>
                      )}
                    </button>
                  </li>
                )
              } else {
                const { scenario } = item
                return (
                  <li key={scenario.path} ref={isSelected ? selectedRef : undefined}>
                    <button
                      type="button"
                      onClick={() => commit(item)}
                      className={cn(
                        'w-full flex items-center gap-2.5 px-3 py-2 cursor-pointer transition-colors text-left',
                        isSelected ? 'bg-[var(--color-accent)]/15' : 'hover:bg-[var(--color-muted)]'
                      )}
                    >
                      <FlaskConical size={12} className="text-purple-400 shrink-0 w-7" />
                      <span className="text-xs text-[var(--color-foreground)] truncate flex-1">{scenario.title}</span>
                      {scenario.folder && (
                        <span className="text-[10px] text-[var(--color-muted-foreground)] shrink-0">{scenario.folder}</span>
                      )}
                    </button>
                  </li>
                )
              }
            })}
          </ul>
        )}
      </div>
    </div>
  )
}
