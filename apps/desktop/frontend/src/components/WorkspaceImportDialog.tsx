import { useState } from 'react'
import { Upload, CheckCircle, AlertCircle, XCircle } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { ScanImportResult, ApplyImportResult } from '@/lib/bridge'
import { applyWorkspaceImport } from '@/lib/bridge'

type Strategy = 'skip' | 'overwrite' | 'rename'

type Props = {
  scan: ScanImportResult
  onClose: () => void
  onComplete: () => void   // triggers workspace refresh
}

export function WorkspaceImportDialog({ scan, onClose, onComplete }: Props) {
  const [strategy, setStrategy] = useState<Strategy>('skip')
  const [applying, setApplying] = useState(false)
  const [result, setResult] = useState<ApplyImportResult | null>(null)

  const hasConflicts = scan.conflicts.length > 0

  async function handleApply() {
    setApplying(true)
    try {
      const r = await applyWorkspaceImport(scan.zipPath, strategy)
      setResult(r)
      if (!r.error) onComplete()
    } finally {
      setApplying(false)
    }
  }

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-[var(--color-card)] border border-[var(--color-border)] rounded-lg shadow-xl w-[480px] max-w-[90vw] flex flex-col">

        {/* Header */}
        <div className="flex items-center gap-3 px-5 py-4 border-b border-[var(--color-border)]">
          <Upload size={16} className="text-emerald-400 shrink-0" />
          <span className="font-semibold text-sm text-[var(--color-foreground)]">
            {result ? 'Import complete' : 'Import Workspace'}
          </span>
        </div>

        {/* Body */}
        <div className="px-5 py-4 flex flex-col gap-4">
          {!result ? (
            <>
              {/* Summary */}
              <div className="text-sm text-[var(--color-foreground)]">
                Found <span className="font-semibold">{scan.totalFiles}</span> file{scan.totalFiles !== 1 ? 's' : ''} in the ZIP.
                {hasConflicts ? (
                  <span className="ml-1 text-amber-400 font-medium">
                    {scan.conflicts.length} conflict{scan.conflicts.length !== 1 ? 's' : ''} with existing files.
                  </span>
                ) : (
                  <span className="ml-1 text-emerald-400">No conflicts — all files are new.</span>
                )}
              </div>

              {/* Conflict details */}
              {hasConflicts && (
                <div className="bg-amber-500/5 border border-amber-500/20 rounded px-3 py-2 max-h-28 overflow-y-auto">
                  {scan.conflicts.slice(0, 20).map(c => (
                    <div key={c} className="text-[11px] font-mono text-amber-300/80 truncate">{c}</div>
                  ))}
                  {scan.conflicts.length > 20 && (
                    <div className="text-[11px] text-amber-400/60 mt-1">…and {scan.conflicts.length - 20} more</div>
                  )}
                </div>
              )}

              {/* Strategy selector */}
              {hasConflicts && (
                <div className="flex flex-col gap-2">
                  <span className="text-xs font-medium text-[var(--color-muted-foreground)] uppercase tracking-wide">For existing files:</span>
                  {(['skip', 'overwrite', 'rename'] as Strategy[]).map(s => (
                    <label key={s} className="flex items-center gap-2.5 cursor-pointer">
                      <input
                        type="radio"
                        name="strategy"
                        value={s}
                        checked={strategy === s}
                        onChange={() => setStrategy(s)}
                        className="accent-emerald-400"
                      />
                      <span className="text-sm text-[var(--color-foreground)]">
                        {s === 'skip' && 'Skip — keep existing files'}
                        {s === 'overwrite' && 'Overwrite — replace existing files'}
                        {s === 'rename' && 'Rename — import as filename-2.yaml, etc.'}
                      </span>
                    </label>
                  ))}
                </div>
              )}
            </>
          ) : (
            /* Result view */
            <div className="flex flex-col gap-3">
              {result.error ? (
                <div className="flex items-start gap-2 text-red-400">
                  <XCircle size={15} className="shrink-0 mt-0.5" />
                  <span className="text-sm">{result.error}</span>
                </div>
              ) : (
                <div className="flex flex-col gap-2 text-sm">
                  <Row icon={<CheckCircle size={14} className="text-emerald-400" />} label="Imported" value={result.imported} />
                  {result.renamed > 0 && <Row icon={<AlertCircle size={14} className="text-amber-400" />} label="Renamed" value={result.renamed} />}
                  {result.skipped > 0 && <Row icon={<AlertCircle size={14} className="text-[var(--color-muted-foreground)]" />} label="Skipped" value={result.skipped} />}
                </div>
              )}
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="flex justify-end gap-2 px-5 py-3 border-t border-[var(--color-border)]">
          {!result ? (
            <>
              <button
                type="button"
                onClick={onClose}
                disabled={applying}
                className="px-3 py-1.5 text-sm rounded text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-muted)] transition-colors"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={handleApply}
                disabled={applying}
                className={cn(
                  'px-4 py-1.5 text-sm rounded font-medium transition-colors',
                  'bg-emerald-500 hover:bg-emerald-400 text-white',
                  applying && 'opacity-60 cursor-not-allowed',
                )}
              >
                {applying ? 'Importing…' : `Import ${scan.totalFiles} file${scan.totalFiles !== 1 ? 's' : ''}`}
              </button>
            </>
          ) : (
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-1.5 text-sm rounded font-medium bg-[var(--color-muted)] hover:bg-[var(--color-border)] text-[var(--color-foreground)] transition-colors"
            >
              Done
            </button>
          )}
        </div>
      </div>
    </div>
  )
}

function Row({ icon, label, value }: { icon: React.ReactNode; label: string; value: number }) {
  return (
    <div className="flex items-center gap-2">
      {icon}
      <span className="text-[var(--color-muted-foreground)]">{label}</span>
      <span className="font-semibold text-[var(--color-foreground)] tabular-nums">{value}</span>
    </div>
  )
}
