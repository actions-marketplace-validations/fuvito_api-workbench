import { useState, useEffect, useRef, useMemo } from 'react'
import { Search, FileCode, FlaskConical, Globe, FilePlus, FlaskConical as NewScenarioIcon, Terminal } from 'lucide-react'
import { cn } from '@/lib/utils'
import { METHOD_COLOR } from '@/lib/httpUtils'
import type { FileNode, ScenarioInfo, EnvInfo } from '@/lib/bridge'

function flattenFiles(nodes: FileNode[]): FileNode[] {
  return nodes.flatMap(n => n.isDir ? flattenFiles(n.children ?? []) : [n])
}

function displayName(node: FileNode): string {
  return node.name.replace(/\.(yaml|yml|json)$/i, '')
}

type ResultItem =
  | { kind: 'request'; node: FileNode }
  | { kind: 'scenario'; scenario: ScenarioInfo }
  | { kind: 'environment'; env: EnvInfo }
  | { kind: 'command'; id: string; label: string; hint: string }

type Props = {
  fileTree: FileNode[]
  scenarios?: ScenarioInfo[]
  environments?: EnvInfo[]
  onSelect: (path: string) => void
  onSelectScenario?: (path: string, name: string) => void
  onSelectEnvironment?: (path: string) => void
  onNewRequest?: () => void
  onNewScenario?: () => void
  onClose: () => void
}

type CommandItem = { kind: 'command'; id: string; label: string; hint: string }

const COMMANDS: CommandItem[] = [
  { kind: 'command', id: 'new-request',  label: 'New Request',  hint: 'Open a blank request tab' },
  { kind: 'command', id: 'new-scenario', label: 'New Scenario', hint: 'Create a new scenario file' },
]

export function QuickOpen({
  fileTree, scenarios = [], environments = [],
  onSelect, onSelectScenario, onSelectEnvironment,
  onNewRequest, onNewScenario, onClose,
}: Props) {
  const [query, setQuery] = useState('')
  const [selectedIdx, setSelectedIdx] = useState(0)
  const inputRef = useRef<HTMLInputElement>(null)
  const selectedRef = useRef<HTMLLIElement>(null)

  const allFiles = useMemo(() => flattenFiles(fileTree), [fileTree])
  const allScenarios = useMemo(() => scenarios.filter(s => !s.isFolder), [scenarios])

  // If query starts with '>' show only commands; otherwise show files + scenarios + envs + commands
  const isCommandMode = query.startsWith('>')
  const q = isCommandMode ? query.slice(1).trim().toLowerCase() : query.trim().toLowerCase()

  const results = useMemo((): ResultItem[] => {
    if (isCommandMode) {
      const cmds: ResultItem[] = (q ? COMMANDS.filter(c => c.label.toLowerCase().includes(q)) : COMMANDS)
      const envCmds: ResultItem[] = (q
        ? environments.filter(e => e.name.toLowerCase().includes(q))
        : environments
      ).map(env => ({ kind: 'environment' as const, env }))
      return [...cmds, ...envCmds].slice(0, 20)
    }

    const requests: ResultItem[] = (q
      ? allFiles.filter(f => displayName(f).toLowerCase().includes(q) || f.path.toLowerCase().includes(q))
      : allFiles
    ).slice(0, 10).map(node => ({ kind: 'request' as const, node }))

    const scenResults: ResultItem[] = (q
      ? allScenarios.filter(s => s.title.toLowerCase().includes(q) || s.path.toLowerCase().includes(q))
      : allScenarios
    ).slice(0, 7).map(scenario => ({ kind: 'scenario' as const, scenario }))

    const envResults: ResultItem[] = (q
      ? environments.filter(e => e.name.toLowerCase().includes(q))
      : environments
    ).slice(0, 4).map(env => ({ kind: 'environment' as const, env }))

    // Commands appear when query is empty or matches command label
    const cmdResults: ResultItem[] = q
      ? COMMANDS.filter(c => c.label.toLowerCase().includes(q))
      : COMMANDS

    return [...requests, ...scenResults, ...envResults, ...cmdResults].slice(0, 25)
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [allFiles, allScenarios, environments, query, isCommandMode, q])

  useEffect(() => { setTimeout(() => inputRef.current?.focus(), 0) }, [])
  useEffect(() => { setSelectedIdx(0) }, [query])
  useEffect(() => {
    selectedRef.current?.scrollIntoView({ block: 'nearest' })
  }, [selectedIdx])

  function commit(item: ResultItem) {
    if (item.kind === 'request') {
      onSelect(item.node.path)
    } else if (item.kind === 'scenario') {
      onSelectScenario?.(item.scenario.path, item.scenario.title)
    } else if (item.kind === 'environment') {
      onSelectEnvironment?.(item.env.path)
    } else if (item.kind === 'command') {
      if (item.id === 'new-request') onNewRequest?.()
      else if (item.id === 'new-scenario') onNewScenario?.()
    }
    onClose()
  }

  function handleKeyDown(e: React.KeyboardEvent) {
    if (e.key === 'ArrowDown') { e.preventDefault(); setSelectedIdx(i => Math.min(i + 1, results.length - 1)) }
    else if (e.key === 'ArrowUp') { e.preventDefault(); setSelectedIdx(i => Math.max(i - 1, 0)) }
    else if (e.key === 'Enter' && results[selectedIdx]) commit(results[selectedIdx])
    else if (e.key === 'Escape') onClose()
  }

  const placeholder = isCommandMode ? 'Type a command or environment name…' : 'Open request, scenario, env  ·  > for commands'

  return (
    <div
      className="fixed inset-0 z-50 flex items-start justify-center pt-[15vh] bg-black/50"
      onClick={onClose}
    >
      <div
        className="w-full max-w-lg bg-[var(--color-card)] border border-[var(--color-border)] rounded-lg shadow-2xl overflow-hidden"
        onClick={e => e.stopPropagation()}
      >
        <div className="flex items-center gap-2 px-3 py-2.5 border-b border-[var(--color-border)]">
          {isCommandMode
            ? <Terminal size={14} className="text-[var(--color-accent)] shrink-0" />
            : <Search size={14} className="text-[var(--color-muted-foreground)] shrink-0" />}
          <input
            ref={inputRef}
            value={query}
            onChange={e => setQuery(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder={placeholder}
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
              const baseClass = cn(
                'w-full flex items-center gap-2.5 px-3 py-2 cursor-pointer transition-colors text-left',
                isSelected ? 'bg-[var(--color-accent)]/15' : 'hover:bg-[var(--color-muted)]'
              )
              if (item.kind === 'request') {
                const file = item.node
                const folder = file.path.replace(/\\/g, '/').split('/').slice(-2, -1)[0] ?? ''
                return (
                  <li key={file.path} ref={isSelected ? selectedRef : undefined}>
                    <button type="button" onClick={() => commit(item)} className={baseClass}>
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
              }
              if (item.kind === 'scenario') {
                const { scenario } = item
                return (
                  <li key={scenario.path} ref={isSelected ? selectedRef : undefined}>
                    <button type="button" onClick={() => commit(item)} className={baseClass}>
                      <FlaskConical size={12} className="text-purple-400 shrink-0 w-7" />
                      <span className="text-xs text-[var(--color-foreground)] truncate flex-1">{scenario.title}</span>
                      {scenario.folder && (
                        <span className="text-[10px] text-[var(--color-muted-foreground)] shrink-0">{scenario.folder}</span>
                      )}
                    </button>
                  </li>
                )
              }
              if (item.kind === 'environment') {
                const { env } = item
                return (
                  <li key={env.path} ref={isSelected ? selectedRef : undefined}>
                    <button type="button" onClick={() => commit(item)} className={baseClass}>
                      <Globe size={12} className="text-cyan-400 shrink-0 w-7" />
                      <span className="text-xs text-[var(--color-foreground)] truncate flex-1">{env.name}</span>
                      <span className="text-[10px] text-[var(--color-muted-foreground)] shrink-0">Switch env</span>
                    </button>
                  </li>
                )
              }
              // command
              return (
                <li key={item.id} ref={isSelected ? selectedRef : undefined}>
                  <button type="button" onClick={() => commit(item)} className={baseClass}>
                    {item.id === 'new-request'
                      ? <FilePlus size={12} className="text-[var(--color-accent)] shrink-0 w-7" />
                      : <NewScenarioIcon size={12} className="text-purple-400 shrink-0 w-7" />}
                    <span className="text-xs text-[var(--color-foreground)] truncate flex-1">{item.label}</span>
                    <span className="text-[10px] text-[var(--color-muted-foreground)] shrink-0">{item.hint}</span>
                  </button>
                </li>
              )
            })}
          </ul>
        )}
        {!isCommandMode && (
          <div className="px-3 py-1.5 border-t border-[var(--color-border)] flex items-center justify-end">
            <span className="text-[10px] text-[var(--color-muted-foreground)]">Type <kbd className="font-mono border border-[var(--color-border)] rounded px-0.5">&gt;</kbd> for commands</span>
          </div>
        )}
      </div>
    </div>
  )
}
