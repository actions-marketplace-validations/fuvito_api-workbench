import { useState } from 'react'
import { X, ShieldCheck, Save, Loader2, CheckCircle2 } from 'lucide-react'
import { validateScenarioYAML } from '@/lib/bridge'
import { YamlEditorPane } from './ScenarioYamlPane'
import { cn } from '@/lib/utils'

type Props = {
  theme: 'dark' | 'light'
  initialContent: string
  onSave: (content: string) => Promise<string | null>
  onClose: () => void
}

export function ScenarioYamlDialog({ theme, initialContent, onSave, onClose }: Props) {
  const [content, setContent] = useState(initialContent)
  const [error, setError] = useState<string | null>(null)
  const [checkState, setCheckState] = useState<'idle' | 'checking' | 'valid'>('idle')
  const [saving, setSaving] = useState(false)
  const [confirmClose, setConfirmClose] = useState(false)
  const isDirty = content !== initialContent

  function handleClose() {
    if (isDirty) { setConfirmClose(true); return }
    onClose()
  }

  async function handleCheck() {
    setCheckState('checking')
    setError(null)
    const err = await validateScenarioYAML(content)
    if (err) { setError(err); setCheckState('idle') }
    else { setCheckState('valid'); setTimeout(() => setCheckState('idle'), 2000) }
  }

  async function handleSave() {
    setSaving(true)
    setError(null)
    const err = await onSave(content)
    if (err) setError(err)
    setSaving(false)
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60" onClick={handleClose}>
      <div
        className="relative flex flex-col w-[800px] max-w-[90vw] h-[600px] max-h-[85vh] rounded-lg border border-[var(--color-border)] bg-[var(--color-card)] shadow-2xl overflow-hidden"
        onClick={e => e.stopPropagation()}
      >
        <div className="shrink-0 flex items-center gap-2 px-4 py-2.5 border-b border-[var(--color-border)]">
          <span className="flex-1 text-sm font-semibold text-[var(--color-foreground)]">YAML</span>
          <button
            type="button"
            onClick={handleCheck}
            disabled={checkState === 'checking'}
            className={cn(
              'flex items-center gap-1.5 px-2.5 py-1.5 rounded text-xs font-medium transition-colors',
              checkState === 'valid'
                ? 'bg-emerald-500/15 text-emerald-400'
                : 'text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-muted)]',
              checkState === 'checking' && 'opacity-50 cursor-not-allowed',
            )}
          >
            {checkState === 'checking'
              ? <><Loader2 size={11} className="animate-spin" /> Checking…</>
              : checkState === 'valid'
                ? <><CheckCircle2 size={11} /> Valid</>
                : <><ShieldCheck size={11} /> Check</>}
          </button>
          <button
            type="button"
            onClick={handleSave}
            disabled={saving || !isDirty}
            className={cn(
              'flex items-center gap-1.5 px-3 py-1.5 rounded text-xs font-semibold transition-opacity bg-[var(--color-accent)] text-white',
              (saving || !isDirty) ? 'opacity-50 cursor-not-allowed' : 'hover:opacity-90',
            )}
          >
            {saving
              ? <><Loader2 size={12} className="animate-spin" /> Saving…</>
              : <><Save size={12} /> Save</>}
          </button>
          <button
            type="button"
            onClick={handleClose}
            className="p-1.5 rounded text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-muted)] transition-colors"
          >
            <X size={14} />
          </button>
        </div>

        <YamlEditorPane
          theme={theme}
          value={content}
          error={error}
          onChange={v => {
            setContent(v)
            if (checkState !== 'idle') setCheckState('idle')
            if (error) setError(null)
          }}
        />
      </div>

      {confirmClose && (
        <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/40" onClick={e => e.stopPropagation()}>
          <div className="bg-[var(--color-card)] border border-[var(--color-border)] rounded-lg shadow-xl px-6 py-5 flex flex-col gap-4 max-w-sm w-full mx-4">
            <p className="text-sm text-[var(--color-foreground)]">You have unsaved YAML edits. Close and discard them?</p>
            <div className="flex justify-end gap-2">
              <button type="button" onClick={() => setConfirmClose(false)} className="px-3 py-1.5 text-sm rounded text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-muted)] transition-colors">Cancel</button>
              <button type="button" onClick={onClose} className="px-4 py-1.5 text-sm rounded font-medium bg-red-500/15 text-red-400 hover:bg-red-500/25 transition-colors">Discard</button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
