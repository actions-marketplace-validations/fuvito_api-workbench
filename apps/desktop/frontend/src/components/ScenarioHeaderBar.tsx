import { Play, Loader2, CheckCircle2, Save, X, Clock, FileCode, Square } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { ScenarioResult, EnvInfo } from '@/lib/bridge'

type Props = {
  title: string
  saving: boolean
  detailLoaded: boolean
  environments: EnvInfo[]
  selectedEnvPath: string
  onEnvPathChange: (path: string) => void
  onTitleChange: (title: string) => void
  result: ScenarioResult | null
  running: boolean
  onShowStats: () => void
  onRun: () => void
  onCancel?: () => void
  saveError: string | null
  savedFlash: boolean
  isDirty: boolean
  onDiscard: () => void
  onSave: () => void
  onShowYaml: () => void
}

export function ScenarioHeaderBar({
  title, saving, detailLoaded,
  environments, selectedEnvPath, onEnvPathChange, onTitleChange,
  result, running, onShowStats, onRun, onCancel,
  saveError, savedFlash, isDirty,
  onDiscard, onSave, onShowYaml,
}: Props) {
  const steps = result?.steps ?? []
  const passed = steps.filter(s => s.passed && !s.skipped).length
  const failed = steps.filter(s => !s.passed && !s.skipped).length
  const skipped = steps.filter(s => s.skipped).length

  return (
    <div className="shrink-0 flex items-center gap-2 px-4 py-2.5 border-b border-[var(--color-border)] bg-[var(--color-card)]">
      <input
        value={title}
        onChange={e => onTitleChange(e.target.value)}
        placeholder="Scenario title"
        className="flex-1 min-w-0 bg-transparent border-b border-[var(--color-border)] hover:border-[var(--color-muted-foreground)] focus:border-[var(--color-accent)] outline-none text-sm font-semibold text-[var(--color-foreground)] py-0.5 transition-colors"
      />

      <button
        type="button"
        onClick={onShowYaml}
        title="View / edit YAML"
        className="shrink-0 p-1.5 rounded text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-muted)] transition-colors"
      >
        <FileCode size={14} />
      </button>

      {environments.length > 0 && (
        <select
          value={selectedEnvPath}
          onChange={e => onEnvPathChange(e.target.value)}
          title="Environment for this run"
          className={cn(
            'shrink-0 text-xs rounded px-2 py-0.5 border outline-none cursor-pointer',
            'bg-[var(--color-input)] border-[var(--color-input-border)]',
            selectedEnvPath ? 'text-[var(--color-accent)]' : 'text-[var(--color-muted-foreground)]',
          )}
        >
          <option value="">No environment</option>
          {environments.map(env => <option key={env.path} value={env.path}>{env.name}</option>)}
        </select>
      )}

      {result && !running && (
        <div className="shrink-0 flex items-center gap-2">
          <span className={cn(
            'text-xs font-semibold px-2 py-0.5 rounded',
            result.passed ? 'bg-emerald-500/15 text-emerald-400' : 'bg-red-500/15 text-red-400',
          )}>
            {result.passed ? 'PASSED' : 'FAILED'}
          </span>
          <span className="text-xs text-[var(--color-muted-foreground)]">
            {passed}/{passed + failed + skipped} · {result.durationMs} ms
          </span>
          <button
            type="button"
            onClick={onShowStats}
            title="View run statistics"
            className="p-1 rounded text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-muted)] transition-colors"
          >
            <Clock size={13} />
          </button>
        </div>
      )}

      {isDirty && !savedFlash && (
        <>
          {saveError && (
            <span className="text-xs text-red-400 shrink-0 max-w-[160px] truncate" title={saveError}>
              {saveError}
            </span>
          )}
          <button
            type="button"
            onClick={onDiscard}
            disabled={saving}
            className="shrink-0 flex items-center gap-1 px-2.5 py-1.5 rounded text-xs text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-muted)] transition-colors"
          >
            <X size={11} /> Discard
          </button>
          <button
            type="button"
            onClick={onSave}
            disabled={saving}
            className={cn(
              'shrink-0 flex items-center gap-1.5 px-3 py-1.5 rounded text-xs font-semibold transition-opacity bg-[var(--color-accent)] text-white',
              saving ? 'opacity-50 cursor-not-allowed' : 'hover:opacity-90',
            )}
          >
            {saving
              ? <><Loader2 size={12} className="animate-spin" /> Saving…</>
              : <><Save size={12} /> Save</>}
          </button>
        </>
      )}

      {savedFlash && (
        <span className="shrink-0 flex items-center gap-1.5 px-3 py-1.5 rounded text-xs font-semibold text-emerald-400 bg-emerald-500/10">
          <CheckCircle2 size={12} /> Saved
        </span>
      )}

      {running && onCancel ? (
        <button
          type="button"
          onClick={onCancel}
          className="shrink-0 flex items-center gap-1.5 px-3 py-1.5 rounded text-xs font-semibold transition-colors bg-amber-500/15 text-amber-400 hover:bg-amber-500/25"
        >
          <Square size={12} /> Stop
        </button>
      ) : (
        <button
          type="button"
          onClick={onRun}
          disabled={running || !detailLoaded}
          className={cn(
            'shrink-0 flex items-center gap-1.5 px-3 py-1.5 rounded text-xs font-semibold transition-opacity bg-[var(--color-accent)] text-white',
            (running || !detailLoaded) ? 'opacity-50 cursor-not-allowed' : 'hover:opacity-90',
          )}
        >
          {running
            ? <><Loader2 size={12} className="animate-spin" /> Running…</>
            : <><Play size={12} /> Run</>}
        </button>
      )}
    </div>
  )
}
