import { useState, useEffect, useRef } from 'react'
import { X, Eye, EyeOff, Trash2, Lock, CopyPlus, ShieldCheck, CheckCircle2, Loader2 } from 'lucide-react'
import { cn } from '@/lib/utils'
import { inputCls as baseInputCls } from '@/lib/styles'
import { getEnvironment, saveEnvironment, deleteEnvironment, getEnvironmentRaw, saveEnvironmentRaw, validateEnvironmentYAML } from '@/lib/bridge'
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
  prefilledVarName?: string
}

function envToYAML(name: string, variables: EnvVar[]): string {
  const lines: string[] = ['schemaVersion: "1"', `name: ${yamlScalar(name)}`, '', 'variables:']
  if (variables.length === 0) lines.push('  []')
  for (const v of variables) {
    lines.push(`  - name: ${yamlScalar(v.name)}`)
    lines.push(`    value: ${yamlScalar(v.value)}`)
    if (v.secret) lines.push(`    secret: true`)
  }
  return lines.join('\n') + '\n'
}

function yamlScalar(s: string): string {
  if (s === '') return "''"
  if (/[:#[\]{},|>&*!%@`'"]/.test(s) || /^\s|\s$/.test(s) || s.includes('\n')) {
    return "'" + s.replace(/'/g, "''") + "'"
  }
  return s
}

export function EnvironmentEditor({ envPath, workspacePath, onClose, onSaved, onDeleted, prefilledVarName }: Props) {
  const [mode, setMode] = useState<'form' | 'yaml'>('form')
  const [name, setName] = useState('')
  const [variables, setVariables] = useState<EnvVar[]>([])
  const [yamlText, setYamlText] = useState('')
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [revealed, setRevealed] = useState<Set<string>>(new Set())
  const [confirmDelete, setConfirmDelete] = useState(false)
  const [yamlCheckState, setYamlCheckState] = useState<'idle' | 'checking' | 'valid'>('idle')
  const textareaRef = useRef<HTMLTextAreaElement>(null)

  const isNew = !envPath

  /* eslint-disable react/set-state-in-effect */
  useEffect(() => {
    setMode('form')
    if (!envPath) {
      setName('')
      setVariables(prefilledVarName
        ? [{ id: crypto.randomUUID(), name: prefilledVarName, value: '', secret: false }]
        : [])
      setYamlText('')
      setError('')
      return
    }
    setLoading(true)
    setError('')
    getEnvironment(envPath)
      .then(env => {
        setName(env.name)
        const vars = env.variables.map(v => ({ id: crypto.randomUUID(), ...v }))
        if (prefilledVarName && !vars.some(v => v.name === prefilledVarName)) {
          setVariables([...vars, { id: crypto.randomUUID(), name: prefilledVarName, value: '', secret: false }])
        } else {
          setVariables(vars)
        }
      })
      .catch(err => setError(String(err)))
      .finally(() => setLoading(false))
  }, [envPath, prefilledVarName])
  /* eslint-enable react/set-state-in-effect */

  async function handleCheckYaml() {
    setYamlCheckState('checking')
    setError('')
    const err = await validateEnvironmentYAML(yamlText)
    if (err) {
      setError(err)
      setYamlCheckState('idle')
    } else {
      setYamlCheckState('valid')
      setTimeout(() => setYamlCheckState('idle'), 2000)
    }
  }

  async function handleSwitchToYAML() {
    setError('')
    if (envPath) {
      try {
        const raw = await getEnvironmentRaw(envPath)
        setYamlText(raw)
      } catch {
        setYamlText(envToYAML(name, variables))
      }
    } else {
      setYamlText(envToYAML(name, variables))
    }
    setMode('yaml')
    setTimeout(() => textareaRef.current?.focus(), 0)
  }

  async function handleSwitchToForm() {
    setError('')
    try {
      const env = await parseYAMLToForm(yamlText)
      setName(env.name)
      setVariables(env.variables.map(v => ({ id: crypto.randomUUID(), ...v })))
      setMode('form')
    } catch (err) {
      setError('Cannot switch to form — fix YAML errors first: ' + String(err))
    }
  }

  async function parseYAMLToForm(text: string) {
    // We use the Go serialization parser by writing to a temp parse call.
    // Since we don't have a standalone parse endpoint, do a lightweight client-side parse
    // of the YAML structure to extract name + variables.
    const lines = text.split('\n')
    let envName = ''
    const vars: Array<{ name: string; value: string; secret: boolean }> = []
    let inVariables = false
    let currentVar: { name?: string; value?: string; secret?: boolean } | null = null

    for (const raw of lines) {
      const line = raw.trimEnd()
      if (line.startsWith('name:') && !inVariables) {
        envName = line.replace(/^name:\s*/, '').replace(/^['"]|['"]$/g, '')
      } else if (line.startsWith('variables:')) {
        inVariables = true
      } else if (inVariables) {
        const trimmed = line.trimStart()
        if (trimmed.startsWith('- name:')) {
          if (currentVar?.name !== undefined) vars.push({ name: currentVar.name ?? '', value: currentVar.value ?? '', secret: !!currentVar.secret })
          currentVar = { name: trimmed.replace(/^- name:\s*/, '').replace(/^['"]|['"]$/g, '') }
        } else if (trimmed.startsWith('value:') && currentVar) {
          currentVar.value = trimmed.replace(/^value:\s*/, '').replace(/^['"]|['"]$/g, '')
        } else if (trimmed.startsWith('secret:') && currentVar) {
          currentVar.secret = trimmed.includes('true')
        }
      }
    }
    if (currentVar?.name !== undefined) vars.push({ name: currentVar.name ?? '', value: currentVar.value ?? '', secret: !!currentVar.secret })

    if (!envName) throw new Error('missing required field: name')
    return { name: envName, variables: vars }
  }

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
    setSaving(true)
    setError('')
    try {
      if (mode === 'yaml') {
        if (!envPath) { setError('Save via the form first to create the file, then edit YAML.'); setSaving(false); return }
        const saved = await saveEnvironmentRaw(envPath, yamlText)
        onSaved(saved)
      } else {
        if (!name.trim()) { setError('Environment name is required.'); setSaving(false); return }
        if (!workspacePath && isNew) { setError('Open a workspace first.'); setSaving(false); return }
        const saved = await saveEnvironment({
          path: envPath ?? '',
          name: name.trim(),
          variables: variables.map(({ name: n, value, secret }) => ({ name: n, value, secret })),
        })
        onSaved(saved)
      }
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
        <div className="flex items-center justify-between px-4 pt-2 border-b border-[var(--color-border)] shrink-0">
          <div className="flex items-center">
            <button
              type="button"
              onClick={() => mode === 'yaml' ? handleSwitchToForm() : undefined}
              className={cn(
                'px-3 py-1.5 text-xs font-medium border-b-2 transition-colors',
                mode === 'form'
                  ? 'border-[var(--color-accent)] text-[var(--color-foreground)]'
                  : 'border-transparent text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] cursor-pointer',
              )}
            >
              {isNew ? 'New Environment' : 'Form'}
            </button>
            {!isNew && (
              <button
                type="button"
                onClick={() => mode === 'form' ? handleSwitchToYAML() : undefined}
                className={cn(
                  'px-3 py-1.5 text-xs font-medium border-b-2 transition-colors',
                  mode === 'yaml'
                    ? 'border-[var(--color-accent)] text-[var(--color-foreground)]'
                    : 'border-transparent text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] cursor-pointer',
                )}
              >
                YAML
              </button>
            )}
          </div>
          <div className="pb-2">
            <button type="button" onClick={onClose} className="text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors">
              <X size={15} />
            </button>
          </div>
        </div>

        {/* Body */}
        <div className="flex flex-col gap-4 p-4 overflow-auto flex-1">
          {loading && (
            <p className="text-xs text-[var(--color-muted-foreground)] animate-pulse">Loading…</p>
          )}

          {!loading && mode === 'yaml' && (
            <div className="flex flex-col gap-2 flex-1">
              <textarea
                ref={textareaRef}
                value={yamlText}
                onChange={e => { setYamlText(e.target.value); if (yamlCheckState !== 'idle') setYamlCheckState('idle'); if (error) setError('') }}
                spellCheck={false}
                className={cn(
                  'flex-1 w-full font-mono text-xs leading-relaxed resize-none rounded border border-[var(--color-border)]',
                  'bg-[var(--color-muted)] text-[var(--color-foreground)] p-3 outline-none',
                  'focus:border-[var(--color-accent)] transition-colors min-h-[300px]',
                )}
              />
              <div className="flex items-center">
                <button
                  type="button"
                  onClick={handleCheckYaml}
                  disabled={yamlCheckState === 'checking'}
                  className={cn(
                    'flex items-center gap-1.5 px-3 py-1 rounded text-xs font-medium transition-colors',
                    yamlCheckState === 'valid'
                      ? 'bg-emerald-500/15 text-emerald-400'
                      : 'text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] border border-[var(--color-border)] hover:border-[var(--color-foreground)]',
                    yamlCheckState === 'checking' && 'opacity-50 cursor-not-allowed',
                  )}
                >
                  {yamlCheckState === 'checking' ? (
                    <><Loader2 size={11} className="animate-spin" /> Checking…</>
                  ) : yamlCheckState === 'valid' ? (
                    <><CheckCircle2 size={11} /> Valid</>
                  ) : (
                    <><ShieldCheck size={11} /> Check YAML</>
                  )}
                </button>
              </div>
            </div>
          )}

          {!loading && mode === 'form' && (
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
            </>
          )}

          {error && (
            <p className="text-xs text-[var(--color-destructive)]">{error}</p>
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

const inputCls = cn(baseInputCls, 'w-full font-mono px-2 py-1')
