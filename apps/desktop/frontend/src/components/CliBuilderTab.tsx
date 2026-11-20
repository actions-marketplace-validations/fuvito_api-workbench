import { useEffect, useRef, useState } from 'react'
import { Terminal, Copy, Check, FolderOpen, FileText, AlertTriangle, XCircle, CheckCircle2 } from 'lucide-react'
import { cn } from '@/lib/utils'
import {
  getCliCatalog, getCliContext, buildCliCommand, pickCliFile, pickCliFolder,
} from '@/lib/bridge'
import type {
  CliCommand, CliInput, CliContext, CliBuildOutput, CliMessage, CliShell, CliFile, CliSuggestion,
} from '@/lib/bridge'

export type CliBuilderPreset = {
  id: number
  command: string
  values?: Record<string, string>
}

type Props = {
  preset?: CliBuilderPreset | null
}

const SHELL_LABELS: Record<CliShell, string> = { bash: 'bash', powershell: 'PowerShell', cmd: 'cmd' }

function defaultShell(): CliShell {
  return typeof navigator !== 'undefined' && navigator.platform.toLowerCase().includes('win') ? 'powershell' : 'bash'
}

const inputClass =
  'w-full rounded border border-[var(--color-border)] bg-[var(--color-background)] px-2 py-1 text-xs text-[var(--color-foreground)] outline-none focus:border-[var(--color-accent)]'

function MessageLine({ m }: { m: CliMessage }) {
  const Icon = m.severity === 'error' ? XCircle : m.severity === 'warning' ? AlertTriangle : CheckCircle2
  const color = m.severity === 'error' ? 'text-red-400' : m.severity === 'warning' ? 'text-amber-400' : 'text-emerald-400'
  return (
    <p className={cn('flex items-start gap-1.5 text-[11px] leading-snug', color)}>
      <Icon size={12} className="mt-[1px] shrink-0" aria-hidden />
      <span>{m.text}</span>
    </p>
  )
}

function filesFor(role: CliInput['role'], ctx: CliContext | null): CliFile[] {
  if (!ctx) return []
  if (role === 'scenario') return [...ctx.scenarioFolders, ...ctx.scenarios]
  if (role === 'environment') return ctx.environments
  if (role === 'spec') return ctx.specs
  return []
}

type FieldProps = {
  input: CliInput
  value: string
  files: CliFile[]
  messages: CliMessage[]
  suggestion?: CliSuggestion
  onChange: (v: string) => void
}

function Field({ input, value, files, messages, suggestion, onChange }: FieldProps) {
  const id = `cli-${input.name}`
  const isPath = input.kind === 'file' || input.kind === 'fileOrURL' || input.kind === 'dir' || input.kind === 'path'

  async function browse(folder: boolean) {
    const picked = folder ? await pickCliFolder(input.label) : await pickCliFile(input.label)
    if (picked) onChange(picked)
  }

  let control: React.ReactNode
  if (input.kind === 'bool') {
    control = (
      <label className="flex items-center gap-2 text-xs cursor-pointer">
        <input
          id={id}
          type="checkbox"
          checked={value === 'true'}
          onChange={e => onChange(e.target.checked ? 'true' : '')}
        />
        <span>{input.label}</span>
      </label>
    )
  } else if (input.kind === 'enum') {
    control = (
      <select id={id} value={value || input.default} onChange={e => onChange(e.target.value)} className={inputClass}>
        {input.enum.map(v => <option key={v} value={v}>{v}</option>)}
      </select>
    )
  } else if (isPath) {
    control = (
      <div className="flex gap-1">
        <input
          id={id}
          value={value}
          onChange={e => onChange(e.target.value)}
          placeholder={input.example}
          spellCheck={false}
          className={cn(inputClass, 'flex-1 min-w-0 font-mono')}
        />
        {files.length > 0 && (
          <select
            aria-label={`Choose ${input.label} from the workspace`}
            value=""
            onChange={e => { if (e.target.value) onChange(e.target.value) }}
            className={cn(inputClass, 'w-28 shrink-0')}
          >
            <option value="">Workspace…</option>
            {files.map(f => <option key={f.path} value={f.path}>{f.label}</option>)}
          </select>
        )}
        {input.kind === 'path' ? (
          <>
            <button
              type="button"
              onClick={() => browse(false)}
              title="Browse for a file"
              aria-label={`Browse for ${input.label} (file)`}
              className="shrink-0 px-2 rounded border border-[var(--color-border)] text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-muted)]"
            >
              <FileText size={13} />
            </button>
            <button
              type="button"
              onClick={() => browse(true)}
              title="Browse for a folder"
              aria-label={`Browse for ${input.label} (folder)`}
              className="shrink-0 px-2 rounded border border-[var(--color-border)] text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-muted)]"
            >
              <FolderOpen size={13} />
            </button>
          </>
        ) : (
          <button
            type="button"
            onClick={() => browse(input.kind === 'dir')}
            title={`Browse for ${input.label}`}
            aria-label={`Browse for ${input.label}`}
            className="shrink-0 px-2 rounded border border-[var(--color-border)] text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-muted)]"
          >
            <FolderOpen size={13} />
          </button>
        )}
      </div>
    )
  } else {
    control = (
      <input
        id={id}
        value={value}
        inputMode={input.kind === 'int' ? 'numeric' : undefined}
        onChange={e => onChange(e.target.value)}
        placeholder={input.example || (input.default ? `default ${input.default}` : '')}
        spellCheck={false}
        className={cn(inputClass, 'font-mono')}
      />
    )
  }

  return (
    <div className="flex flex-col gap-1">
      {input.kind !== 'bool' && (
        <label htmlFor={id} className="text-xs font-medium text-[var(--color-foreground)]">
          {input.label}
          {input.required && <span className="text-red-400"> *</span>}
        </label>
      )}
      {control}
      <p className="text-[11px] leading-snug text-[var(--color-muted-foreground)]">{input.help}</p>
      {messages.map((m, i) => <MessageLine key={i} m={m} />)}
      {suggestion && (
        <button
          type="button"
          onClick={() => onChange(suggestion.value)}
          title={suggestion.why}
          className="self-start text-[11px] text-[var(--color-accent)] hover:underline font-mono"
        >
          Use {suggestion.value}
        </button>
      )}
    </div>
  )
}

export function CliBuilderTab({ preset }: Props) {
  const [catalog, setCatalog] = useState<CliCommand[] | null>(null)
  const [ctx, setCtx] = useState<CliContext | null>(null)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [command, setCommand] = useState('')
  const [values, setValues] = useState<Record<string, string>>({})
  const [runFrom, setRunFrom] = useState('')
  const [shell, setShell] = useState<CliShell>(defaultShell)
  const [oneline, setOneline] = useState(false)
  const [result, setResult] = useState<CliBuildOutput | null>(null)
  const [copied, setCopied] = useState(false)
  const buildSeq = useRef(0)

  useEffect(() => {
    let cancelled = false
    Promise.all([getCliCatalog(), getCliContext()])
      .then(([cat, context]) => {
        if (cancelled) return
        setCatalog(cat)
        setCtx(context)
        setRunFrom(prev => prev || context.runFrom)
        setCommand(prev => prev || cat[0]?.name || '')
      })
      .catch(e => { if (!cancelled) setLoadError(String(e)) })
    return () => { cancelled = true }
  }, [])

  const [appliedPreset, setAppliedPreset] = useState<typeof preset>(null)
  if (preset !== appliedPreset) {
    setAppliedPreset(preset)
    if (preset) {
      setCommand(preset.command)
      setValues(preset.values ?? {})
    }
  }

  useEffect(() => {
    if (!command) return
    const seq = ++buildSeq.current
    const timer = setTimeout(() => {
      buildCliCommand({ command, values, runFrom })
        .then(out => { if (seq === buildSeq.current) setResult(out) })
        .catch(e => { if (seq === buildSeq.current) setLoadError(String(e)) })
    }, 150)
    return () => clearTimeout(timer)
  }, [command, values, runFrom])

  const current = catalog?.find(c => c.name === command) ?? null
  const rendering = result?.renderings.find(r => r.shell === shell)
  const text = rendering ? (oneline ? rendering.oneline : rendering.multiline) : ''

  function setValue(name: string, v: string) {
    setValues(prev => {
      const next = { ...prev }
      if (v === '') delete next[name]
      else next[name] = v
      return next
    })
  }

  function chooseCommand(name: string) {
    setCommand(name)
    setValues({})
  }

  async function copy() {
    if (!text || !result?.valid) return
    try {
      await navigator.clipboard.writeText(text)
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    } catch {
      setCopied(false)
    }
  }

  async function browseRunFrom() {
    const picked = await pickCliFolder('Folder you will run the command from')
    if (picked) setRunFrom(picked)
  }

  if (loadError) {
    return <div className="p-6 text-sm text-red-400" role="alert">Could not load the CLI builder: {loadError}</div>
  }
  if (!catalog || !current) {
    return <div className="p-6 text-sm text-[var(--color-muted-foreground)]">Loading…</div>
  }

  const knownFields = new Set(current.inputs.map(i => i.name))
  const forField = (name: string) => (result?.messages ?? []).filter(m => m.field === name)
  const general = (result?.messages ?? []).filter(m => !knownFields.has(m.field) && m.severity !== 'info')
  const essential = current.inputs.filter(i => i.group === 'essential')
  const advanced = current.inputs.filter(i => i.group === 'advanced')

  const renderField = (input: CliInput) => (
    <Field
      key={input.name}
      input={input}
      value={values[input.name] ?? ''}
      files={filesFor(input.role, ctx)}
      messages={forField(input.name)}
      suggestion={result?.suggestions.find(s => s.field === input.name)}
      onChange={v => setValue(input.name, v)}
    />
  )

  return (
    <div className="flex flex-col h-full w-full overflow-hidden bg-[var(--color-background)]">
      <div className="shrink-0 px-5 py-3 border-b border-[var(--color-border)] bg-[var(--color-card)]">
        <h1 className="flex items-center gap-2 text-sm font-semibold text-[var(--color-foreground)]">
          <Terminal size={15} className="text-sky-400" aria-hidden />
          CLI Command Builder
        </h1>
        <p className="mt-0.5 text-xs text-[var(--color-muted-foreground)]">
          Build a <code>clrsig</code> command for your terminal or CI pipeline. Every part is checked and explained.
        </p>
      </div>

      <div className="flex flex-1 min-h-0 overflow-hidden">
        {/* Form */}
        <div className="w-[26rem] shrink-0 overflow-y-auto border-r border-[var(--color-border)] p-4 flex flex-col gap-4">
          <div role="radiogroup" aria-label="What do you want to do?" className="flex flex-col gap-1.5">
            <span className="text-xs font-medium text-[var(--color-foreground)]">What do you want to do?</span>
            {catalog.map(c => (
              <button
                key={c.name}
                type="button"
                role="radio"
                aria-checked={c.name === command}
                onClick={() => chooseCommand(c.name)}
                className={cn(
                  'text-left rounded border px-3 py-2 text-xs transition-colors',
                  c.name === command
                    ? 'border-[var(--color-accent)] bg-[var(--color-muted)] text-[var(--color-foreground)]'
                    : 'border-[var(--color-border)] text-[var(--color-muted-foreground)] hover:bg-[var(--color-muted)]',
                )}
              >
                <span className="font-medium">{c.goal}</span>
                {c.name === command && <span className="block mt-0.5 text-[11px] opacity-80">{c.when}</span>}
              </button>
            ))}
          </div>

          {essential.map(renderField)}

          {advanced.length > 0 && (
            <details className="flex flex-col gap-3">
              <summary className="cursor-pointer text-xs font-medium text-[var(--color-muted-foreground)]">
                More options ({advanced.length})
              </summary>
              <div className="mt-3 flex flex-col gap-4">{advanced.map(renderField)}</div>
            </details>
          )}

          <div className="flex flex-col gap-1 pt-3 border-t border-[var(--color-border)]">
            <label htmlFor="cli-runfrom" className="text-xs font-medium text-[var(--color-foreground)]">Run from folder</label>
            <div className="flex gap-1">
              <input
                id="cli-runfrom"
                value={runFrom}
                onChange={e => setRunFrom(e.target.value)}
                spellCheck={false}
                className={cn(inputClass, 'flex-1 min-w-0 font-mono')}
              />
              <button
                type="button"
                onClick={browseRunFrom}
                aria-label="Browse for the run-from folder"
                title="Browse"
                className="shrink-0 px-2 rounded border border-[var(--color-border)] text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-muted)]"
              >
                <FolderOpen size={13} />
              </button>
            </div>
            <p className="text-[11px] leading-snug text-[var(--color-muted-foreground)]">
              The folder you will be in when you run the command. Paths in the command are written relative to it,
              so it works the same on your machine and on a CI runner that checks out the repository. The default is
              your repository root.
            </p>
            {forField('runFrom').map((m, i) => <MessageLine key={i} m={m} />)}
          </div>
        </div>

        {/* Result */}
        <div className="flex-1 min-w-0 overflow-y-auto p-4 flex flex-col gap-4">
          <section aria-label="Command">
            <div className="flex items-center gap-2 mb-2">
              <div role="tablist" aria-label="Shell" className="flex rounded border border-[var(--color-border)] overflow-hidden">
                {(Object.keys(SHELL_LABELS) as CliShell[]).map(s => (
                  <button
                    key={s}
                    type="button"
                    role="tab"
                    aria-selected={shell === s}
                    onClick={() => setShell(s)}
                    className={cn(
                      'px-3 py-1 text-xs',
                      shell === s ? 'bg-[var(--color-accent)] text-white' : 'text-[var(--color-muted-foreground)] hover:bg-[var(--color-muted)]',
                    )}
                  >
                    {SHELL_LABELS[s]}
                  </button>
                ))}
              </div>
              <label className="flex items-center gap-1.5 text-xs text-[var(--color-muted-foreground)] cursor-pointer">
                <input type="checkbox" checked={oneline} onChange={e => setOneline(e.target.checked)} />
                One line
              </label>
              <button
                type="button"
                onClick={copy}
                disabled={!result?.valid}
                className="ml-auto flex items-center gap-1.5 rounded border border-[var(--color-border)] px-3 py-1 text-xs hover:bg-[var(--color-muted)] disabled:opacity-40 disabled:cursor-not-allowed"
              >
                {copied ? <Check size={12} className="text-emerald-400" /> : <Copy size={12} />}
                {copied ? 'Copied' : 'Copy'}
              </button>
            </div>
            <pre
              data-testid="cli-command"
              className={cn(
                'rounded border border-[var(--color-border)] bg-[var(--color-card)] p-3 text-xs font-mono whitespace-pre-wrap break-all',
                result && !result.valid && 'opacity-60',
              )}
            >
              {text || 'Fill in the fields on the left to build the command.'}
            </pre>
            {result && !result.valid && (
              <p className="mt-1 text-[11px] text-red-400">Fix the problems below before using this command.</p>
            )}
          </section>

          {general.length > 0 && (
            <section aria-label="Problems and warnings" className="flex flex-col gap-1">
              {general.map((m, i) => <MessageLine key={i} m={m} />)}
            </section>
          )}

          {result && result.needs.length > 0 && (
            <section aria-label="Environment variables" className="rounded border border-[var(--color-border)] p-3">
              <h2 className="text-xs font-semibold text-[var(--color-foreground)] mb-1">Set these where the command runs</h2>
              <p className="text-[11px] text-[var(--color-muted-foreground)] mb-2">
                Your environment file reads these from the machine&apos;s environment variables. On a CI runner, define them in
                the pipeline&apos;s secrets or variables.
              </p>
              <ul className="flex flex-col gap-0.5 text-xs font-mono">
                {result.needs.map(n => (
                  <li key={n.name}>
                    {n.name}
                    {n.secret && <span className="ml-2 text-[10px] text-amber-400 font-sans">secret</span>}
                    <span className="ml-2 text-[10px] text-[var(--color-muted-foreground)] font-sans">used by {n.from}</span>
                  </li>
                ))}
              </ul>
            </section>
          )}

          {result && (
            <section aria-label="Explanation" className="rounded border border-[var(--color-border)] p-3 flex flex-col gap-3">
              <div>
                <h2 className="text-xs font-semibold text-[var(--color-foreground)]">What this does</h2>
                <p className="text-xs text-[var(--color-muted-foreground)] mt-0.5">{result.explanation.summary}</p>
              </div>

              <div>
                <h3 className="text-xs font-semibold text-[var(--color-foreground)] mb-1">Part by part</h3>
                <dl className="flex flex-col gap-1.5">
                  {result.explanation.parts.map((p, i) => (
                    <div key={i} className="grid grid-cols-[minmax(8rem,16rem)_1fr] gap-3 text-xs">
                      <dt className="font-mono break-all text-[var(--color-foreground)]">{p.token}</dt>
                      <dd className="text-[var(--color-muted-foreground)]">{p.meaning}</dd>
                    </div>
                  ))}
                </dl>
              </div>

              <div>
                <h3 className="text-xs font-semibold text-[var(--color-foreground)] mb-1">What you get</h3>
                <ul className="list-disc pl-4 text-xs text-[var(--color-muted-foreground)] flex flex-col gap-0.5">
                  {result.explanation.outputs.map((o, i) => <li key={i}>{o}</li>)}
                </ul>
              </div>

              <div>
                <h3 className="text-xs font-semibold text-[var(--color-foreground)] mb-1">Exit codes</h3>
                <table className="w-full text-xs">
                  <tbody>
                    {result.explanation.exitCodes.map(e => (
                      <tr key={e.code} className="align-top">
                        <td className="pr-3 py-0.5 font-mono text-[var(--color-foreground)]">{e.code}</td>
                        <td className="pr-3 py-0.5 text-[var(--color-muted-foreground)]">{e.meaning}</td>
                        <td className="py-0.5 text-[var(--color-muted-foreground)] italic">{e.forCI}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </section>
          )}
        </div>
      </div>
    </div>
  )
}
