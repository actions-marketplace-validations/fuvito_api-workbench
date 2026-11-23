import { useState, useRef, useMemo } from 'react'
import { cn } from '@/lib/utils'
import { suggest } from '@/lib/dynamicFunctions'

function renderVarTokens(text: string): React.ReactNode[] {
  const nodes: React.ReactNode[] = []
  const re = /(\{\{[^}]*\}\})/g
  let last = 0, i = 0
  for (const m of text.matchAll(re)) {
    if (m.index! > last) nodes.push(<span key={`t${i++}`}>{text.slice(last, m.index)}</span>)
    nodes.push(<span key={`v${i++}`} style={{ color: 'var(--color-accent)' }}>{m[0]}</span>)
    last = m.index! + m[0].length
  }
  if (last < text.length) nodes.push(<span key={`t${i++}`}>{text.slice(last)}</span>)
  return nodes
}

export function HighlightedInput({
  value, onChange, onKeyDown, onPaste, placeholder, className, wrapperClassName = 'min-w-0', envVarNames, padding = 'px-3',
}: {
  value: string
  onChange: (v: string) => void
  onKeyDown?: (e: React.KeyboardEvent<HTMLInputElement>) => void
  onPaste?: (e: React.ClipboardEvent<HTMLInputElement>) => void
  placeholder?: string
  className?: string
  // Classes for the element around the input (it is a flex child that takes the free width).
  wrapperClassName?: string
  envVarNames?: string[]
  // horizontal padding of the field, so the highlight layer lines up with the typed text
  padding?: string
}) {
  const inputRef = useRef<HTMLInputElement>(null)
  const mirrorRef = useRef<HTMLDivElement>(null)
  const hasVars = /\{\{/.test(value)
  const [acPartial, setAcPartial] = useState<string | null>(null)
  const [acIndex, setAcIndex] = useState(0)

  const acMatches = useMemo(
    () => (acPartial === null ? [] : suggest(acPartial, envVarNames)),
    [acPartial, envVarNames],
  )

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

  function syncScroll() {
    if (mirrorRef.current && inputRef.current) {
      mirrorRef.current.scrollLeft = inputRef.current.scrollLeft
    }
  }

  function handleKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (acMatches.length > 0 && acPartial !== null) {
      if (e.key === 'ArrowDown') { e.preventDefault(); setAcIndex(i => Math.min(i + 1, acMatches.length - 1)); return }
      if (e.key === 'ArrowUp') { e.preventDefault(); setAcIndex(i => Math.max(i - 1, 0)); return }
      if (e.key === 'Tab' || e.key === 'Enter') {
        if (acMatches[acIndex]) { e.preventDefault(); insertVar(acMatches[acIndex].insert); return }
      }
      if (e.key === 'Escape') { e.preventDefault(); setAcPartial(null); return }
    }
    onKeyDown?.(e)
  }

  return (
    <div className={cn('relative flex-1', wrapperClassName)}>
      <input
        ref={inputRef}
        type="text"
        value={value}
        onChange={e => { onChange(e.target.value); syncScroll(); checkAc(e.target, e.target.value) }}
        onKeyDown={handleKeyDown}
        onPaste={onPaste}
        onScroll={syncScroll}
        onClick={e => checkAc(e.currentTarget, value)}
        onBlur={() => setTimeout(() => setAcPartial(null), 150)}
        placeholder={placeholder}
        className={cn(className, 'w-full', hasVars && 'text-transparent placeholder:text-transparent')}
        style={hasVars ? { caretColor: 'var(--color-foreground)' } : undefined}
      />
      {hasVars && (
        <div
          ref={mirrorRef}
          aria-hidden
          className={cn('absolute inset-0 flex items-center pointer-events-none overflow-hidden text-xs font-mono', padding)}
        >
          {renderVarTokens(value)}
        </div>
      )}
      {acPartial !== null && acMatches.length > 0 && (
        <ul className="absolute top-full left-0 right-0 z-50 bg-[var(--color-card)] border border-[var(--color-border)] rounded shadow-lg mt-0.5 max-h-40 overflow-auto py-0.5">
          {acMatches.map((item, i) => (
            <li key={item.label}>
              <button
                type="button"
                onMouseDown={e => { e.preventDefault(); insertVar(item.insert) }}
                className={cn(
                  'w-full text-left px-3 py-1 text-xs font-mono transition-colors',
                  i === acIndex ? 'bg-[var(--color-accent)]/15 text-[var(--color-foreground)]' : 'text-[var(--color-muted-foreground)] hover:bg-[var(--color-muted)]'
                )}
              >
                {item.label}
                {item.hint && <span className="ml-2 font-sans text-[10px] opacity-70">{item.hint}</span>}
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
