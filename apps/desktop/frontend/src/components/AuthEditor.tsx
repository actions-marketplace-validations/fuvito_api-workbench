import { useState } from 'react'
import { Eye, EyeOff } from 'lucide-react'
import { cn } from '@/lib/utils'
import { inputCls as baseInputCls } from '@/lib/styles'
import type { AuthState, AuthType } from '@/types'

const AUTH_TYPES: { value: AuthType; label: string }[] = [
  { value: 'none', label: 'None' },
  { value: 'bearer', label: 'Bearer Token' },
  { value: 'basic', label: 'Basic Auth' },
  { value: 'apikey', label: 'API Key' },
]

const inputCls = cn(baseInputCls, 'w-full font-mono px-3 py-1.5')

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

function SecretInput({ value, onChange, placeholder, autoComplete }: {
  value: string
  onChange: (v: string) => void
  placeholder?: string
  autoComplete?: string
}) {
  const [show, setShow] = useState(false)
  return (
    <div className="relative flex items-center">
      <input
        type={show ? 'text' : 'password'}
        value={value}
        onChange={e => onChange(e.target.value)}
        placeholder={placeholder}
        className={cn(inputCls, 'pr-8')}
        autoComplete={autoComplete ?? 'off'}
      />
      <button
        type="button"
        onClick={() => setShow(s => !s)}
        title={show ? 'Hide' : 'Show'}
        className="absolute right-2 text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors"
      >
        {show ? <EyeOff size={12} /> : <Eye size={12} />}
      </button>
    </div>
  )
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
          <SecretInput
            value={auth.token}
            onChange={v => onChange({ ...auth, token: v })}
            placeholder="eyJhbGci…"
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
            <SecretInput
              value={auth.value}
              onChange={v => onChange({ ...auth, value: v })}
              placeholder="api-key-value"
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
