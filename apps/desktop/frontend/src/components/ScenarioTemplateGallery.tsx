import { useState, useEffect, useRef } from 'react'
import { Search, X, BookOpen, Play, Loader2, Globe, CheckCircle } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { TemplateInfo } from '@/lib/bridge'

type Props = {
  templates: TemplateInfo[]
  loading: boolean
  onUse: (templateID: string, name: string) => void
  onClose: () => void
}

const CATEGORY_COLORS: Record<string, string> = {
  'Conformance': 'text-violet-400 bg-violet-400/10 border-violet-400/20',
  'Public APIs': 'text-emerald-400 bg-emerald-400/10 border-emerald-400/20',
  'Other':       'text-[var(--color-muted-foreground)] bg-[var(--color-muted)] border-[var(--color-border)]',
}

function TemplateCard({ t, onUse }: { t: TemplateInfo; onUse: (id: string, name: string) => void }) {
  const [naming, setNaming] = useState(false)
  const [name, setName] = useState(t.title)
  const inputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    if (naming) inputRef.current?.select()
  }, [naming])

  function confirm() {
    const n = name.trim() || t.title
    setNaming(false)
    onUse(t.id, n)
  }

  return (
    <div className={cn(
      'flex flex-col rounded border border-[var(--color-border)] bg-[var(--color-card)] p-3 gap-2',
      'hover:border-[var(--color-accent)]/40 transition-colors',
    )}>
      <div className="flex items-start justify-between gap-2">
        <span className="text-xs font-semibold text-[var(--color-foreground)] leading-snug flex-1">
          {t.title}
        </span>
        <span className={cn(
          'shrink-0 text-[9px] font-semibold uppercase tracking-wide px-1.5 py-0.5 rounded border',
          CATEGORY_COLORS[t.category] ?? CATEGORY_COLORS['Other'],
        )}>
          {t.category}
        </span>
      </div>

      <p className="text-[11px] text-[var(--color-muted-foreground)] leading-relaxed flex-1">
        {t.description}
      </p>

      <div className="flex items-center gap-1.5 flex-wrap">
        {t.tags?.map(tag => (
          <span key={tag} className="text-[9px] px-1.5 py-0.5 rounded bg-[var(--color-muted)] text-[var(--color-muted-foreground)]">
            {tag}
          </span>
        ))}
        <span className="ml-auto text-[10px] text-[var(--color-muted-foreground)]">
          {t.stepCount} {t.stepCount === 1 ? 'step' : 'steps'}
        </span>
      </div>

      {t.note && (
        <p className="text-[10px] text-amber-400/80 bg-amber-400/5 border border-amber-400/20 rounded px-2 py-1 leading-relaxed">
          {t.note}
        </p>
      )}

      {naming ? (
        <div className="flex items-center gap-1.5 mt-1">
          <input
            ref={inputRef}
            value={name}
            onChange={e => setName(e.target.value)}
            onKeyDown={e => {
              e.stopPropagation()
              if (e.key === 'Enter') confirm()
              if (e.key === 'Escape') { setNaming(false); setName(t.title) }
            }}
            placeholder="Scenario name…"
            className={cn(
              'flex-1 min-w-0 text-xs rounded px-2 py-1 outline-none',
              'bg-[var(--color-input)] border border-[var(--color-accent)]',
              'text-[var(--color-foreground)] placeholder:text-[var(--color-muted-foreground)]',
            )}
          />
          <button
            type="button"
            onClick={confirm}
            className="shrink-0 px-2 py-1 rounded text-xs bg-[var(--color-accent)] text-white hover:opacity-90 transition-opacity"
          >
            Create
          </button>
          <button
            type="button"
            onClick={() => { setNaming(false); setName(t.title) }}
            className="shrink-0 p-1 rounded text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)]"
          >
            <X size={12} />
          </button>
        </div>
      ) : (
        <button
          type="button"
          onClick={() => setNaming(true)}
          className={cn(
            'mt-1 flex items-center justify-center gap-1.5 text-xs rounded px-3 py-1.5 font-semibold',
            'bg-[var(--color-muted)] border border-[var(--color-border)]',
            'text-[var(--color-foreground)] hover:bg-[var(--color-border)] transition-colors',
          )}
        >
          <Play size={10} /> Use this template
        </button>
      )}
    </div>
  )
}

export function ScenarioTemplateGallery({ templates, loading, onUse, onClose }: Props) {
  const [query, setQuery] = useState('')
  const [category, setCategory] = useState<string>('All')

  const categories = ['All', ...Array.from(new Set(templates.map(t => t.category)))]

  const filtered = templates.filter(t => {
    const matchCat = category === 'All' || t.category === category
    const q = query.toLowerCase()
    const matchQ = !q || t.title.toLowerCase().includes(q) || t.description.toLowerCase().includes(q) || t.tags?.some(tag => tag.toLowerCase().includes(q))
    return matchCat && matchQ
  })

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
      onMouseDown={e => { if (e.target === e.currentTarget) onClose() }}
    >
      <div className="relative flex flex-col w-[720px] max-w-[95vw] max-h-[82vh] rounded-lg border border-[var(--color-border)] bg-[var(--color-background)] shadow-2xl overflow-hidden">
        {/* Header */}
        <div className="flex items-center gap-3 px-4 py-3 border-b border-[var(--color-border)] shrink-0">
          <BookOpen size={15} className="text-[var(--color-accent)] shrink-0" />
          <span className="font-semibold text-sm text-[var(--color-foreground)]">Scenario templates</span>
          <div className="flex-1" />
          <button
            type="button"
            onClick={onClose}
            className="text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors"
          >
            <X size={14} />
          </button>
        </div>

        {/* Search + category filter */}
        <div className="flex items-center gap-2 px-4 py-2.5 border-b border-[var(--color-border)] shrink-0">
          <div className="flex items-center gap-1.5 flex-1 bg-[var(--color-muted)] rounded px-2 py-1.5">
            <Search size={11} className="shrink-0 text-[var(--color-muted-foreground)]" />
            <input
              autoFocus
              value={query}
              onChange={e => setQuery(e.target.value)}
              placeholder="Search templates…"
              className="flex-1 min-w-0 bg-transparent text-xs outline-none text-[var(--color-foreground)] placeholder:text-[var(--color-muted-foreground)]"
            />
            {query && (
              <button type="button" onClick={() => setQuery('')} className="shrink-0 text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)]">
                <X size={10} />
              </button>
            )}
          </div>
          <div className="flex items-center gap-1 shrink-0">
            {categories.map(cat => (
              <button
                key={cat}
                type="button"
                onClick={() => setCategory(cat)}
                className={cn(
                  'px-2 py-1 rounded text-[10px] font-semibold transition-colors',
                  category === cat
                    ? 'bg-[var(--color-accent)] text-white'
                    : 'bg-[var(--color-muted)] text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)]',
                )}
              >
                {cat === 'Public APIs' ? <Globe size={9} className="inline mr-0.5" /> : null}
                {cat === 'Conformance' ? <CheckCircle size={9} className="inline mr-0.5" /> : null}
                {cat}
              </button>
            ))}
          </div>
        </div>

        {/* Grid */}
        <div className="flex-1 overflow-auto p-4">
          {loading ? (
            <div className="flex items-center justify-center py-16 text-[var(--color-muted-foreground)]">
              <Loader2 size={18} className="animate-spin" />
            </div>
          ) : filtered.length === 0 ? (
            <p className="text-center text-xs text-[var(--color-muted-foreground)] py-16">No templates match your search.</p>
          ) : (
            <div className="grid grid-cols-2 gap-3">
              {filtered.map(t => (
                <TemplateCard key={t.id} t={t} onUse={onUse} />
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
