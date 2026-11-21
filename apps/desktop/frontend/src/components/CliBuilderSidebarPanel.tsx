import { useEffect, useState } from 'react'
import { cn } from '@/lib/utils'
import { getCliCatalog } from '@/lib/bridge'
import type { CliCommand } from '@/lib/bridge'

type Props = {
  // The command selected on the CLI Builder page ('' until it has loaded).
  selected: string
  // Selects the command on the page and shows the page.
  onSelect: (name: string) => void
}

// The commands the builder offers, each with its one-line goal.
export function CliBuilderSidebarPanel({ selected, onSelect }: Props) {
  const [catalog, setCatalog] = useState<CliCommand[] | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    let current = true
    getCliCatalog()
      .then(c => { if (current) setCatalog(c) })
      .catch(e => { if (current) setError(String(e)) })
    return () => { current = false }
  }, [])

  return (
    <div data-testid="sidebar-cli" className="flex flex-col flex-1 overflow-auto text-xs">
      <h3 className="px-3 pt-3 pb-1 text-[10px] font-semibold uppercase tracking-widest text-[var(--color-muted-foreground)]">Commands</h3>
      {error && <p role="alert" className="px-3 py-2 text-red-400">{error}</p>}
      {!catalog && !error && <p className="px-3 py-3 text-[var(--color-muted-foreground)]">Loading…</p>}
      {catalog && (
        <ul aria-label="Commands">
          {catalog.map(c => (
            <li key={c.name}>
              <button
                type="button"
                aria-pressed={c.name === selected}
                onClick={() => onSelect(c.name)}
                className={cn(
                  'w-full text-left px-3 py-1.5 hover:bg-[var(--color-muted)]',
                  c.name === selected && 'bg-[var(--color-muted)] border-l-2 border-[var(--color-accent)] pl-[10px]',
                )}
              >
                <span className="block font-mono font-medium">clrsig {c.name}</span>
                <span className="block text-[var(--color-muted-foreground)]">{c.goal}</span>
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
