import { useState } from 'react'
import { X } from 'lucide-react'
import { cn } from '@/lib/utils'
import { inputCls as baseInputCls } from '@/lib/styles'
import type { AppSettings } from '@/lib/bridge'

export type { AppSettings }

type SettingsPanelProps = {
  settings: AppSettings
  onSave: (s: AppSettings) => void
  onReset: () => void
  onClose: () => void
  onPreview?: (s: AppSettings) => void
}

const DEFAULTS: AppSettings = {
  theme: 'dark', fontSize: 13, defaultTimeoutMs: 30000, proxyUrl: '', tlsSkipVerify: false,
  telemetryEnabled: false, demoServerPort: 0, conformanceServerPort: 0,
}

export function SettingsPanel({ settings, onSave, onReset, onClose, onPreview }: SettingsPanelProps) {
  const [local, setLocal] = useState<AppSettings>({ ...settings })
  const [proxyError, setProxyError] = useState('')
  const [portError, setPortError] = useState('')

  function isChanged<K extends keyof AppSettings>(key: K): boolean {
    return (local[key] ?? DEFAULTS[key] ?? '') !== (DEFAULTS[key] ?? '') // an unset optional setting is its default
  }

  function set<K extends keyof AppSettings>(key: K, value: AppSettings[K]) {
    const updated = { ...local, [key]: value }
    setLocal(updated)
    if (key === 'proxyUrl') setProxyError('')
    if (key === 'demoServerPort' || key === 'conformanceServerPort') setPortError('')
    onPreview?.(updated)
  }

  function handleSave() {
    if (local.proxyUrl.trim()) {
      try {
        new URL(local.proxyUrl.trim())
      } catch {
        setProxyError('Enter a valid URL, e.g. http://proxy.example.com:8080')
        return
      }
    }
    const problem = portProblem(local.demoServerPort ?? 0, local.conformanceServerPort ?? 0)
    if (problem) {
      setPortError(problem)
      return
    }
    onSave(local)
    onClose()
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
      <div className={cn(
        'w-[460px] rounded-lg border border-[var(--color-border)] bg-[var(--color-card)]',
        'shadow-2xl flex flex-col',
      )}>
        {/* Header */}
        <div className="flex items-center justify-between px-4 py-3 border-b border-[var(--color-border)]">
          <span className="text-sm font-semibold text-[var(--color-foreground)]">Settings</span>
          <button type="button" onClick={onClose} aria-label="Close settings" className="text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors">
            <X size={15} />
          </button>
        </div>

        {/* Body */}
        <div className="flex flex-col gap-5 p-5 overflow-auto">
          {/* Appearance */}
          <section>
            <h3 className="text-[11px] font-semibold uppercase tracking-widest text-[var(--color-muted-foreground)] mb-3">Appearance</h3>
            <div className="flex flex-col gap-3">

              <Row label="Theme" changed={isChanged('theme')}>
                <select
                  value={local.theme}
                  onChange={e => set('theme', e.target.value as 'dark' | 'light')}
                  className={inputCls}
                >
                  <option value="dark">Dark</option>
                  <option value="light">Light</option>
                </select>
              </Row>

              <Row label={`Font size — ${local.fontSize}px`} changed={isChanged('fontSize')}>
                <input
                  type="range"
                  min={10} max={24} step={1}
                  value={local.fontSize}
                  onChange={e => set('fontSize', Number(e.target.value))}
                  className="w-full accent-[var(--color-accent)]"
                />
              </Row>

            </div>
          </section>

          {/* Network */}
          <section>
            <h3 className="text-[11px] font-semibold uppercase tracking-widest text-[var(--color-muted-foreground)] mb-3">Network</h3>
            <div className="flex flex-col gap-3">

              <Row label="Default timeout (ms)" changed={isChanged('defaultTimeoutMs')}>
                <input
                  type="number"
                  min={0}
                  value={local.defaultTimeoutMs}
                  onChange={e => set('defaultTimeoutMs', Number(e.target.value))}
                  className={inputCls}
                  placeholder="30000"
                />
              </Row>

              <Row label="Proxy URL" changed={isChanged('proxyUrl')}>
                <div className="flex flex-col gap-1">
                  <input
                    type="text"
                    value={local.proxyUrl}
                    onChange={e => set('proxyUrl', e.target.value)}
                    placeholder="http://proxy.example.com:8080"
                    className={cn(inputCls, proxyError && 'border-red-500')}
                  />
                  {proxyError && (
                    <span className="text-[10px] text-red-400">{proxyError}</span>
                  )}
                </div>
              </Row>

              <Row label="Skip TLS verification" changed={isChanged('tlsSkipVerify')}>
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={local.tlsSkipVerify}
                    onChange={e => set('tlsSkipVerify', e.target.checked)}
                    className="accent-[var(--color-accent)] focus-visible:ring-2 focus-visible:ring-[var(--color-accent)] focus-visible:ring-offset-1 focus-visible:ring-offset-[var(--color-background)]"
                  />
                  <span className="text-xs text-[var(--color-muted-foreground)]">
                    Allow self-signed / invalid certificates
                  </span>
                </label>
              </Row>

              <Row label="Client certificate file" changed={isChanged('clientCertFile')}>
                <input
                  type="text"
                  value={local.clientCertFile ?? ''}
                  onChange={e => set('clientCertFile', e.target.value)}
                  placeholder="/path/to/client.pem (or ${CLIENT_CERT})"
                  aria-label="Client certificate file"
                  className={inputCls}
                />
              </Row>

              <Row label="Client key file" changed={isChanged('clientKeyFile')}>
                <input
                  type="text"
                  value={local.clientKeyFile ?? ''}
                  onChange={e => set('clientKeyFile', e.target.value)}
                  placeholder="/path/to/client.key (not password-protected)"
                  aria-label="Client key file"
                  className={inputCls}
                />
              </Row>

              <Row label="CA certificate file" changed={isChanged('caFile')}>
                <input
                  type="text"
                  value={local.caFile ?? ''}
                  onChange={e => set('caFile', e.target.value)}
                  placeholder="/path/to/ca.pem (extra CAs to trust)"
                  aria-label="CA certificate file"
                  className={inputCls}
                />
              </Row>

            </div>
          </section>
          {/* Dev servers */}
          <section>
            <h3 className="text-[11px] font-semibold uppercase tracking-widest text-[var(--color-muted-foreground)] mb-3">Dev servers</h3>
            <div className="flex flex-col gap-3">
              <Row label="Demo Server port" changed={isChanged('demoServerPort')}>
                <input
                  type="number"
                  min={0}
                  max={65535}
                  value={local.demoServerPort || ''}
                  placeholder="Automatic"
                  onChange={e => set('demoServerPort', Number.parseInt(e.target.value, 10) || 0)}
                  aria-label="Demo Server port"
                  className={inputCls}
                />
              </Row>
              <Row label="Conformance Server port" changed={isChanged('conformanceServerPort')}>
                <input
                  type="number"
                  min={0}
                  max={65535}
                  value={local.conformanceServerPort || ''}
                  placeholder="Automatic"
                  onChange={e => set('conformanceServerPort', Number.parseInt(e.target.value, 10) || 0)}
                  aria-label="Conformance Server port"
                  className={inputCls}
                />
              </Row>
              <p className="text-[10px] text-[var(--color-muted-foreground)]">
                Leave a port empty to pick a free one automatically (from 8090 and 8120). Set a port (1024–65535) to keep the same address every time; if it is taken, the server will not start and says so.
              </p>
              {portError && <span role="alert" className="text-[10px] text-red-400">{portError}</span>}
            </div>
          </section>
          {/* Privacy */}
          <section>
            <h3 className="text-[11px] font-semibold uppercase tracking-widest text-[var(--color-muted-foreground)] mb-3">Privacy</h3>
            <div className="flex flex-col gap-3">

              <Row label="Usage telemetry" changed={isChanged('telemetryEnabled')}>
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={local.telemetryEnabled}
                    onChange={e => set('telemetryEnabled', e.target.checked)}
                    className="accent-[var(--color-accent)] focus-visible:ring-2 focus-visible:ring-[var(--color-accent)] focus-visible:ring-offset-1 focus-visible:ring-offset-[var(--color-background)]"
                  />
                  <span className="text-xs text-[var(--color-muted-foreground)]">
                    The clrsig command line sends each run’s command, result, duration and counts, with a random installation ID, version, system and time. No URLs, headers or bodies
                  </span>
                </label>
              </Row>

            </div>
          </section>

        </div>

        {/* Footer */}
        <div className="flex items-center justify-between px-4 py-3 border-t border-[var(--color-border)]">
          <button
            type="button"
            onClick={onReset}
            className={cn(
              'text-xs rounded px-3 py-1.5',
              'text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors',
            )}
          >
            Reset to defaults
          </button>
          <div className="flex items-center gap-2">
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
              className={cn(
                'text-xs rounded px-4 py-1.5 font-semibold',
                'bg-[var(--color-accent)] text-white hover:opacity-90 transition-opacity',
              )}
            >
              Save
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

const inputCls = cn(baseInputCls, 'w-full px-2 py-1')

// A configured Dev Server port is 0 (automatic) or 1024-65535, and the two servers cannot share one.
export function portProblem(demo: number, conformance: number): string {
  for (const [name, port] of [['Demo Server', demo], ['Conformance Server', conformance]] as const) {
    if (port !== 0 && (port < 1024 || port > 65535)) {
      return `${name} port must be 0 (automatic) or between 1024 and 65535.`
    }
  }
  if (demo !== 0 && demo === conformance) return 'The two servers need different ports.'
  return ''
}

function Row({ label, children, changed }: { label: string; children: React.ReactNode; changed?: boolean }) {
  return (
    <div className="flex items-center gap-3">
      <div className="w-40 shrink-0 flex items-center gap-1.5">
        <span className={cn('text-xs', changed ? 'text-[var(--color-foreground)]' : 'text-[var(--color-muted-foreground)]')}>
          {label}
        </span>
        {changed && <span className="w-1.5 h-1.5 rounded-full bg-[var(--color-accent)] shrink-0" title="Modified from default" />}
      </div>
      <div className="flex-1">{children}</div>
    </div>
  )
}
