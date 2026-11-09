import { Play, Loader2, CheckCircle2, Save, X, Clock } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { ScenarioResult, EnvInfo } from '@/lib/bridge'

export type ScenarioMode = 'view' | 'visual' | 'yaml'

type Props = {
  title: string
  description?: string
  mode: ScenarioMode
  isEditing: boolean
  saving: boolean
  loadingRaw: boolean
  detailLoaded: boolean
  environments: EnvInfo[]
  selectedEnvPath: string
  onEnvPathChange: (path: string) => void
  result: ScenarioResult | null
  running: boolean
  onShowStats: () => void
  onRun: () => void
  saveError: string | null
  savedFlash: boolean
  isDirty: boolean
  onModeView: () => void
  onEnterVisual: () => void
  onEnterYaml: () => void
  onDiscard: () => void
  onSave: () => void
}

const toggleActiveCls = 'border-b-2 border-[var(--color-accent)] text-[var(--color-foreground)]'
const toggleInactiveCls = 'border-b-2 border-transparent text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)]'

export function ScenarioHeaderBar({
  title, description, mode, isEditing, saving, loadingRaw, detailLoaded,
  environments, selectedEnvPath, onEnvPathChange,
  result, running, onShowStats, onRun,
  saveError, savedFlash, isDirty,
  onModeView, onEnterVisual, onEnterYaml, onDiscard, onSave,
}: Props) {
  const steps = result?.steps ?? []
  const passed = steps.filter(s => s.passed && !s.skipped).length
  const failed = steps.filter(s => !s.passed && !s.skipped).length
  const skipped = steps.filter(s => s.skipped).length

  return (
    <div className="shrink-0 flex items-center gap-2 px-4 py-2.5 border-b border-[var(--color-border)] bg-[var(--color-card)]">
      <div className="flex-1 min-w-0">
        <h2 className="text-sm font-semibold text-[var(--color-foreground)] truncate">
          {title}
        </h2>
        {!isEditing && description && (
          <p className="text-xs text-[var(--color-muted-foreground)] truncate">{description}</p>
        )}
      </div>

      {/* View / Edit / YAML toggle — underline-style tabs */}
      <div className="shrink-0 flex text-xs">
        <button type="button" onClick={onModeView} disabled={mode === 'view' || saving}
          className={cn('px-3 py-1 transition-colors', mode === 'view' ? toggleActiveCls : toggleInactiveCls)}>
          View
        </button>
        <button type="button" onClick={onEnterVisual} disabled={mode === 'visual' || saving || !detailLoaded}
          className={cn('px-3 py-1 transition-colors', mode === 'visual' ? toggleActiveCls : toggleInactiveCls)}>
          Edit
        </button>
        <button type="button" onClick={onEnterYaml} disabled={mode === 'yaml' || loadingRaw || saving}
          className={cn('px-3 py-1 transition-colors flex items-center gap-1', mode === 'yaml' ? toggleActiveCls : toggleInactiveCls)}>
          {loadingRaw ? <Loader2 size={10} className="animate-spin" /> : null}
          YAML
        </button>
      </div>

      {/* Controls */}
      {!isEditing ? (
        <>
          {environments.length > 0 && (
            <select value={selectedEnvPath} onChange={e => onEnvPathChange(e.target.value)} title="Environment for this run"
              className={cn('shrink-0 text-xs rounded px-2 py-0.5 border outline-none cursor-pointer',
                'bg-[var(--color-input)] border-[var(--color-input-border)]',
                selectedEnvPath ? 'text-[var(--color-accent)]' : 'text-[var(--color-muted-foreground)]')}>
              <option value="">No environment</option>
              {environments.map(env => <option key={env.path} value={env.path}>{env.name}</option>)}
            </select>
          )}
          {result && !running && (
            <div className="shrink-0 flex items-center gap-2">
              <span className={cn('text-xs font-semibold px-2 py-0.5 rounded',
                result.passed ? 'bg-emerald-500/15 text-emerald-400' : 'bg-red-500/15 text-red-400')}>
                {result.passed ? 'PASSED' : 'FAILED'}
              </span>
              <span className="text-xs text-[var(--color-muted-foreground)]">
                {passed}/{passed + failed + skipped} · {result.durationMs} ms
              </span>
              <button
                type="button"
                onClick={onShowStats}
                title="View run statistics"
                className="p-1 rounded text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-muted)] transition-colors">
                <Clock size={13} />
              </button>
            </div>
          )}
          <button type="button" onClick={onRun} disabled={running || !detailLoaded}
            className={cn('shrink-0 flex items-center gap-1.5 px-3 py-1.5 rounded text-xs font-semibold transition-opacity',
              'bg-[var(--color-accent)] text-white',
              (running || !detailLoaded) ? 'opacity-50 cursor-not-allowed' : 'hover:opacity-90')}>
            {running ? <><Loader2 size={12} className="animate-spin" /> Running…</> : <><Play size={12} /> Run</>}
          </button>
        </>
      ) : (
        <>
          {saveError && <span className="text-xs text-red-400 shrink-0 max-w-[200px] truncate" title={saveError}>{saveError}</span>}
          {savedFlash ? (
            <span className="shrink-0 flex items-center gap-1.5 px-3 py-1.5 rounded text-xs font-semibold text-emerald-400 bg-emerald-500/10">
              <CheckCircle2 size={12} /> Saved
            </span>
          ) : (
            <>
              <button type="button" onClick={onDiscard} disabled={saving}
                className="shrink-0 flex items-center gap-1 px-2.5 py-1.5 rounded text-xs text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-muted)] transition-colors">
                <X size={11} /> Discard
              </button>
              <button type="button" onClick={onSave} disabled={saving || (mode === 'yaml' && !isDirty)}
                className={cn('shrink-0 flex items-center gap-1.5 px-3 py-1.5 rounded text-xs font-semibold transition-opacity',
                  'bg-[var(--color-accent)] text-white',
                  (saving || (mode === 'yaml' && !isDirty)) ? 'opacity-50 cursor-not-allowed' : 'hover:opacity-90')}>
                {saving ? <><Loader2 size={12} className="animate-spin" /> Saving…</> : <><Save size={12} /> Save</>}
              </button>
            </>
          )}
        </>
      )}
    </div>
  )
}
