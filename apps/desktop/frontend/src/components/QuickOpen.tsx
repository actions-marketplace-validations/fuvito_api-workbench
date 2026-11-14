import { useState, useEffect, useRef, useMemo } from 'react'
import { Search, FileCode } from 'lucide-react'
import { cn } from '@/lib/utils'
import { METHOD_COLOR } from '@/lib/httpUtils'
import type { FileNode } from '@/lib/bridge'

function flattenFiles(nodes: FileNode[]): FileNode[] {
  return nodes.flatMap(n => n.isDir ? flattenFiles(n.children ?? []) : [n])
}

function displayName(node: FileNode): string {
  return node.name.replace(/\.(yaml|yml|json)$/i, '')
}

type Props = {
  fileTree: FileNode[]
  onSelect: (path: string) => void
  onClose: () => void
}

export function QuickOpen({ fileTree, onSelect, onClose }: Props) {
  const [query, setQuery] = useState('')
  const [selectedIdx, setSelectedIdx] = useState(0)
  const inputRef = useRef<HTMLInputElement>(null)
  const selectedRef = useRef<HTMLLIElement>(null)

  const allFiles = useMemo(() => flattenFiles(fileTree), [fileTree])
  const results = useMemo(() => {
    if (!query.trim()) return allFiles.slice(0, 20)
    const q = query.toLowerCase()
    return allFiles.filter(f =>
      displayName(f).toLowerCase().includes(q) || f.path.toLowerCase().includes(q)
    ).slice(0, 20)
  }, [allFiles, query])

  useEffect(() => { setTimeout(() => inputRef.current?.focus(), 0) }, [])
  useEffect(() => { setSelectedIdx(0) }, [query])
  useEffect(() => {
    selectedRef.current?.scrollIntoView({ block: 'nearest' })
  }, [selectedIdx])

  function commit(path: string) { onSelect(path); onClose() }

  function handleKeyDown(e: React.KeyboardEvent) {
    if (e.key === 'ArrowDown') { e.preventDefault(); setSelectedIdx(i => Math.min(i + 1, results.length - 1)) }
    else if (e.key === 'ArrowUp') { e.preventDefault(); setSelectedIdx(i => Math.max(i - 1, 0)) }
    else if (e.key === 'Enter' && results[selectedIdx]) commit(results[selectedIdx].path)
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
            placeholder="Open request…"
            className="flex-1 bg-transparent outline-none text-sm text-[var(--color-foreground)] placeholder:text-[var(--color-muted-foreground)]"
          />
          <kbd className="text-[10px] text-[var(--color-muted-foreground)] border border-[var(--color-border)] rounded px-1 py-0.5 font-mono">Esc</kbd>
        </div>
        {results.length === 0 ? (
          <p className="px-4 py-6 text-xs text-center text-[var(--color-muted-foreground)]">No requests found.</p>
        ) : (
          <ul className="max-h-80 overflow-auto py-1">
            {results.map((file, i) => {
              const folder = file.path.replace(/\\/g, '/').split('/').slice(-2, -1)[0] ?? ''
              return (
                <li key={file.path} ref={i === selectedIdx ? selectedRef : undefined}>
                  <button
                    type="button"
                    onClick={() => commit(file.path)}
                    className={cn(
                      'w-full flex items-center gap-2.5 px-3 py-2 cursor-pointer transition-colors text-left',
                      i === selectedIdx ? 'bg-[var(--color-accent)]/15' : 'hover:bg-[var(--color-muted)]'
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
            })}
          </ul>
        )}
      </div>
    </div>
  )
}
