import { useState } from 'react'
import { ChevronDown, ChevronRight, Plus, X, Lock } from 'lucide-react'
import { cn } from '@/lib/utils'
import { inputCls } from '@/lib/styles'
import { VarValueInput } from '@/components/KeyValueEditor'
import type { ScenarioVariable } from '@/lib/bridge'

type Props = {
  variables: ScenarioVariable[]
  onChange: (variables: ScenarioVariable[]) => void
}

export function ScenarioVariablesSection({ variables, onChange }: Props) {
  const [open, setOpen] = useState(variables.length > 0)

  function update(i: number, patch: Partial<ScenarioVariable>) {
    onChange(variables.map((v, idx) => (idx === i ? { ...v, ...patch } : v)))
  }

  return (
    <div className="shrink-0 px-4 py-2 border-b border-[var(--color-border)] bg-[var(--color-card)]">
      <button
        type="button"
        onClick={() => setOpen(o => !o)}
        aria-expanded={open}
        className="flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-widest text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)]"
      >
        {open ? <ChevronDown size={11} /> : <ChevronRight size={11} />}
        Variables
        {variables.length > 0 && (
          <span className="px-1.5 rounded-full bg-[var(--color-muted)] text-[9px] normal-case tracking-normal">{variables.length}</span>
        )}
      </button>

      {open && (
        <div className="mt-2 flex flex-col gap-1.5">
          <p className="text-[11px] text-[var(--color-muted-foreground)]">
            Generated once per run and the same in every step. Values can use dynamic functions such as{' '}
            <code className="font-mono">{'{{$randomString 8}}'}</code> or <code className="font-mono">{'{{$date +7d}}'}</code> (type <code className="font-mono">{'{{'}</code> to browse them).
          </p>
          {variables.map((v, i) => (
            <div key={i} className="flex items-center gap-1.5">
              <input
                value={v.name}
                onChange={e => update(i, { name: e.target.value })}
                placeholder="name"
                aria-label={`Variable ${i + 1} name`}
                className={cn(inputCls, 'w-40 shrink-0 font-mono px-2 py-1')}
              />
              <VarValueInput
                value={v.value}
                onChange={val => update(i, { value: val })}
                placeholder="value, e.g. user-{{$randomString 6}}@example.com"
                envVarNames={variables.slice(0, i).map(x => x.name).filter(Boolean)}
              />
              <button
                type="button"
                onClick={() => update(i, { secret: !v.secret })}
                aria-pressed={!!v.secret}
                title={v.secret ? 'Secret: masked in reports' : 'Mark as secret (masked in reports)'}
                className={cn(
                  'shrink-0 p-1 rounded transition-colors',
                  v.secret ? 'text-amber-400 bg-amber-500/10' : 'text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-muted)]',
                )}
              >
                <Lock size={12} />
              </button>
              <button
                type="button"
                onClick={() => onChange(variables.filter((_, idx) => idx !== i))}
                aria-label={`Remove variable ${v.name || i + 1}`}
                className="shrink-0 p-1 rounded text-[var(--color-muted-foreground)] hover:text-red-400 hover:bg-red-500/10 transition-colors"
              >
                <X size={12} />
              </button>
            </div>
          ))}
          <button
            type="button"
            onClick={() => onChange([...variables, { name: '', value: '' }])}
            className="self-start flex items-center gap-1 text-[11px] text-[var(--color-accent)] hover:underline"
          >
            <Plus size={11} /> Add variable
          </button>
        </div>
      )}
    </div>
  )
}
