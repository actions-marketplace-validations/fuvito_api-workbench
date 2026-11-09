import { useState } from 'react'
import { FolderOpen, X, Plus } from 'lucide-react'
import { cn } from '@/lib/utils'
import { inputCls, selectCls } from '@/lib/styles'
import type { BodyState, BodyType } from '@/types'
import { KeyValueEditor } from '@/components/KeyValueEditor'
import type { KVPair } from '@/types'

const BODY_TYPES: { value: BodyType; label: string }[] = [
  { value: 'none', label: 'None' },
  { value: 'json', label: 'JSON' },
  { value: 'raw', label: 'Raw' },
  { value: 'form-urlencoded', label: 'Form URL Encoded' },
  { value: 'multipart', label: 'Multipart Form' },
]

const PLACEHOLDER: Record<BodyType, string> = {
  none: '',
  json: '{\n  "key": "value"\n}',
  raw: 'Body content…',
  'form-urlencoded': '',
  multipart: '',
}

// ---- Multipart field model ----

type MultipartFieldType = 'text' | 'file'

type MultipartField = {
  id: string
  name: string
  fieldType: MultipartFieldType
  value: string
  enabled: boolean
}

function parseMultipartContent(content: string): MultipartField[] {
  if (!content.trim()) return []
  if (content.trimStart().startsWith('[')) {
    try {
      const parsed = JSON.parse(content) as { n: string; t: string; v: string }[]
      return parsed.map(f => ({
        id: crypto.randomUUID(),
        name: f.n ?? '',
        fieldType: f.t === 'file' ? 'file' : 'text',
        value: f.v ?? '',
        enabled: true,
      }))
    } catch { /* fall through to legacy */ }
  }
  // Legacy url-encoded text-only format
  return content.split('&').flatMap(pair => {
    const eqIdx = pair.indexOf('=')
    try {
      const name = decodeURIComponent(eqIdx === -1 ? pair : pair.slice(0, eqIdx))
      const value = eqIdx === -1 ? '' : decodeURIComponent(pair.slice(eqIdx + 1))
      return name ? [{ id: crypto.randomUUID(), name, fieldType: 'text' as const, value, enabled: true }] : []
    } catch { return [] }
  })
}

function serializeMultipartContent(fields: MultipartField[]): string {
  const active = fields.filter(f => f.enabled && f.name.trim())
  if (active.length === 0) return ''
  return JSON.stringify(active.map(f => ({ n: f.name.trim(), t: f.fieldType, v: f.value })))
}

// ---- Form-urlencoded helpers (unchanged) ----

function parseFormPairs(content: string): KVPair[] {
  if (!content.trim()) return []
  return content.split('&').flatMap(pair => {
    const eqIdx = pair.indexOf('=')
    try {
      const name = decodeURIComponent(eqIdx === -1 ? pair : pair.slice(0, eqIdx))
      const value = eqIdx === -1 ? '' : decodeURIComponent(pair.slice(eqIdx + 1))
      return name ? [{ id: crypto.randomUUID(), name, value, enabled: true }] : []
    } catch { return [] }
  })
}

function serializeFormPairs(pairs: KVPair[]): string {
  return pairs
    .filter(p => p.enabled && p.name.trim())
    .map(p => `${encodeURIComponent(p.name.trim())}=${encodeURIComponent(p.value)}`)
    .join('&')
}

// ---- MultipartEditor ----

type MultipartEditorProps = {
  fields: MultipartField[]
  onChange: (fields: MultipartField[]) => void
  onPickFile: () => Promise<string>
}

function MultipartEditor({ fields, onChange, onPickFile }: MultipartEditorProps) {
  function update(id: string, patch: Partial<MultipartField>) {
    onChange(fields.map(f => f.id === id ? { ...f, ...patch } : f))
  }
  function remove(id: string) {
    onChange(fields.filter(f => f.id !== id))
  }
  function add() {
    onChange([...fields, { id: crypto.randomUUID(), name: '', fieldType: 'text', value: '', enabled: true }])
  }
  async function browse(id: string) {
    const path = await onPickFile()
    if (path) update(id, { value: path })
  }

  return (
    <div className="flex flex-col gap-1">
      <div className="grid gap-1" style={{ gridTemplateColumns: '16px 1fr 70px 1fr 20px' }}>
        <span />
        <span className="text-[9px] font-semibold uppercase tracking-wider text-[var(--color-muted-foreground)] px-1">Name</span>
        <span className="text-[9px] font-semibold uppercase tracking-wider text-[var(--color-muted-foreground)] px-1">Type</span>
        <span className="text-[9px] font-semibold uppercase tracking-wider text-[var(--color-muted-foreground)] px-1">Value</span>
        <span />
      </div>

      {fields.map(f => (
        <div key={f.id} className="grid items-center gap-1" style={{ gridTemplateColumns: '16px 1fr 70px 1fr 20px' }}>
          <input
            type="checkbox"
            checked={f.enabled}
            onChange={e => update(f.id, { enabled: e.target.checked })}
            className="accent-[var(--color-accent)]"
          />
          <input
            value={f.name}
            onChange={e => update(f.id, { name: e.target.value })}
            placeholder="name"
            className={cn(inputCls, 'text-xs px-2 py-1')}
          />
          <select
            value={f.fieldType}
            onChange={e => update(f.id, { fieldType: e.target.value as MultipartFieldType, value: '' })}
            className={cn(selectCls, 'text-xs px-1 py-1')}
          >
            <option value="text">Text</option>
            <option value="file">File</option>
          </select>
          {f.fieldType === 'file' ? (
            <div className="flex items-center gap-1">
              <input
                value={f.value}
                onChange={e => update(f.id, { value: e.target.value })}
                placeholder="file path…"
                className={cn(inputCls, 'text-xs px-2 py-1 flex-1 min-w-0 font-mono')}
              />
              <button
                type="button"
                title="Browse"
                onClick={() => browse(f.id)}
                className="shrink-0 p-1 rounded text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-muted)] transition-colors"
              >
                <FolderOpen size={12} />
              </button>
            </div>
          ) : (
            <input
              value={f.value}
              onChange={e => update(f.id, { value: e.target.value })}
              placeholder="value"
              className={cn(inputCls, 'text-xs px-2 py-1')}
            />
          )}
          <button
            type="button"
            onClick={() => remove(f.id)}
            className="p-0.5 rounded text-[var(--color-muted-foreground)] hover:text-red-400 transition-colors"
          >
            <X size={11} />
          </button>
        </div>
      ))}

      <button
        type="button"
        onClick={add}
        className="flex items-center gap-1 text-[10px] text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors mt-1 w-fit"
      >
        <Plus size={10} /> Add field
      </button>
    </div>
  )
}

// ---- BodyEditor ----

type Props = {
  body: BodyState
  onChange: (b: BodyState) => void
  onPickFile?: () => Promise<string>
}

export function BodyEditor({ body, onChange, onPickFile }: Props) {
  const [formPairs, setFormPairs] = useState<KVPair[]>(() => parseFormPairs(body.content))
  const [multipartFields, setMultipartFields] = useState<MultipartField[]>(() => parseMultipartContent(body.content))

  function switchType(t: BodyType) {
    if (t === 'form-urlencoded') setFormPairs(parseFormPairs(body.content))
    if (t === 'multipart') setMultipartFields(parseMultipartContent(body.content))
    onChange({ type: t, content: t === 'none' ? '' : body.content })
  }

  function formatJSON() {
    try {
      onChange({ ...body, content: JSON.stringify(JSON.parse(body.content), null, 2) })
    } catch { /* leave as-is */ }
  }

  function handleFormPairsChange(pairs: KVPair[]) {
    setFormPairs(pairs)
    onChange({ type: 'form-urlencoded', content: serializeFormPairs(pairs) })
  }

  function handleMultipartChange(fields: MultipartField[]) {
    setMultipartFields(fields)
    onChange({ type: 'multipart', content: serializeMultipartContent(fields) })
  }

  const defaultPickFile = async () => ''

  return (
    <div className="flex flex-col gap-2 flex-1 overflow-hidden">
      <div className="flex items-center gap-2 shrink-0">
        <select
          value={body.type}
          onChange={e => switchType(e.target.value as BodyType)}
          className={cn(selectCls, 'px-2 py-1')}
        >
          {BODY_TYPES.map(t => (
            <option key={t.value} value={t.value}>{t.label}</option>
          ))}
        </select>

        {body.type === 'json' && body.content.trim() && (
          <button
            type="button"
            onClick={formatJSON}
            className="text-xs text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors"
          >
            Format
          </button>
        )}
      </div>

      {body.type === 'multipart' ? (
        <div className="flex-1 overflow-auto">
          <MultipartEditor
            fields={multipartFields}
            onChange={handleMultipartChange}
            onPickFile={onPickFile ?? defaultPickFile}
          />
        </div>
      ) : body.type === 'form-urlencoded' ? (
        <div className="flex-1 overflow-auto">
          <KeyValueEditor
            pairs={formPairs}
            onChange={handleFormPairsChange}
            keyPlaceholder="Key"
            valuePlaceholder="Value"
          />
        </div>
      ) : body.type !== 'none' ? (
        <textarea
          value={body.content}
          onChange={e => onChange({ ...body, content: e.target.value })}
          placeholder={PLACEHOLDER[body.type]}
          spellCheck={false}
          className={cn(inputCls, 'flex-1 resize-none font-mono px-3 py-2 leading-relaxed')}
        />
      ) : (
        <p className="text-xs text-[var(--color-muted-foreground)]">This request has no body.</p>
      )}
    </div>
  )
}
