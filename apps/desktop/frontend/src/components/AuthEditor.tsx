import { cn } from '@/lib/utils'
import type { AuthState, AuthType } from '@/types'

const AUTH_TYPES: { value: AuthType; label: string }[] = [
  { value: 'none', label: 'None' },
  { value: 'bearer', label: 'Bearer Token' },
  { value: 'basic', label: 'Basic Auth' },
  { value: 'apikey', label: 'API Key' },
]

const inputCls = cn(
  'w-full bg-[var(--color-muted)] text-xs font-mono rounded px-3 py-1.5',
  'border border-[var(--color-border)] outline-none',
  'focus:border-[var(--color-accent)]',
  'placeholder:text-[var(--color-muted-foreground)]',
  'text-[var(--color-foreground)]',
)

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-1">
      <span className="text-xs text-[var(--color-muted-foreground)]">{label}</span>
      {children}
    </div>
  )
}

type Props = {
  auth: AuthState
  onChange: (a: AuthState) => void
}

export function AuthEditor({ auth, onChange }: Props) {
  function switchType(t: AuthType) {
    if (t === 'none') onChange({ type: 'none' })
    else if (t === 'bearer') onChange({ type: 'bearer', token: '' })
    else if (t === 'basic') onChange({ type: 'basic', username: '', password: '' })
    else onChange({ type: 'apikey', key: '', value: '', location: 'header' })
  }

  return (
    <div className="flex flex-col gap-4">
      <Field label="Type">
        <select
          value={auth.type}
          onChange={e => switchType(e.target.value as AuthType)}
          className={cn(inputCls, 'cursor-pointer')}
        >
          {AUTH_TYPES.map(t => (
            <option key={t.value} value={t.value}>{t.label}</option>
          ))}
        </select>
      </Field>

      {auth.type === 'bearer' && (
        <Field label="Token">
          <input
            type="text"
            value={auth.token}
            onChange={e => onChange({ ...auth, token: e.target.value })}
            placeholder="eyJhbGci…"
            className={inputCls}
            autoComplete="off"
          />
        </Field>
      )}

      {auth.type === 'basic' && (
        <>
          <Field label="Username">
            <input
              type="text"
              value={auth.username}
              onChange={e => onChange({ ...auth, username: e.target.value })}
              placeholder="username"
              className={inputCls}
              autoComplete="username"
            />
          </Field>
          <Field label="Password">
            <input
              type="password"
              value={auth.password}
              onChange={e => onChange({ ...auth, password: e.target.value })}
              placeholder="••••••••"
              className={inputCls}
              autoComplete="current-password"
            />
          </Field>
        </>
      )}

      {auth.type === 'apikey' && (
        <>
          <Field label="Key">
            <input
              type="text"
              value={auth.key}
              onChange={e => onChange({ ...auth, key: e.target.value })}
              placeholder="X-API-Key"
              className={inputCls}
            />
          </Field>
          <Field label="Value">
            <input
              type="text"
              value={auth.value}
              onChange={e => onChange({ ...auth, value: e.target.value })}
              placeholder="api-key-value"
              className={inputCls}
              autoComplete="off"
            />
          </Field>
          <Field label="Add to">
            <select
              value={auth.location}
              onChange={e => onChange({ ...auth, location: e.target.value as 'header' | 'query' })}
              className={cn(inputCls, 'cursor-pointer')}
            >
              <option value="header">Header</option>
              <option value="query">Query Param</option>
            </select>
          </Field>
        </>
      )}

      {auth.type === 'none' && (
        <p className="text-xs text-[var(--color-muted-foreground)]">
          No authentication. Select a type to configure credentials.
        </p>
      )}
    </div>
  )
}
