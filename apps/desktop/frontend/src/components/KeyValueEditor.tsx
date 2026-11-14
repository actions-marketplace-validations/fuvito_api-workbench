import { useId, useState, useMemo, useRef } from 'react'
import { X } from 'lucide-react'
import { cn } from '@/lib/utils'
import { inputCls as baseInputCls } from '@/lib/styles'
import type { KVPair } from '@/types'

type Props = {
  pairs: KVPair[]
  onChange: (pairs: KVPair[]) => void
  keyPlaceholder?: string
  valuePlaceholder?: string
  keySuggestions?: string[]
  envVarNames?: string[]
}

const inputCls = cn(baseInputCls, 'flex-1 min-w-0 font-mono px-2 py-1')

type VarValueInputProps = {
  value: string
  onChange: (v: string) => void
  placeholder?: string
  envVarNames?: string[]
}

function VarValueInput({ value, onChange, placeholder, envVarNames }: VarValueInputProps) {
  const inputRef = useRef<HTMLInputElement>(null)
  const [acPartial, setAcPartial] = useState<string | null>(null)
  const [acIndex, setAcIndex] = useState(0)

  const acMatches = useMemo(() => {
    if (acPartial === null || !envVarNames?.length) return []
    const q = acPartial.toLowerCase()
    return envVarNames.filter(n => n.toLowerCase().includes(q))
  }, [acPartial, envVarNames])

  function checkAc(el: HTMLInputElement, v: string) {
    const pos = el.selectionStart ?? v.length
    const before = v.slice(0, pos)
    const oi = before.lastIndexOf('{{')
    if (oi !== -1 && !before.slice(oi + 2).includes('}}')) {
      setAcPartial(before.slice(oi + 2))
      setAcIndex(0)
    } else {
      setAcPartial(null)
    }
  }

  function insertVar(name: string) {
    const el = inputRef.current; if (!el) return
    const pos = el.selectionStart ?? value.length
    const before = value.slice(0, pos)
    const oi = before.lastIndexOf('{{')
    const newVal = value.slice(0, oi) + '{{' + name + '}}' + value.slice(pos)
    onChange(newVal); setAcPartial(null)
    requestAnimationFrame(() => {
      el.setSelectionRange(oi + 4 + name.length, oi + 4 + name.length)
      el.focus()
    })
  }

  function handleKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (acMatches.length > 0 && acPartial !== null) {
      if (e.key === 'ArrowDown') { e.preventDefault(); setAcIndex(i => Math.min(i + 1, acMatches.length - 1)); return }
      if (e.key === 'ArrowUp') { e.preventDefault(); setAcIndex(i => Math.max(i - 1, 0)); return }
      if (e.key === 'Tab' || e.key === 'Enter') {
        if (acMatches[acIndex]) { e.preventDefault(); insertVar(acMatches[acIndex]); return }
      }
      if (e.key === 'Escape') { e.preventDefault(); setAcPartial(null); return }
    }
  }

  return (
    <div className="relative flex-1 min-w-0">
      <input
        ref={inputRef}
        type="text"
        value={value}
        onChange={e => { onChange(e.target.value); checkAc(e.target, e.target.value) }}
        onKeyDown={handleKeyDown}
        onClick={e => checkAc(e.currentTarget, value)}
        onBlur={() => setTimeout(() => setAcPartial(null), 150)}
        placeholder={placeholder}
        className={inputCls}
      />
      {acPartial !== null && acMatches.length > 0 && (
        <ul className="absolute top-full left-0 right-0 z-50 bg-[var(--color-card)] border border-[var(--color-border)] rounded shadow-lg mt-0.5 max-h-40 overflow-auto py-0.5">
          {acMatches.map((name, i) => (
            <li key={name}>
              <button
                type="button"
                onMouseDown={e => { e.preventDefault(); insertVar(name) }}
                className={cn(
                  'w-full text-left px-3 py-1 text-xs font-mono transition-colors',
                  i === acIndex ? 'bg-[var(--color-accent)]/15 text-[var(--color-foreground)]' : 'text-[var(--color-muted-foreground)] hover:bg-[var(--color-muted)]'
                )}
              >
                {name}
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}

export function KeyValueEditor({
  pairs,
  onChange,
  keyPlaceholder = 'Key',
  valuePlaceholder = 'Value',
  keySuggestions,
  envVarNames,
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
          <input
            type="checkbox"
            title="Toggle all"
            checked={pairs.every(p => p.enabled)}
            ref={el => {
              if (el) el.indeterminate = pairs.some(p => p.enabled) && !pairs.every(p => p.enabled)
            }}
            onChange={e => onChange(pairs.map(p => ({ ...p, enabled: e.target.checked })))}
            className="shrink-0 accent-[var(--color-accent)] w-3.5 h-3.5 focus-visible:ring-2 focus-visible:ring-[var(--color-accent)] focus-visible:ring-offset-1 focus-visible:ring-offset-[var(--color-background)]"
          />
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
        <div key={pair.id} className="flex items-center gap-1.5 rounded px-1 -mx-1 hover:bg-[var(--color-muted)] transition-colors group">
          <input
            type="checkbox"
            checked={pair.enabled}
            onChange={e => update(pair.id, 'enabled', e.target.checked)}
            className="shrink-0 accent-[var(--color-accent)] w-3.5 h-3.5 focus-visible:ring-2 focus-visible:ring-[var(--color-accent)] focus-visible:ring-offset-1 focus-visible:ring-offset-[var(--color-background)]"
          />
          <input
            type="text"
            value={pair.name}
            onChange={e => update(pair.id, 'name', e.target.value)}
            placeholder={keyPlaceholder}
            list={keySuggestions ? listId : undefined}
            className={inputCls}
          />
          <VarValueInput
            value={pair.value}
            onChange={v => update(pair.id, 'value', v)}
            placeholder={valuePlaceholder}
            envVarNames={envVarNames}
          />
          <button
            type="button"
            onClick={() => remove(pair.id)}
            aria-label="Remove row"
            className="shrink-0 w-4 flex items-center justify-center text-[var(--color-muted-foreground)] hover:text-[var(--color-destructive)] transition-colors opacity-0 group-hover:opacity-100"
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
