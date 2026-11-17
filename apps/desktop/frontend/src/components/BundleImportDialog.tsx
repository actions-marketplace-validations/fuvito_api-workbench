import { useEffect, useRef, useState } from 'react'
import { FileUp, AlertTriangle, XCircle, FolderOpen, Loader2 } from 'lucide-react'
import { cn } from '@/lib/utils'
import { inputCls } from '@/lib/styles'
import { pickBundleFile, previewBundle, importBundle } from '@/lib/bridge'
import type { BundlePreview, BundleImportResult, BundleStrategy, BundleEntry } from '@/lib/bridge'

type Props = {
  onClose: () => void
  /** Called after files were written so the workspace tree can refresh. */
  onComplete: () => void
}

const STRATEGIES: Array<{ value: BundleStrategy; label: string }> = [
  { value: 'rename', label: 'Rename — keep both (name-2.yaml)' },
  { value: 'skip', label: 'Skip — keep existing files' },
  { value: 'overwrite', label: 'Overwrite — replace existing files' },
]

const KIND_LABEL: Record<BundleEntry['kind'], string> = { environment: 'Environment', request: 'Request', scenario: 'Scenario' }

const ACTION_STYLE: Record<BundleEntry['action'], string> = {
  create: 'text-emerald-400',
  overwrite: 'text-amber-400',
  rename: 'text-amber-400',
  skip: 'text-[var(--color-muted-foreground)]',
}

export function BundleImportDialog({ onClose, onComplete }: Props) {
  const [text, setText] = useState('')
  const [specFile, setSpecFile] = useState('')
  const [strategy, setStrategy] = useState<BundleStrategy>('rename')
  const [preview, setPreview] = useState<BundlePreview | null>(null)
  const [previewing, setPreviewing] = useState(false)
  const [importing, setImporting] = useState(false)
  const [result, setResult] = useState<BundleImportResult | null>(null)
  const [fileError, setFileError] = useState('')
  const requestId = useRef(0)

  // Re-check the bundle whenever the input changes; ignore answers from superseded requests.
  useEffect(() => {
    if (!text.trim()) {
      requestId.current++
      return
    }
    const id = ++requestId.current
    const timer = setTimeout(async () => {
      setPreviewing(true)
      try {
        const p = await previewBundle(text, specFile.trim(), strategy)
        if (id === requestId.current) setPreview(p)
      } catch (err) {
        if (id === requestId.current) {
          setPreview({ title: '', specFile: '', issues: [], entries: [], hasErrors: true, parseError: String(err) })
        }
      } finally {
        if (id === requestId.current) setPreviewing(false)
      }
    }, 250)
    return () => clearTimeout(timer)
  }, [text, specFile, strategy])

  async function handlePickFile() {
    setFileError('')
    const f = await pickBundleFile()
    if (f.error) { setFileError(f.error); return }
    if (f.path) setText(f.text)
  }

  async function handleImport() {
    setImporting(true)
    try {
      const r = await importBundle(text, specFile.trim(), strategy)
      setResult(r)
      if (!r.error) onComplete()
    } catch (err) {
      setResult({ created: 0, overwritten: 0, renamed: 0, skipped: 0, written: [], error: String(err) })
    } finally {
      setImporting(false)
    }
  }

  const hasText = text.trim() !== ''
  const current = hasText ? preview : null
  const errors = current?.issues.filter(i => i.severity === 'error') ?? []
  const warnings = current?.issues.filter(i => i.severity === 'warning') ?? []
  const entries = current?.entries ?? []
  const writing = entries.filter(e => e.action !== 'skip').length
  const canImport = hasText && !!current && !current.hasErrors && !previewing && !importing && writing > 0
  const hasConflicts = entries.some(e => e.action !== 'create')

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-6">
      <div className="flex flex-col w-full max-w-3xl max-h-[90vh] rounded-lg border border-[var(--color-border)] bg-[var(--color-card)] shadow-2xl">

        <div className="flex items-center gap-3 px-5 py-3 border-b border-[var(--color-border)] shrink-0">
          <FileUp size={16} className="text-[var(--color-accent)] shrink-0" />
          <span className="text-sm font-semibold text-[var(--color-foreground)]">
            {result && !result.error ? 'Import complete' : 'Import Bundle'}
          </span>
        </div>

        {result ? (
          <div className="px-5 py-4 flex flex-col gap-3 overflow-auto">
            {result.error ? (
              <div className="flex items-start gap-2 text-red-400 text-sm">
                <XCircle size={15} className="shrink-0 mt-0.5" />
                <span>{result.error}</span>
              </div>
            ) : (
              <>
                <div className="flex flex-wrap gap-x-5 gap-y-1 text-sm text-[var(--color-foreground)]">
                  <span><b>{result.created}</b> created</span>
                  {result.renamed > 0 && <span><b>{result.renamed}</b> renamed</span>}
                  {result.overwritten > 0 && <span><b>{result.overwritten}</b> overwritten</span>}
                  {result.skipped > 0 && <span><b>{result.skipped}</b> skipped</span>}
                </div>
                <ul className="text-xs font-mono text-[var(--color-muted-foreground)] space-y-0.5">
                  {result.written.map(p => <li key={p}>{p}</li>)}
                </ul>
              </>
            )}
          </div>
        ) : (
          <div className="px-5 py-4 flex flex-col gap-4 overflow-auto">
            <p className="text-xs text-[var(--color-muted-foreground)]">
              Paste a bundle — one document with environments, requests and scenarios, for example the reply from an AI assistant — or open it from a file.
              Nothing is written until you click Import.
            </p>

            <div className="flex flex-col gap-1.5">
              <div className="flex items-center justify-between">
                <label htmlFor="bundle-text" className="text-xs font-medium text-[var(--color-muted-foreground)] uppercase tracking-wide">Bundle</label>
                <button
                  type="button"
                  onClick={handlePickFile}
                  className="flex items-center gap-1.5 text-xs text-[var(--color-accent)] hover:underline"
                >
                  <FolderOpen size={12} /> Open file…
                </button>
              </div>
              <textarea
                id="bundle-text"
                value={text}
                onChange={e => setText(e.target.value)}
                spellCheck={false}
                placeholder={'schemaVersion: "1"\nenvironments:\n  - name: staging\n    variables: [...]\nscenarios:\n  - title: Smoke test\n    steps: [...]'}
                className={cn(inputCls, 'w-full h-48 font-mono text-xs leading-relaxed resize-y p-3')}
              />
              {fileError && <span className="text-xs text-red-400">{fileError}</span>}
            </div>

            <div className="flex flex-col gap-1.5">
              <label htmlFor="bundle-spec" className="text-xs font-medium text-[var(--color-muted-foreground)] uppercase tracking-wide">
                Spec file (optional)
              </label>
              <input
                id="bundle-spec"
                type="text"
                value={specFile}
                onChange={e => setSpecFile(e.target.value)}
                placeholder={current?.specFile ? `Bundle uses: ${current.specFile}` : 'Path or URL of the OpenAPI spec'}
                className={cn(inputCls, 'w-full font-mono px-3 py-1.5')}
              />
              <span className="text-[11px] text-[var(--color-muted-foreground)]">
                Replaces the spec in every imported scenario — use it to point at a local file.
              </span>
            </div>

            {hasText && (
              <div className="flex flex-col gap-3" aria-live="polite">
                {previewing && !current && (
                  <span className="flex items-center gap-2 text-xs text-[var(--color-muted-foreground)]">
                    <Loader2 size={12} className="animate-spin" /> Checking…
                  </span>
                )}

                {current?.parseError && (
                  <div className="flex items-start gap-2 text-red-400 text-xs rounded border border-red-500/30 bg-red-500/5 px-3 py-2">
                    <XCircle size={13} className="shrink-0 mt-0.5" />
                    <span className="whitespace-pre-wrap">{current.parseError}</span>
                  </div>
                )}

                {errors.length > 0 && (
                  <IssueList title={`${errors.length} problem${errors.length !== 1 ? 's' : ''} to fix`} tone="error" issues={errors} />
                )}
                {warnings.length > 0 && (
                  <IssueList title={`${warnings.length} warning${warnings.length !== 1 ? 's' : ''}`} tone="warning" issues={warnings} />
                )}

                {entries.length > 0 && (
                  <div className="flex flex-col gap-2">
                    <div className="flex items-center gap-2 text-xs font-medium text-[var(--color-muted-foreground)] uppercase tracking-wide">
                      Files
                      {current?.title && <span className="normal-case font-normal">— {current.title}</span>}
                    </div>
                    <ul className="rounded border border-[var(--color-border)] divide-y divide-[var(--color-border)] text-xs">
                      {entries.map(e => (
                        <li key={e.path + e.kind} className="flex items-center gap-3 px-3 py-1.5">
                          <span className="w-20 shrink-0 text-[var(--color-muted-foreground)]">{KIND_LABEL[e.kind]}</span>
                          <span className="flex-1 min-w-0 truncate font-mono text-[var(--color-foreground)]" title={e.path}>{e.path}</span>
                          <span className={cn('shrink-0 font-medium', ACTION_STYLE[e.action])}>{e.action}</span>
                        </li>
                      ))}
                    </ul>
                  </div>
                )}

                {hasConflicts && (
                  <fieldset className="flex flex-col gap-1.5">
                    <legend className="text-xs font-medium text-[var(--color-muted-foreground)] uppercase tracking-wide mb-1">
                      Files that already exist
                    </legend>
                    {STRATEGIES.map(s => (
                      <label key={s.value} className="flex items-center gap-2.5 cursor-pointer text-sm text-[var(--color-foreground)]">
                        <input
                          type="radio"
                          name="bundle-strategy"
                          value={s.value}
                          checked={strategy === s.value}
                          onChange={() => setStrategy(s.value)}
                          className="accent-[var(--color-accent)]"
                        />
                        {s.label}
                      </label>
                    ))}
                  </fieldset>
                )}
              </div>
            )}
          </div>
        )}

        <div className="flex items-center justify-end gap-2 px-5 py-3 border-t border-[var(--color-border)] shrink-0">
          {result ? (
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-1.5 text-sm rounded font-medium bg-[var(--color-muted)] hover:bg-[var(--color-border)] text-[var(--color-foreground)] transition-colors"
            >
              Done
            </button>
          ) : (
            <>
              <button
                type="button"
                onClick={onClose}
                disabled={importing}
                className="px-3 py-1.5 text-sm rounded text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-muted)] transition-colors"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={handleImport}
                disabled={!canImport}
                className={cn(
                  'px-4 py-1.5 text-sm rounded font-medium bg-[var(--color-accent)] text-white transition-opacity',
                  canImport ? 'hover:opacity-90' : 'opacity-40 cursor-not-allowed',
                )}
              >
                {importing ? 'Importing…' : writing > 0 ? `Import ${writing} file${writing !== 1 ? 's' : ''}` : 'Import'}
              </button>
            </>
          )}
        </div>
      </div>
    </div>
  )
}

function IssueList({ title, tone, issues }: { title: string; tone: 'error' | 'warning'; issues: BundlePreview['issues'] }) {
  const isError = tone === 'error'
  return (
    <div className={cn('rounded border px-3 py-2 text-xs', isError ? 'border-red-500/30 bg-red-500/5' : 'border-amber-500/30 bg-amber-500/5')}>
      <div className={cn('flex items-center gap-1.5 font-semibold mb-1', isError ? 'text-red-400' : 'text-amber-400')}>
        {isError ? <XCircle size={13} /> : <AlertTriangle size={13} />}
        {title}
      </div>
      <ul className="space-y-1 max-h-40 overflow-y-auto">
        {issues.map((i, n) => (
          <li key={n} className="text-[var(--color-foreground)]">
            {i.where && <span className="font-mono text-[var(--color-muted-foreground)]">{i.where}: </span>}
            {i.message}
          </li>
        ))}
      </ul>
    </div>
  )
}
