import { useState } from 'react'
import { cn } from '@/lib/utils'

const RELEASE_NOTES: Array<{ version: string; date: string; items: string[] }> = [
  {
    version: '0.7.2',
    date: '2026-11-22',
    items: [
      'Fix: a spec that lists its servers (servers: [{url: https://host/v1}]) is now matched by path, so responses are compared with it. Before, such a spec matched nothing, every response passed unchecked and coverage showed 0%',
      'Fix: a run report says when responses were not compared with the spec and why',
      'Generated requests now carry the credentials of the security scheme an operation names (basic and API key as well as bearer), required cookie parameters, array query parameters written by style and explode, schema-level and OpenAPI 3.1 examples as bodies, and form and multipart fields',
      'Runs that passed only because nothing was compared can now fail. Record the current state with clrsig conform --write-baseline and adopt the rest gradually',
    ],
  },
  {
    version: '0.7.1',
    date: '2026-11-21',
    items: [
      'Fix: the GitHub Action in 0.7.0 could not run (action.yml contained a stray character that made it invalid YAML). Use fuvito/api-workbench@v0.7.1; the desktop app, the CLI binaries, the Docker image and the Scoop and Homebrew packages of 0.7.0 were not affected',
      'A test now loads action.yml, checks that it is valid YAML with the inputs its scripts read, and runs bash -n on every script',
    ],
  },
  {
    version: '0.7.0',
    date: '2026-11-21',
    items: [
      'Every rail icon works the same way: Workspace, Requests, Scenarios, Contracts, Check Conformance and CLI Command Builder each open a sidebar, and clicking an icon brings back the tab you last had open under it (for this session). Clicking the lit icon collapses the sidebar; clicking a tab lights its icon and never opens or closes the sidebar, so a collapse you chose is kept',
      'Workspace sidebar: an environment selector (the same choice as the status bar) with where it came from, the Needs attention list (click an item to see its card on the Workspace page), and counts of requests, scenarios and specs. It reads the same snapshot as the Workspace page, so the two always agree',
      'Check Conformance sidebar: the workspace\'s specs and environments (click one to fill the matching field on the page), the choice marked, and a Last result card. CLI Command Builder sidebar: the commands with their goals; click one to select it on the page',
      'A scenario result explains each spec violation the way the Check Conformance page does: what the spec expected, what came back, where, in the spec, why it fails and how to fix it (the validator\'s own text is under Technical detail)',
      'Behaviour change: the Check Conformance page reads spec and environment files only from the open workspace (symbolic links are resolved), and says to copy a file into the workspace when it is not there. A spec that scenarios reach through a link out of the workspace is shown as outside',
      'Dev servers: the Demo and Conformance Servers no longer send CORS headers and refuse requests that are not addressed to localhost or that come from another website; a second running copy of the app no longer loses its temporary workspace when you start a dev server',
      'GitHub Action: inputs for the newer conform and diff flags (fail-on, against, read-only, negative, explain, baseline, write-baseline, tags, paths, request-timeout, var, report); inputs are passed as environment variables so a value cannot change what runs',
      'One name per feature (Check Conformance, CLI Command Builder, Contracts) and tabs carry the rail\'s icons instead of three-letter codes; the request URL field keeps a usable width, with the timeout and redirect options wrapping under it',
      'CLI Command Builder opens without a red error, unselected choices look clickable and the explanation stacks under its term; the Check Conformance Base URL hint waits for a spec; Settings shows Automatic for the dev server ports; a dismissed Workspace reminder stays dismissed',
    ],
  },
  {
    version: '0.6.0',
    date: '2026-11-21',
    items: [
      'Workspace page: the first item in the rail, opened when you open a workspace. It shows the active environment and where that choice came from, the specs your scenarios use, the last results, and a Needs attention list (no environment, unset ${NAME}, missing spec or request file, broken scenario), each with a fix',
      'Check Conformance page in the desktop app: safe-by-default (reads only) spec conformance runs with four totals, a failure-first report, negative probes, export and an optional Contracts coverage record; Help → Dev Servers adds a Conformance Server with deliberate defects',
      'More accurate response checks (behaviour change): the status, Content-Type and declared headers are now compared, not only the body; CI that was silently green can turn red. Adopt gradually with clrsig conform --write-baseline / --baseline',
      'Every failure says what, where, why and how to fix it (--explain, HTML, Markdown, JUnit, JSON); operations with a missing value are reported as not sent instead of being sent with a placeholder',
      'Generated requests are valid by default: typed required-only bodies, and required query and header parameters become {{placeholders}}; clrsig conform --read-only',
      'Dynamic values and scenario variables ({{$guid}}, {{$randomInt}}, {{$calc}}, --seed), the CLI Command Builder, running scenarios from folders, --report, --var, markdown output, Postman/Insomnia import',
      'Secret values are masked in every report; ${NAME} in environment files is now expanded from the process environment',
      'What We Check catalog in the Help panel: every check, its ID, and the known gaps',
      'Fix: a valid XML (or other non-JSON) response is no longer reported as a violation; it is listed as not checked',
      'Fix: clrsig impact now matches steps whose URL path contains a {{placeholder}}',
      'The conformance text report shows the spec path, and a dead server gives one clear line instead of one error per operation; the page shows the run verdict and says when no operation was reached',
      'The rail follows your tabs: switching tabs lights the matching rail icon and shows or hides the sidebar; opening a workspace shows the Workspace tab first, then the first request',
      'Demo Server workspace now includes its spec and passes a read-only conformance check out of the box; its scenarios and the conformance run share one Contracts card',
      'Contracts: each card shows where its spec comes from, and a spec named by a relative path can be opened and exported',
      'Fix: Run All no longer runs folders as scenarios, and the Scenarios panel no longer shows a scrollbar when its few items fit',
      'Fix: the Docker image reports its version; DeleteFolder and workspace path checks are stricter',
      'Known: builds are not code-signed yet, so Windows SmartScreen and macOS Gatekeeper warn on first run',
    ],
  },
  {
    version: '0.5.0',
    date: '2026-11-17',
    items: [
      'HTML reports: --format html for run, conform, diff and bundle run — one self-contained file you can open in a browser or keep as a CI artifact',
      'Fix: --spec on run, bundle run and bundle validate now resolves from the current directory (it was resolved against the scenario folder, silently skipping schema validation and coverage)',
      'Docs aligned with the code: all CLI commands documented, output formats table, corrected retry/timeout notes',
    ],
  },
  {
    version: '0.4.0',
    date: '2026-11-17',
    items: [
      'Bundle import/export — a whole workspace (environments, requests, scenarios) as a single YAML/JSON file',
      'clrsig bundle run|validate|import|export — run scenarios straight from a bundle file',
      'Help panel: Example Scenario and AI Prompt tabs with Copy and Download',
      'clrsig impact command and MCP server over stdio',
      'clrsig conform --negative — negative-path conformance checks',
      'Coverage quality indicators in CLI and desktop UI; Contracts tab export to CSV, Markdown and PDF',
      'Fix: blank page when a scenario run exercised no spec operations',
      'Broad Go and frontend test coverage; several bugs found and fixed along the way',
    ],
  },
  {
    version: '0.3.0',
    date: '2026-11-13',
    items: [
      'Ctrl+P quick open — fuzzy-search all request files in the workspace',
      'Env-var autocomplete — typing {{ in URL bar or value fields shows available variables',
      'Collapsible response panel — collapse to a narrow strip to give the editor more room',
      'Tab overflow — gradient fade indicators and active-tab auto-scroll when tabs exceed bar width',
      'Empty states for Explorer, Scenarios, and History panels',
      'Auto-open first request when a workspace is opened',
      'Switching workspaces closes the previous workspace and all its tabs',
    ],
  },
  {
    version: '0.2.0',
    date: '2026-11-09',
    items: [
      'Inno Setup Windows installer with Start Menu and optional Desktop shortcut',
      'clrsig conform: cross-run coverage aggregation via history.db — cumulative view shows operations never covered across all runs',
      'clrsig conform: request validation (violations reported separately from response violations)',
      'clrsig diff --format junit: breaking changes become JUnit <failure> elements for PR annotation',
      'Homebrew tap (brew tap fuvito/api-workbench && brew install clrsig)',
      'Scoop manifest (scoop bucket add + scoop install clrsig)',
      'Docker image published to ghcr.io on every release',
      'GitHub Actions Marketplace: uses: fuvito/api-workbench@v1',
      'clrsig --version via cobra Version field set by build ldflags',
      'os.Exit refactored out of cobra handlers (exitError sentinel)',
      'Demo GIF regenerated automatically by CI on every tape/CLI/core change',
    ],
  },
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
            <p className="text-xs text-[var(--color-muted-foreground)]">Version 0.7.2</p>
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
              {RELEASE_NOTES.map((rel, idx) => (
                <div key={rel.version}>
                  <div className="flex items-baseline gap-3 mb-3">
                    <span className="text-sm font-bold text-[var(--color-foreground)]">v{rel.version}</span>
                    <span className="text-xs text-[var(--color-muted-foreground)]">{rel.date}</span>
                    {idx === 0 && (
                      <span className="text-[10px] px-1.5 py-0.5 rounded bg-[var(--color-accent)]/15 text-[var(--color-accent)] font-semibold uppercase tracking-wide">
                        Latest
                      </span>
                    )}
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
