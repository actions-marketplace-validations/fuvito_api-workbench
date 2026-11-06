import { useState } from 'react'
import { cn } from '@/lib/utils'
import { inputCls, selectCls } from '@/lib/styles'
import type { BodyState, BodyType, KVPair } from '@/types'
import { KeyValueEditor } from '@/components/KeyValueEditor'

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

const IS_FORM_TYPE = (t: BodyType) => t === 'form-urlencoded' || t === 'multipart'

function parseFormPairs(content: string): KVPair[] {
  if (!content.trim()) return []
  return content.split('&').flatMap(pair => {
    const eqIdx = pair.indexOf('=')
    try {
      const name = decodeURIComponent(eqIdx === -1 ? pair : pair.slice(0, eqIdx))
      const value = eqIdx === -1 ? '' : decodeURIComponent(pair.slice(eqIdx + 1))
      return name ? [{ id: crypto.randomUUID(), name, value, enabled: true }] : []
    } catch {
      return []
    }
  })
}

function serializeFormPairs(pairs: KVPair[]): string {
  return pairs
    .filter(p => p.enabled && p.name.trim())
    .map(p => `${encodeURIComponent(p.name.trim())}=${encodeURIComponent(p.value)}`)
    .join('&')
}

type Props = {
  body: BodyState
  onChange: (b: BodyState) => void
}

export function BodyEditor({ body, onChange }: Props) {
  const [formPairs, setFormPairs] = useState<KVPair[]>(() => parseFormPairs(body.content))

  function switchType(t: BodyType) {
    if (IS_FORM_TYPE(t)) {
      setFormPairs(parseFormPairs(body.content))
    }
    onChange({ type: t, content: t === 'none' ? '' : body.content })
  }

  function formatJSON() {
    try {
      onChange({ ...body, content: JSON.stringify(JSON.parse(body.content), null, 2) })
    } catch {
      // leave as-is if not valid JSON
    }
  }

  function handleFormPairsChange(pairs: KVPair[]) {
    setFormPairs(pairs)
    onChange({ type: IS_FORM_TYPE(body.type) ? body.type : 'form-urlencoded', content: serializeFormPairs(pairs) })
  }

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

      {IS_FORM_TYPE(body.type) ? (
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
