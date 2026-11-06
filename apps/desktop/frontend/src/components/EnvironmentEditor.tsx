import { useState, useEffect } from 'react'
import { X, Eye, EyeOff, Trash2, Lock, CopyPlus } from 'lucide-react'
import { cn } from '@/lib/utils'
import { getEnvironment, saveEnvironment, deleteEnvironment } from '@/lib/bridge'
import type { EnvInfo } from '@/lib/bridge'

interface EnvVar {
  id: string
  name: string
  value: string
  secret: boolean
}

type Props = {
  envPath: string | null
  workspacePath: string
  onClose: () => void
  onSaved: (env: EnvInfo) => void
  onDeleted: () => void
}

export function EnvironmentEditor({ envPath, workspacePath, onClose, onSaved, onDeleted }: Props) {
  const [name, setName] = useState('')
  const [variables, setVariables] = useState<EnvVar[]>([])
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [revealed, setRevealed] = useState<Set<string>>(new Set())
  const [confirmDelete, setConfirmDelete] = useState(false)

  const isNew = !envPath

  /* eslint-disable react/set-state-in-effect */
  useEffect(() => {
    if (!envPath) {
      setName('')
      setVariables([])
      setError('')
      return
    }
    setLoading(true)
    setError('')
    getEnvironment(envPath)
      .then(env => {
        setName(env.name)
        setVariables(env.variables.map(v => ({ id: crypto.randomUUID(), ...v })))
      })
      .catch(err => setError(String(err)))
      .finally(() => setLoading(false))
  }, [envPath])
  /* eslint-enable react/set-state-in-effect */

  function addVariable() {
    setVariables(prev => [...prev, { id: crypto.randomUUID(), name: '', value: '', secret: false }])
  }

  function updateVar(id: string, field: keyof Omit<EnvVar, 'id'>, value: string | boolean) {
    setVariables(prev => prev.map(v => v.id === id ? { ...v, [field]: value } : v))
  }

  function removeVar(id: string) {
    setVariables(prev => prev.filter(v => v.id !== id))
    setRevealed(prev => { const next = new Set(prev); next.delete(id); return next })
  }

  function toggleReveal(id: string) {
    setRevealed(prev => {
      const next = new Set(prev)
      if (next.has(id)) { next.delete(id) } else { next.add(id) }
      return next
    })
  }

  async function handleSave() {
    if (!name.trim()) { setError('Environment name is required.'); return }
    if (!workspacePath && isNew) { setError('Open a workspace first.'); return }
    setSaving(true)
    setError('')
    try {
      const saved = await saveEnvironment({
        path: envPath ?? '',
        name: name.trim(),
        variables: variables.map(({ name: n, value, secret }) => ({ name: n, value, secret })),
      })
      onSaved(saved)
    } catch (err) {
      setError(String(err))
    } finally {
      setSaving(false)
    }
  }

  async function handleDuplicate() {
    setSaving(true)
    setError('')
    try {
      const saved = await saveEnvironment({
        path: '',
        name: name.trim() + ' Copy',
        variables: variables.map(({ name: n, value, secret }) => ({ name: n, value, secret })),
      })
      onSaved(saved)
    } catch (err) {
      setError(String(err))
    } finally {
      setSaving(false)
    }
  }

  async function handleDelete() {
    if (!envPath) return
    if (!confirmDelete) { setConfirmDelete(true); return }
    try {
      await deleteEnvironment(envPath)
      onDeleted()
    } catch (err) {
      setError(String(err))
      setConfirmDelete(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60">
      <div className={cn(
        'w-[560px] max-h-[80vh] rounded-lg border border-[var(--color-border)] bg-[var(--color-card)]',
        'shadow-2xl flex flex-col',
      )}>
        {/* Header */}
        <div className="flex items-center justify-between px-4 py-3 border-b border-[var(--color-border)] shrink-0">
          <span className="text-xs font-semibold text-[var(--color-foreground)]">
            {isNew ? 'New Environment' : 'Edit Environment'}
          </span>
          <button type="button" onClick={onClose} className="text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors">
            <X size={15} />
          </button>
        </div>

        {/* Body */}
        <div className="flex flex-col gap-4 p-4 overflow-auto flex-1">
          {loading && (
            <p className="text-xs text-[var(--color-muted-foreground)] animate-pulse">Loading…</p>
          )}

          {!loading && (
            <>
              {/* Name */}
              <div className="flex items-center gap-3">
                <span className="w-28 shrink-0 text-xs text-[var(--color-muted-foreground)]">Name</span>
                <input
                  type="text"
                  value={name}
                  onChange={e => setName(e.target.value)}
                  placeholder="e.g. Local Dev"
                  className={inputCls}
                />
              </div>

              {/* Variables */}
              <div className="flex flex-col gap-2">
                <div className="flex items-center justify-between">
                  <h3 className="text-[11px] font-semibold uppercase tracking-widest text-[var(--color-muted-foreground)]">
                    Variables
                  </h3>
                </div>

                {variables.length > 0 && (
                  <div className="flex flex-col gap-1">
                    {/* Column headers */}
                    <div className="grid grid-cols-[1fr_1fr_auto_auto_auto] gap-1.5 items-center mb-0.5">
                      <span className="text-[10px] text-[var(--color-muted-foreground)] uppercase tracking-widest font-semibold">Name</span>
                      <span className="text-[10px] text-[var(--color-muted-foreground)] uppercase tracking-widest font-semibold">Value</span>
                      <span className="w-6" />
                      <span className="w-6" />
                      <span className="w-4" />
                    </div>

                    {variables.map(v => (
                      <div key={v.id} className="grid grid-cols-[1fr_1fr_auto_auto_auto] gap-1.5 items-center">
                        <input
                          type="text"
                          value={v.name}
                          onChange={e => updateVar(v.id, 'name', e.target.value)}
                          placeholder="VARIABLE_NAME"
                          className={inputCls}
                        />
                        <input
                          type={v.secret && !revealed.has(v.id) ? 'password' : 'text'}
                          value={v.value}
                          onChange={e => updateVar(v.id, 'value', e.target.value)}
                          placeholder={v.secret ? '••••••••' : 'value'}
                          className={inputCls}
                        />
                        {/* Reveal toggle (only for secret) */}
                        <button
                          type="button"
                          onClick={() => toggleReveal(v.id)}
                          disabled={!v.secret}
                          className={cn(
                            'w-6 h-6 flex items-center justify-center rounded transition-colors',
                            v.secret
                              ? 'text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)]'
                              : 'text-transparent cursor-default',
                          )}
                          title={v.secret ? (revealed.has(v.id) ? 'Hide value' : 'Reveal value') : undefined}
                        >
                          {revealed.has(v.id) ? <EyeOff size={12} /> : <Eye size={12} />}
                        </button>
                        {/* Secret toggle */}
                        <button
                          type="button"
                          onClick={() => { updateVar(v.id, 'secret', !v.secret); if (!v.secret) setRevealed(prev => { const n = new Set(prev); n.delete(v.id); return n }) }}
                          className={cn(
                            'w-6 h-6 flex items-center justify-center rounded transition-colors',
                            v.secret
                              ? 'text-[var(--color-accent)]'
                              : 'text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)]',
                          )}
                          title={v.secret ? 'Mark as plain text' : 'Mark as secret'}
                        >
                          <Lock size={12} />
                        </button>
                        {/* Remove */}
                        <button
                          type="button"
                          onClick={() => removeVar(v.id)}
                          className="w-4 flex items-center justify-center text-[var(--color-muted-foreground)] hover:text-[var(--color-destructive)] transition-colors"
                        >
                          <X size={11} />
                        </button>
                      </div>
                    ))}
                  </div>
                )}

                <button
                  type="button"
                  onClick={addVariable}
                  className="self-start text-xs text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors"
                >
                  + Add variable
                </button>
              </div>

              {error && (
                <p className="text-xs text-[var(--color-destructive)]">{error}</p>
              )}
            </>
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center gap-2 px-4 py-3 border-t border-[var(--color-border)] shrink-0">
          {!isNew && (
            <button
              type="button"
              onClick={handleDuplicate}
              disabled={saving || loading}
              title="Duplicate this environment"
              className={cn(
                'text-xs rounded px-3 py-1.5 mr-auto flex items-center gap-1.5 transition-colors',
                'text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] border border-[var(--color-border)]',
                'disabled:opacity-40',
              )}
            >
              <CopyPlus size={11} />
              Duplicate
            </button>
          )}
          {!isNew && (
            <button
              type="button"
              onClick={handleDelete}
              className={cn(
                'text-xs rounded px-3 py-1.5 transition-colors',
                confirmDelete
                  ? 'bg-[var(--color-destructive)] text-white'
                  : 'text-[var(--color-destructive)] hover:bg-[var(--color-destructive)] hover:text-white border border-[var(--color-destructive)]',
              )}
            >
              <span className="flex items-center gap-1">
                <Trash2 size={11} />
                {confirmDelete ? 'Confirm delete' : 'Delete'}
              </span>
            </button>
          )}
          <div className="ml-auto flex items-center gap-2">
            <button
              type="button"
              onClick={onClose}
              className={cn(
                'text-xs rounded px-4 py-1.5',
                'bg-[var(--color-muted)] text-[var(--color-foreground)] border border-[var(--color-border)]',
                'hover:bg-[var(--color-border)] transition-colors',
              )}
            >
              Cancel
            </button>
            <button
              type="button"
              onClick={handleSave}
              disabled={saving || loading}
              className={cn(
                'text-xs rounded px-4 py-1.5 font-semibold',
                'bg-[var(--color-accent)] text-white hover:opacity-90 transition-opacity',
                'disabled:opacity-40',
              )}
            >
              {saving ? 'Saving…' : 'Save'}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

const inputCls = cn(
  'w-full bg-[var(--color-muted)] text-xs font-mono rounded px-2 py-1',
  'border border-[var(--color-border)] outline-none',
  'focus:border-[var(--color-accent)] text-[var(--color-foreground)]',
  'placeholder:text-[var(--color-muted-foreground)]',
)
