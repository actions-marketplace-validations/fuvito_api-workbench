import { useState } from 'react'
import { cn } from '@/lib/utils'

const RELEASE_NOTES: Array<{ version: string; date: string; items: string[] }> = [
  {
    version: '0.1.0',
    date: '2026-11-04',
    items: [
      'Multi-tab request editor with drag-to-resize panels',
      'Workspace management — collections as YAML/JSON files',
      'Inline rename and delete for requests and folders',
      'Environment management with variable interpolation ({{var}})',
      'Full HTTP execution with DNS/TCP/TLS/TTFB/transfer timing breakdown',
      'OpenAPI 3.0/3.1 spec import — generates request collections automatically',
      'Scenario engine: multi-step execution, variable extraction, assertions',
      'CLI — apidesk run for scenario execution in CI/CD pipelines',
      'Request and execution history via SQLite',
      'Settings: theme, font size, timeout, proxy URL, TLS skip-verify',
      'Cross-platform release builds (Windows, macOS, Linux) via GitHub Actions',
    ],
  },
]

export function AboutDialog({ onClose }: { onClose: () => void }) {
  const [tab, setTab] = useState<'about' | 'release-notes'>('about')

  return (
    <div
      className="fixed inset-0 z-[200] flex items-center justify-center bg-black/60"
      onClick={onClose}
    >
      <div
        className="bg-[var(--color-card)] border border-[var(--color-border)] rounded-xl shadow-2xl flex flex-col w-[520px] max-h-[90vh]"
        onClick={e => e.stopPropagation()}
      >
        {/* Header band with icon */}
        <div className="flex flex-col items-center gap-4 px-8 pt-8 pb-6 border-b border-[var(--color-border)] shrink-0">
          <img
            src="/appicon.png"
            alt="ApiDesk icon"
            className="w-24 h-24 rounded-2xl shadow-lg select-none"
            draggable={false}
          />
          <div className="flex flex-col items-center gap-1">
            <h2 className="text-xl font-bold text-[var(--color-foreground)] tracking-tight">ApiDesk</h2>
            <p className="text-xs text-[var(--color-muted-foreground)]">Version 0.1.0 — Public MVP</p>
          </div>
          <p className="text-xs text-[var(--color-muted-foreground)] leading-relaxed text-center max-w-xs">
            Your local-first API workbench. Test requests, validate OpenAPI contracts,
            and run scenario pipelines in CI/CD — all from plain files your team can version-control.
          </p>
        </div>

        {/* Tabs */}
        <div className="flex border-b border-[var(--color-border)] px-8 shrink-0">
          {(['about', 'release-notes'] as const).map(t => (
            <button
              key={t}
              type="button"
              onClick={() => setTab(t)}
              className={cn(
                'px-3 py-2 text-xs border-b-2 transition-colors capitalize',
                tab === t
                  ? 'border-[var(--color-accent)] text-[var(--color-foreground)]'
                  : 'border-transparent text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)]',
              )}
            >
              {t === 'release-notes' ? 'Release Notes' : 'About'}
            </button>
          ))}
        </div>

        {/* Tab content */}
        <div className="flex-1 overflow-y-auto min-h-0">
          {tab === 'about' && (
            <div className="px-8 py-5 flex flex-col gap-2">
              {[
                ['Runtime',   'Go 1.26 + Wails v2'],
                ['Frontend',  'React 19 · TypeScript 7 · Vite 8'],
                ['UI',        'Tailwind CSS v4 · shadcn/ui'],
                ['Storage',   'YAML / JSON · SQLite history'],
                ['License',   'MIT'],
              ].map(([label, value]) => (
                <div key={label} className="flex gap-3 text-xs">
                  <span className="w-20 shrink-0 text-[var(--color-muted-foreground)]">{label}</span>
                  <span className="text-[var(--color-foreground)] font-mono">{value}</span>
                </div>
              ))}
            </div>
          )}

          {tab === 'release-notes' && (
            <div className="px-8 py-5 flex flex-col gap-6">
              {RELEASE_NOTES.map(rel => (
                <div key={rel.version}>
                  <div className="flex items-baseline gap-3 mb-3">
                    <span className="text-sm font-bold text-[var(--color-foreground)]">v{rel.version}</span>
                    <span className="text-xs text-[var(--color-muted-foreground)]">{rel.date}</span>
                    <span className="text-[10px] px-1.5 py-0.5 rounded bg-[var(--color-accent)]/15 text-[var(--color-accent)] font-semibold uppercase tracking-wide">
                      Latest
                    </span>
                  </div>
                  <ul className="flex flex-col gap-1.5">
                    {rel.items.map(item => (
                      <li key={item} className="flex gap-2 text-xs text-[var(--color-muted-foreground)]">
                        <span className="shrink-0 mt-px text-[var(--color-accent)]">•</span>
                        {item}
                      </li>
                    ))}
                  </ul>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="flex justify-between items-center px-8 py-4 border-t border-[var(--color-border)] shrink-0">
          <span className="text-[10px] text-[var(--color-muted-foreground)]">
            © 2026 Fuat Yazar
          </span>
          <button
            type="button"
            onClick={onClose}
            className="px-4 py-1.5 text-xs rounded bg-[var(--color-muted)] text-[var(--color-foreground)] hover:bg-[var(--color-border)] transition-colors"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  )
}
