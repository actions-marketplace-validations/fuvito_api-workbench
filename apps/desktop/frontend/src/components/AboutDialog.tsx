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
      'CLI — clrsig run for scenario execution in CI/CD pipelines',
      'Request and execution history via SQLite',
      'Settings: theme, font size, timeout, proxy URL, TLS skip-verify',
      'Cross-platform release builds (Windows, macOS, Linux) via GitHub Actions',
    ],
  },
]

const THIRD_PARTY: Array<{ name: string; license: string; url: string }> = [
  { name: 'Wails v2',                license: 'MIT',           url: 'github.com/wailsapp/wails' },
  { name: 'React',                   license: 'MIT',           url: 'github.com/facebook/react' },
  { name: 'Vite',                    license: 'MIT',           url: 'github.com/vitejs/vite' },
  { name: 'Tailwind CSS',            license: 'MIT',           url: 'github.com/tailwindlabs/tailwindcss' },
  { name: 'lucide-react',            license: 'ISC',           url: 'github.com/lucide-icons/lucide' },
  { name: 'react-markdown',          license: 'MIT',           url: 'github.com/remarkjs/react-markdown' },
  { name: 'CodeMirror / @uiw/react-codemirror', license: 'MIT', url: 'github.com/uiwjs/react-codemirror' },
  { name: 'kin-openapi',             license: 'Apache 2.0',    url: 'github.com/getkin/kin-openapi' },
  { name: 'PaesslerAG/jsonpath',     license: 'BSD 3-Clause',  url: 'github.com/PaesslerAG/jsonpath' },
  { name: 'modernc.org/sqlite',      license: 'BSD / MIT',     url: 'gitlab.com/cznic/sqlite' },
  { name: 'gopkg.in/yaml.v3',        license: 'MIT',           url: 'github.com/go-yaml/yaml' },
  { name: 'cobra',                   license: 'Apache 2.0',    url: 'github.com/spf13/cobra' },
]

const MIT_LICENSE = `Copyright 2026 Fuat Yazar

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.`

export function AboutDialog({ onClose }: { onClose: () => void }) {
  const [tab, setTab] = useState<'about' | 'release-notes' | 'legal'>('about')

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
            alt="Clear Signal icon"
            className="w-24 h-24 rounded-2xl shadow-lg select-none"
            draggable={false}
          />
          <div className="flex flex-col items-center gap-1">
            <h2 className="text-xl font-bold text-[var(--color-foreground)] tracking-tight">Clear Signal</h2>
            <p className="text-xs text-[var(--color-muted-foreground)]">Version 0.1.0 — Public MVP</p>
          </div>
          <p className="text-xs text-[var(--color-muted-foreground)] leading-relaxed text-center max-w-xs">
            Your local-first API workbench. Test requests, validate OpenAPI contracts,
            and run scenario pipelines in CI/CD — all from plain files your team can version-control.
          </p>
        </div>

        {/* Tabs */}
        <div className="flex border-b border-[var(--color-border)] px-8 shrink-0">
          {(['about', 'release-notes', 'legal'] as const).map(t => (
            <button
              key={t}
              type="button"
              onClick={() => setTab(t)}
              className={cn(
                'px-3 py-2 text-xs border-b-2 transition-colors',
                tab === t
                  ? 'border-[var(--color-accent)] text-[var(--color-foreground)]'
                  : 'border-transparent text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)]',
              )}
            >
              {t === 'release-notes' ? 'Release Notes' : t === 'legal' ? 'Legal' : 'About'}
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

          {tab === 'legal' && (
            <div className="px-8 py-5 flex flex-col gap-6">

              {/* Privacy */}
              <div>
                <h3 className="text-xs font-semibold text-[var(--color-foreground)] mb-2 uppercase tracking-wide">Privacy</h3>
                <div className="flex items-center gap-2 mb-2">
                  <span className="text-[10px] px-2 py-0.5 rounded-full bg-emerald-500/15 text-emerald-500 font-semibold">
                    No data leaves your device
                  </span>
                </div>
                <p className="text-xs text-[var(--color-muted-foreground)] leading-relaxed">
                  Clear Signal is local-first. There are no accounts, no telemetry, and no cloud sync.
                  Request history is stored only in a local SQLite database
                  (<code className="font-mono text-[10px]">~/.clrsig/history.db</code>) and is never transmitted anywhere.
                  Credentials are never written to history or reports.
                </p>
              </div>

              {/* License */}
              <div>
                <h3 className="text-xs font-semibold text-[var(--color-foreground)] mb-2 uppercase tracking-wide">License</h3>
                <pre className="text-[10px] font-mono text-[var(--color-muted-foreground)] leading-relaxed whitespace-pre-wrap bg-[var(--color-muted)] rounded p-3">
                  {MIT_LICENSE}
                </pre>
              </div>

              {/* Third-party */}
              <div>
                <h3 className="text-xs font-semibold text-[var(--color-foreground)] mb-2 uppercase tracking-wide">Third-party Notices</h3>
                <div className="flex flex-col divide-y divide-[var(--color-border)]">
                  {THIRD_PARTY.map(dep => (
                    <div key={dep.name} className="flex items-baseline gap-3 py-1.5 text-xs">
                      <span className="text-[var(--color-foreground)] shrink-0 w-52">{dep.name}</span>
                      <span className="text-[var(--color-muted-foreground)] font-mono text-[10px] shrink-0 w-24">{dep.license}</span>
                      <span className="text-[var(--color-muted-foreground)] text-[10px] truncate">{dep.url}</span>
                    </div>
                  ))}
                </div>
                <p className="mt-3 text-[10px] text-[var(--color-muted-foreground)]">
                  Apache 2.0 licensed dependencies are used unmodified. Full license texts are available at the URLs above.
                </p>
              </div>

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
