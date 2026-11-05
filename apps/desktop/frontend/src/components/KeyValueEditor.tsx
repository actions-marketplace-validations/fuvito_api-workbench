import { useId } from 'react'
import { X } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { KVPair } from '@/types'

type Props = {
  pairs: KVPair[]
  onChange: (pairs: KVPair[]) => void
  keyPlaceholder?: string
  valuePlaceholder?: string
  keySuggestions?: string[]
}

const inputCls = cn(
  'flex-1 min-w-0 bg-[var(--color-muted)] text-xs font-mono rounded px-2 py-1',
  'border border-[var(--color-border)] outline-none',
  'focus:border-[var(--color-accent)]',
  'placeholder:text-[var(--color-muted-foreground)]',
  'text-[var(--color-foreground)]',
)

export function KeyValueEditor({
  pairs,
  onChange,
  keyPlaceholder = 'Key',
  valuePlaceholder = 'Value',
  keySuggestions,
}: Props) {
  const listId = useId()

  function add() {
    onChange([...pairs, { id: crypto.randomUUID(), name: '', value: '', enabled: true }])
  }

  function update(id: string, field: 'name' | 'value' | 'enabled', value: string | boolean) {
    onChange(pairs.map(p => (p.id === id ? { ...p, [field]: value } : p)))
  }

  function remove(id: string) {
    onChange(pairs.filter(p => p.id !== id))
  }

  return (
    <div className="flex flex-col gap-1.5">
      {keySuggestions && (
        <datalist id={listId}>
          {keySuggestions.map(s => <option key={s} value={s} />)}
        </datalist>
      )}
      {pairs.length > 0 && (
        <div className="flex items-center gap-1.5 mb-0.5">
          <span className="w-4 shrink-0" />
          <span className="flex-1 text-[10px] text-[var(--color-muted-foreground)] uppercase tracking-widest font-semibold">
            {keyPlaceholder}
          </span>
          <span className="flex-1 text-[10px] text-[var(--color-muted-foreground)] uppercase tracking-widest font-semibold">
            {valuePlaceholder}
          </span>
          <span className="w-4 shrink-0" />
        </div>
      )}
      {pairs.map(pair => (
        <div key={pair.id} className="flex items-center gap-1.5">
          <input
            type="checkbox"
            checked={pair.enabled}
            onChange={e => update(pair.id, 'enabled', e.target.checked)}
            className="shrink-0 accent-[var(--color-accent)] w-3.5 h-3.5"
          />
          <input
            type="text"
            value={pair.name}
            onChange={e => update(pair.id, 'name', e.target.value)}
            placeholder={keyPlaceholder}
            list={keySuggestions ? listId : undefined}
            className={inputCls}
          />
          <input
            type="text"
            value={pair.value}
            onChange={e => update(pair.id, 'value', e.target.value)}
            placeholder={valuePlaceholder}
            className={inputCls}
          />
          <button
            type="button"
            onClick={() => remove(pair.id)}
            className="shrink-0 w-4 flex items-center justify-center text-[var(--color-muted-foreground)] hover:text-[var(--color-destructive)] transition-colors"
          >
            <X size={11} />
          </button>
        </div>
      ))}
      <button
        type="button"
        onClick={add}
        className="self-start text-xs text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors mt-0.5"
      >
        + Add row
      </button>
    </div>
  )
}
