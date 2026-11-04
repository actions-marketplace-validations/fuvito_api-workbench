import { useState, useEffect, useCallback, useRef } from 'react'
import { FolderOpen, Save, FilePlus, Settings, Pencil } from 'lucide-react'
import { MenuBar } from '@/components/MenuBar'
import type { MenuDef } from '@/components/MenuBar'
import { TabBar } from '@/components/TabBar'
import {
  executeRequest, getHistory, openWorkspace, newWorkspace, getWorkspacePath, listRequests,
  loadRequest, saveRequest, openRequest, deleteRequest, setHasDirtyTabs,
  createFolder, deleteFolder, renameFolder, renameRequest,
  listEnvironments, selectEnvironment, getActiveEnvironment,
  importOpenAPISpec, getSettings, saveSettings,
  type ResponseData, type AuthPayload, type HistoryEntry, type FileNode,
  type EnvInfo, type RequestFileIO, type AppSettings,
} from '@/lib/bridge'
import { SettingsPanel } from '@/components/SettingsPanel'
import { EnvironmentEditor } from '@/components/EnvironmentEditor'
import { cn } from '@/lib/utils'
import { KeyValueEditor } from '@/components/KeyValueEditor'
import { BodyEditor } from '@/components/BodyEditor'
import { AuthEditor } from '@/components/AuthEditor'
import { WorkspacePanel } from '@/components/WorkspacePanel'
import type { KVPair, BodyState, AuthState } from '@/types'

// ---- Dirty-state helpers ----

function computeStateKey(
  method: string,
  url: string,
  queryParams: KVPair[],
  headers: KVPair[],
  body: BodyState,
  auth: AuthState,
  name: string | null,
): string {
  return JSON.stringify({
    method,
    url: url.trim(),
    name,
    params: queryParams
      .filter(p => p.name.trim())
      .map(p => ({ name: p.name.trim(), value: p.value, enabled: p.enabled })),
    headers: headers
      .filter(h => h.name.trim())
      .map(h => ({ name: h.name.trim(), value: h.value, enabled: h.enabled })),
    body,
    auth,
  })
}

const HTTP_METHODS = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'HEAD', 'OPTIONS'] as const
type HttpMethod = typeof HTTP_METHODS[number]

const METHOD_COLOR: Record<HttpMethod, string> = {
  GET:     'text-emerald-400',
  POST:    'text-blue-400',
  PUT:     'text-amber-400',
  PATCH:   'text-orange-400',
  DELETE:  'text-red-400',
  HEAD:    'text-purple-400',
  OPTIONS: 'text-cyan-400',
}

const METHOD_BADGE: Record<string, string> = { ...METHOD_COLOR as Record<string, string> }

function statusColor(code: number): string {
  if (code >= 200 && code < 300) return 'text-emerald-400'
  if (code >= 300 && code < 400) return 'text-blue-400'
  if (code >= 400 && code < 500) return 'text-amber-400'
  return 'text-red-400'
}

function prettyBody(body: string): string {
  try { return JSON.stringify(JSON.parse(body), null, 2) } catch { return body }
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

function authStateToPayload(auth: AuthState): AuthPayload | undefined {
  if (auth.type === 'bearer') {
    return { type: 'bearer', token: auth.token, username: '', password: '', key: '', value: '', location: '' }
  }
  if (auth.type === 'basic') {
    return { type: 'basic', token: '', username: auth.username, password: auth.password, key: '', value: '', location: '' }
  }
  if (auth.type === 'apikey') {
    return { type: 'apikey', token: '', username: '', password: '', key: auth.key, value: auth.value, location: auth.location }
  }
  return undefined
}

function requestFileIOToState(r: RequestFileIO): {
  method: HttpMethod
  url: string
  headers: KVPair[]
  queryParams: KVPair[]
  body: BodyState
  auth: AuthState
} {
  const toKV = (pairs: Array<{ name: string; value: string }> = []): KVPair[] =>
    pairs.map(p => ({ id: crypto.randomUUID(), name: p.name, value: p.value, enabled: true }))

  let auth: AuthState = { type: 'none' }
  if (r.auth && r.auth.type !== 'none' && r.auth.type !== '') {
    if (r.auth.type === 'bearer') auth = { type: 'bearer', token: r.auth.token }
    else if (r.auth.type === 'basic') auth = { type: 'basic', username: r.auth.username, password: r.auth.password }
    else if (r.auth.type === 'apikey') {
      auth = { type: 'apikey', key: r.auth.key, value: r.auth.value, location: (r.auth.location as 'header' | 'query') || 'header' }
    }
  }

  const method = (HTTP_METHODS as readonly string[]).includes(r.method?.toUpperCase())
    ? r.method.toUpperCase() as HttpMethod
    : 'GET'

  return {
    method,
    url: r.url ?? '',
    headers: toKV(r.headers),
    queryParams: toKV(r.queryParams),
    body: r.body ? { type: r.body.type as BodyState['type'], content: r.body.content } : { type: 'none', content: '' },
    auth,
  }
}

// ---- Sidebar ----

type SidebarTab = 'workspace' | 'history'

type SidebarProps = {
  width: number
  tab: SidebarTab
  onTabChange: (t: SidebarTab) => void
  workspacePath: string
  fileTree: FileNode[]
  history: HistoryEntry[]
  onOpenWorkspace: () => void
  onNewWorkspace: () => void
  onSelectFile: (path: string) => void
  onDeleteFile: (path: string) => void
  onDeleteFolder: (path: string) => void
  onNewRequestInFolder: (folderPath: string) => void
  onCreateFolder: (parentPath: string) => void
  onRenameFile: (path: string, newName: string) => void
  onRenameFolder: (path: string, newName: string) => void
  onSelectHistory: (entry: HistoryEntry) => void
  onImportOpenAPI: () => void
  importing?: boolean
  activeFilePath?: string
  isDirty?: boolean
}

function Sidebar({
  width,
  tab, onTabChange,
  workspacePath, fileTree, history,
  onOpenWorkspace, onNewWorkspace,
  onSelectFile, onDeleteFile, onDeleteFolder, onNewRequestInFolder, onCreateFolder,
  onRenameFile, onRenameFolder,
  onSelectHistory,
  onImportOpenAPI, importing,
  activeFilePath, isDirty,
}: SidebarProps) {
  return (
    <aside
      className="shrink-0 flex flex-col border-r border-[var(--color-border)] bg-[var(--color-card)]"
      style={{ width }}
    >
      {/* Tab bar */}
      <div className="flex border-b border-[var(--color-border)]">
        {(['workspace', 'history'] as SidebarTab[]).map(t => (
          <button
            key={t}
            type="button"
            onClick={() => onTabChange(t)}
            className={cn(
              'flex-1 py-1.5 text-[10px] font-semibold uppercase tracking-widest transition-colors',
              tab === t
                ? 'text-[var(--color-foreground)] border-b-2 border-[var(--color-accent)]'
                : 'text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] border-b-2 border-transparent',
            )}
          >
            {t}
          </button>
        ))}
      </div>

      <div className="flex-1 overflow-hidden flex flex-col">
        {tab === 'workspace' && (
          <WorkspacePanel
            workspacePath={workspacePath}
            fileTree={fileTree}
            onOpenWorkspace={onOpenWorkspace}
            onNewWorkspace={onNewWorkspace}
            onSelectFile={onSelectFile}
            onDeleteFile={onDeleteFile}
            onDeleteFolder={onDeleteFolder}
            onNewRequestInFolder={onNewRequestInFolder}
            onCreateFolder={onCreateFolder}
            onRenameFile={onRenameFile}
            onRenameFolder={onRenameFolder}
            onImportOpenAPI={onImportOpenAPI}
            importing={importing}
            activeFilePath={activeFilePath}
            isDirty={isDirty}
          />
        )}

        {tab === 'history' && (
          <div className="flex-1 overflow-auto">
            {history.length === 0 ? (
              <p className="p-3 text-xs text-[var(--color-muted-foreground)]">No requests yet.</p>
            ) : (
              <ul>
                {history.map(entry => (
                  <li key={entry.id}>
                    <button
                      type="button"
                      onClick={() => onSelectHistory(entry)}
                      className={cn(
                        'w-full text-left px-3 py-2 flex flex-col gap-0.5',
                        'hover:bg-[var(--color-muted)] transition-colors border-b border-[var(--color-border)]',
                      )}
                    >
                      <div className="flex items-center gap-1.5">
                        <span className={cn('text-[10px] font-mono font-bold shrink-0', METHOD_BADGE[entry.method] ?? 'text-[var(--color-foreground)]')}>
                          {entry.method}
                        </span>
                        <span className={cn('text-[10px] font-mono', statusColor(entry.statusCode))}>
                          {entry.statusCode || '—'}
                        </span>
                        <span className="ml-auto text-[10px] text-[var(--color-muted-foreground)] shrink-0">
                          {entry.durationMs}ms
                        </span>
                      </div>
                      <span className="text-[10px] text-[var(--color-muted-foreground)] truncate font-mono w-full">
                        {entry.url}
                      </span>
                    </button>
                  </li>
                ))}
              </ul>
            )}
          </div>
        )}
      </div>
    </aside>
  )
}

// ---- Request editor ----

type RequestEditorTab = 'Params' | 'Headers' | 'Body' | 'Auth'

type RequestEditorProps = {
  method: HttpMethod
  url: string
  loading: boolean
  queryParams: KVPair[]
  headers: KVPair[]
  body: BodyState
  auth: AuthState
  onMethodChange: (m: HttpMethod) => void
  onUrlChange: (u: string) => void
  onUrlPaste: (pastedUrl: string) => void
  onQueryParamsChange: (p: KVPair[]) => void
  onHeadersChange: (h: KVPair[]) => void
  onBodyChange: (b: BodyState) => void
  onAuthChange: (a: AuthState) => void
  onSend: () => void
}

function RequestEditor({
  method, url, loading,
  queryParams, headers, body, auth,
  onMethodChange, onUrlChange, onUrlPaste,
  onQueryParamsChange, onHeadersChange, onBodyChange, onAuthChange,
  onSend,
}: RequestEditorProps) {
  const [activeTab, setActiveTab] = useState<RequestEditorTab>('Params')

  const activeParamCount = queryParams.filter(p => p.enabled && p.name).length
  const activeHeaderCount = headers.filter(h => h.enabled && h.name).length

  const tabBadge: Record<RequestEditorTab, string | undefined> = {
    Params: activeParamCount > 0 ? String(activeParamCount) : undefined,
    Headers: activeHeaderCount > 0 ? String(activeHeaderCount) : undefined,
    Body: body.type !== 'none' ? '•' : undefined,
    Auth: auth.type !== 'none' ? '•' : undefined,
  }

  function handleKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === 'Enter' && !loading) onSend()
  }

  return (
    <div className="flex flex-col h-full">
      {/* URL bar */}
      <div className="flex items-center gap-2 px-3 py-2 border-b border-[var(--color-border)] bg-[var(--color-muted)]">
        <select
          value={method}
          onChange={e => onMethodChange(e.target.value as HttpMethod)}
          className={cn(
            'bg-[var(--color-muted)] text-xs font-mono font-semibold rounded px-2 py-1',
            'border border-[var(--color-border)] outline-none cursor-pointer',
            METHOD_COLOR[method],
          )}
        >
          {HTTP_METHODS.map(m => (
            <option key={m} value={m}>{m}</option>
          ))}
        </select>

        <input
          type="text"
          value={url}
          onChange={e => onUrlChange(e.target.value)}
          onKeyDown={handleKeyDown}
          onPaste={e => {
            const text = e.clipboardData.getData('text')
            if (text.includes('?')) {
              e.preventDefault()
              onUrlPaste(text)
            }
          }}
          placeholder="https://api.example.com/endpoint"
          className={cn(
            'flex-1 bg-[var(--color-muted)] text-xs font-mono rounded px-3 py-1',
            'border border-[var(--color-border)] outline-none',
            'focus:border-[var(--color-accent)]',
            'placeholder:text-[var(--color-muted-foreground)]',
            'text-[var(--color-foreground)]',
          )}
        />

        <button
          type="button"
          onClick={onSend}
          disabled={loading || !url.trim()}
          className={cn(
            'bg-[var(--color-accent)] text-white text-xs font-semibold rounded px-4 py-1',
            'transition-opacity disabled:opacity-40 hover:opacity-90 shrink-0',
          )}
        >
          {loading ? 'Sending…' : 'Send'}
        </button>
      </div>

      {/* Tabs */}
      <div className="flex border-b border-[var(--color-border)] px-3 shrink-0">
        {(['Params', 'Headers', 'Body', 'Auth'] as const).map(tab => (
          <button
            key={tab}
            type="button"
            onClick={() => setActiveTab(tab)}
            className={cn(
              'flex items-center gap-1 px-3 py-1.5 text-xs border-b-2 transition-colors',
              activeTab === tab
                ? 'border-[var(--color-accent)] text-[var(--color-foreground)]'
                : 'border-transparent text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)]',
            )}
          >
            {tab}
            {tabBadge[tab] && (
              <span className="text-[10px] text-[var(--color-accent)] font-semibold">
                {tabBadge[tab]}
              </span>
            )}
          </button>
        ))}
      </div>

      {/* Tab content */}
      <div className="flex-1 overflow-hidden flex flex-col">
        {activeTab === 'Params' && (
          <div className="flex-1 overflow-auto p-3">
            <KeyValueEditor
              pairs={queryParams}
              onChange={onQueryParamsChange}
              keyPlaceholder="Parameter"
              valuePlaceholder="Value"
            />
          </div>
        )}
        {activeTab === 'Headers' && (
          <div className="flex-1 overflow-auto p-3">
            <KeyValueEditor
              pairs={headers}
              onChange={onHeadersChange}
              keyPlaceholder="Header"
              valuePlaceholder="Value"
            />
          </div>
        )}
        {activeTab === 'Body' && (
          <div className="flex-1 overflow-hidden flex flex-col p-3">
            <BodyEditor body={body} onChange={onBodyChange} />
          </div>
        )}
        {activeTab === 'Auth' && (
          <div className="flex-1 overflow-auto p-3">
            <AuthEditor auth={auth} onChange={onAuthChange} />
          </div>
        )}
      </div>
    </div>
  )
}

// ---- Timing bar ----

type TimingSegment = { label: string; ms: number; color: string }

function TimingBar({ timing, total }: { timing: ResponseData['timing']; total: number }) {
  if (!timing || total === 0) return null

  const segments: TimingSegment[] = [
    { label: 'DNS',      ms: timing.dnsMs,      color: 'bg-purple-500' },
    { label: 'Connect',  ms: timing.connectMs,  color: 'bg-blue-500' },
    { label: 'TLS',      ms: timing.tlsMs,      color: 'bg-yellow-500' },
    { label: 'TTFB',     ms: timing.ttfbMs,     color: 'bg-emerald-500' },
    { label: 'Transfer', ms: timing.transferMs, color: 'bg-cyan-500' },
  ].filter(s => s.ms > 0)

  if (segments.length === 0) return null

  const sum = segments.reduce((a, s) => a + s.ms, 0)
  const scale = sum > 0 ? total / sum : 1

  return (
    <div className="w-full mt-1.5 flex flex-col gap-0.5">
      <div className="flex h-2 rounded overflow-hidden gap-px">
        {segments.map(s => (
          <div
            key={s.label}
            title={`${s.label}: ${s.ms}ms`}
            className={cn('h-full', s.color)}
            style={{ width: `${Math.max((s.ms / (total || sum)) * 100 * scale, 1)}%` }}
          />
        ))}
      </div>
      <div className="flex gap-3 flex-wrap">
        {segments.map(s => (
          <span key={s.label} className="flex items-center gap-1 text-label text-[var(--color-muted-foreground)]">
            <span className={cn('inline-block rounded-sm shrink-0', s.color)} style={{ width: 'calc(var(--font-size-code) * 0.8)', height: 'calc(var(--font-size-code) * 0.8)' }} />
            {s.label} {s.ms}ms
          </span>
        ))}
      </div>
    </div>
  )
}

// ---- JSON syntax highlighting ----

function highlightJSON(json: string): React.ReactNode {
  const nodes: React.ReactNode[] = []
  // Order matters: key (string + lookahead colon) must come before plain string
  const re = /("(?:[^"\\]|\\.)*")(?=\s*:)|("(?:[^"\\]|\\.)*")|([-]?\d+(?:\.\d+)?(?:[eE][+-]?\d+)?)|(true|false|null)|([{}\[\],:])/g
  let last = 0
  let i = 0
  for (const m of json.matchAll(re)) {
    if (m.index > last) nodes.push(json.slice(last, m.index))
    if      (m[1]) nodes.push(<span key={i++} style={{ color: 'var(--color-json-key)' }}>{m[1]}</span>)
    else if (m[2]) nodes.push(<span key={i++} style={{ color: 'var(--color-json-string)' }}>{m[2]}</span>)
    else if (m[3]) nodes.push(<span key={i++} style={{ color: 'var(--color-json-number)' }}>{m[3]}</span>)
    else if (m[4]) nodes.push(<span key={i++} style={{ color: 'var(--color-json-literal)' }}>{m[4]}</span>)
    else if (m[5]) nodes.push(<span key={i++} style={{ color: 'var(--color-json-punct)' }}>{m[5]}</span>)
    last = m.index + m[0].length
  }
  if (last < json.length) nodes.push(json.slice(last))
  return nodes
}

function JsonBody({ body }: { body: string }) {
  const trimmed = body.trim()
  const looksLikeJson = trimmed.startsWith('{') || trimmed.startsWith('[')
  let isJson = false
  if (looksLikeJson) try { JSON.parse(body); isJson = true } catch { /* plain text */ }

  const content = isJson && body.length < 500_000
    ? highlightJSON(prettyBody(body))
    : prettyBody(body)

  return (
    <pre className="text-xs font-mono whitespace-pre-wrap break-all leading-relaxed" style={{ color: 'var(--color-foreground)' }}>
      {content}
    </pre>
  )
}

// ---- Response viewer ----

type ResponseViewerProps = {
  response: ResponseData | null
  loading: boolean
}

function ResponseViewer({ response, loading }: ResponseViewerProps) {
  const [activeTab, setActiveTab] = useState<'Body' | 'Headers'>('Body')

  return (
    <div className="flex flex-col h-full border-l border-[var(--color-border)]">
      {/* Status bar */}
      <div className="flex items-center gap-3 px-3 py-2 border-b border-[var(--color-border)] shrink-0">
        {response && !response.error && response.statusCode > 0 ? (
          <>
            <span className={cn('text-xs font-mono font-semibold', statusColor(response.statusCode))}>
              {response.statusCode}
            </span>
            <span className="text-xs text-[var(--color-muted-foreground)]">{response.status}</span>
            <span className="ml-auto text-xs text-[var(--color-muted-foreground)]">
              {response.durationMs} ms
            </span>
            <span className="text-xs text-[var(--color-muted-foreground)]">
              {formatSize(response.size)}
            </span>
          </>
        ) : (
          <span className="text-xs text-[var(--color-muted-foreground)]">Response</span>
        )}
        {response?.timing && !response.error && <TimingBar timing={response.timing} total={response.durationMs} />}
      </div>

      {/* Tabs */}
      {response && !response.error && (
        <div className="flex border-b border-[var(--color-border)] px-3 shrink-0">
          {(['Body', 'Headers'] as const).map(tab => (
            <button
              key={tab}
              type="button"
              onClick={() => setActiveTab(tab)}
              className={cn(
                'px-3 py-1.5 text-xs border-b-2 transition-colors',
                activeTab === tab
                  ? 'border-[var(--color-accent)] text-[var(--color-foreground)]'
                  : 'border-transparent text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)]',
              )}
            >
              {tab}
            </button>
          ))}
        </div>
      )}

      {/* Content */}
      <div className="flex-1 overflow-auto p-3">
        {loading && (
          <p className="text-xs text-[var(--color-muted-foreground)] animate-pulse">Sending request…</p>
        )}
        {!loading && !response && (
          <p className="text-xs text-[var(--color-muted-foreground)]">
            Send a request to see the response.
          </p>
        )}
        {!loading && response?.error && (
          <div className="text-xs text-red-400 font-mono whitespace-pre-wrap break-all">
            {response.error}
          </div>
        )}
        {!loading && response && !response.error && activeTab === 'Body' && (
          <JsonBody body={response.body} />
        )}
        {!loading && response && !response.error && activeTab === 'Headers' && (
          <table className="w-full text-xs font-mono">
            <tbody>
              {response.headers.map((h, i) => (
                <tr key={i} className="border-b border-[var(--color-border)]">
                  <td className="py-1 pr-4 text-[var(--color-muted-foreground)] whitespace-nowrap">{h.name}</td>
                  <td className="py-1 text-[var(--color-foreground)] break-all">{h.value}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  )
}

// ---- Tab state ----

type TabState = {
  id: string
  method: HttpMethod
  url: string
  queryParams: KVPair[]
  headers: KVPair[]
  body: BodyState
  auth: AuthState
  name: string | null
  filePath: string | null
  targetDir: string | null
  savedKey: string
  response: ResponseData | null
  loading: boolean
  saveError: string | null
  editingName: boolean
}

function createBlankTab(targetDir: string | null = null): TabState {
  const savedKey = computeStateKey('GET', '', [], [], { type: 'none', content: '' }, { type: 'none' }, null)
  return {
    id: crypto.randomUUID(),
    method: 'GET',
    url: '',
    queryParams: [],
    headers: [],
    body: { type: 'none', content: '' },
    auth: { type: 'none' },
    name: null,
    filePath: null,
    targetDir,
    savedKey,
    response: null,
    loading: false,
    saveError: null,
    editingName: false,
  }
}

function isTabDirty(tab: TabState): boolean {
  return computeStateKey(tab.method, tab.url, tab.queryParams, tab.headers, tab.body, tab.auth, tab.name) !== tab.savedKey
}

function isBlankTab(tab: TabState): boolean {
  return tab.url === '' && tab.name === null && tab.filePath === null && !isTabDirty(tab)
}

// ---- App ----

export default function App() {
  // --- Tabs ---
  const [{ tabs, activeTabId }, setTabsState] = useState(() => {
    const tab = createBlankTab()
    return { tabs: [tab] as TabState[], activeTabId: tab.id }
  })

  // Always-current ref used by stable callbacks (keydown handler, etc.)
  const tabsStateRef = useRef({ tabs, activeTabId })
  tabsStateRef.current = { tabs, activeTabId }

  // --- Resize state ---
  const [sidebarWidth, setSidebarWidth] = useState(224)
  const [responseWidth, setResponseWidth] = useState(0)
  const draggingRef = useRef<'sidebar' | 'response' | null>(null)

  // --- Global UI state ---
  const [settings, setSettings] = useState<AppSettings>({ theme: 'dark', fontSize: 13, defaultTimeoutMs: 30000, proxyUrl: '', tlsSkipVerify: false })
  const [showSettings, setShowSettings] = useState(false)
  const [showEnvEditor, setShowEnvEditor] = useState(false)
  const [editingEnvPath, setEditingEnvPath] = useState<string | null>(null)
  const [workspacePath, setWorkspacePath] = useState<string>('')
  const [fileTree, setFileTree] = useState<FileNode[]>([])
  const [environments, setEnvironments] = useState<EnvInfo[]>([])
  const [activeEnv, setActiveEnv] = useState<EnvInfo | null>(null)
  const [sidebarTab, setSidebarTab] = useState<SidebarTab>('workspace')
  const [importing, setImporting] = useState(false)
  const [history, setHistory] = useState<HistoryEntry[]>([])
  const [closingDirtyTabId, setClosingDirtyTabId] = useState<string | null>(null)
  const [deletingFilePath, setDeletingFilePath] = useState<string | null>(null)
  const [showAbout, setShowAbout] = useState(false)
  const [deletingFolderPath, setDeletingFolderPath] = useState<string | null>(null)
  const [newFolderDialog, setNewFolderDialog] = useState<{ parentPath: string; parentLabel: string } | null>(null)
  const [newFolderName, setNewFolderName] = useState('')

  // --- Derived ---
  const activeTab = tabs.find(t => t.id === activeTabId) ?? tabs[0]

  // --- Tab helpers ---

  function updateTab(id: string, patch: Partial<TabState>) {
    setTabsState(prev => ({
      ...prev,
      tabs: prev.tabs.map(t => t.id === id ? { ...t, ...patch } : t),
    }))
  }

  function switchTo(id: string) {
    setTabsState(prev => ({ ...prev, activeTabId: id }))
  }

  function openFileAsTab(r: RequestFileIO, path: string) {
    const state = requestFileIOToState(r)
    const savedKey = computeStateKey(state.method, state.url, state.queryParams, state.headers, state.body, state.auth, r.name || null)
    const newTab: TabState = {
      id: crypto.randomUUID(),
      ...state,
      name: r.name || null,
      filePath: path,
      targetDir: null,
      savedKey,
      response: null,
      loading: false,
      saveError: null,
      editingName: false,
    }
    setTabsState(prev => {
      // Reuse blank active tab if it's untouched
      const active = prev.tabs.find(t => t.id === prev.activeTabId)
      if (active && isBlankTab(active)) {
        return {
          tabs: prev.tabs.map(t => t.id === prev.activeTabId ? { ...newTab, id: prev.activeTabId } : t),
          activeTabId: prev.activeTabId,
        }
      }
      return { tabs: [...prev.tabs, newTab], activeTabId: newTab.id }
    })
  }

  // ---- Init ----

  const refreshWorkspace = useCallback(async () => {
    try {
      const [path, tree, envs, activeE] = await Promise.all([
        getWorkspacePath() as Promise<string>,
        listRequests(),
        listEnvironments(),
        getActiveEnvironment(),
      ])
      setWorkspacePath(path ?? '')
      setFileTree(tree ?? [])
      setEnvironments(envs ?? [])
      setActiveEnv(activeE)
    } catch {
      // best-effort
    }
  }, [])

  const refreshHistory = useCallback(async () => {
    try { setHistory(await getHistory()) } catch { /* best-effort */ }
  }, [])

  useEffect(() => {
    refreshWorkspace()
    refreshHistory()
    getSettings().then(s => {
      setSettings(s)
      applySettings(s)
    }).catch(() => {})
  }, [refreshWorkspace, refreshHistory])

  // Keep Go side informed so OnBeforeClose can guard unsaved changes.
  const hasAnyDirtyTab = tabs.some(t => isTabDirty(t))
  useEffect(() => {
    setHasDirtyTabs(hasAnyDirtyTab).catch(() => {})
  }, [hasAnyDirtyTab])

  // ---- Resize ----

  useEffect(() => {
    function onMouseMove(e: MouseEvent) {
      if (draggingRef.current === 'sidebar') {
        setSidebarWidth(prev => Math.max(160, Math.min(480, prev + e.movementX)))
      } else if (draggingRef.current === 'response') {
        setResponseWidth(prev => {
          const current = prev > 0 ? prev : window.innerWidth * 0.45
          return Math.max(200, Math.min(window.innerWidth - 400, current - e.movementX))
        })
      }
    }
    function onMouseUp() {
      draggingRef.current = null
      document.body.style.cursor = ''
      document.body.style.userSelect = ''
    }
    document.addEventListener('mousemove', onMouseMove)
    document.addEventListener('mouseup', onMouseUp)
    return () => {
      document.removeEventListener('mousemove', onMouseMove)
      document.removeEventListener('mouseup', onMouseUp)
    }
  }, [])

  // ---- Keyboard shortcuts ----

  const saveTabByIdRef = useRef<((id: string) => Promise<boolean>) | null>(null)

  useEffect(() => {
    function onKeyDown(e: KeyboardEvent) {
      if ((e.ctrlKey || e.metaKey) && e.key === 's') {
        e.preventDefault()
        saveTabByIdRef.current?.(tabsStateRef.current.activeTabId)
      }
      if ((e.ctrlKey || e.metaKey) && e.key === 'n') {
        e.preventDefault()
        handleNewTab()
      }
      if ((e.ctrlKey || e.metaKey) && e.key === 'w') {
        e.preventDefault()
        handleCloseTab(tabsStateRef.current.activeTabId)
      }
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, []) // registered once; reads from refs

  // ---- Tab operations ----

  function handleNewTab() {
    const tab = createBlankTab()
    setTabsState(prev => ({ tabs: [...prev.tabs, tab], activeTabId: tab.id }))
  }

  function handleSelectTab(id: string) {
    switchTo(id)
  }

  function handleCloseTab(id: string) {
    const { tabs: currentTabs } = tabsStateRef.current
    const tab = currentTabs.find(t => t.id === id)
    if (!tab) return
    if (isTabDirty(tab)) {
      setClosingDirtyTabId(id)
    } else {
      removeTab(id)
    }
  }

  function removeTab(id: string) {
    setTabsState(prev => {
      if (prev.tabs.length === 1) {
        // Reset last tab to blank instead of removing
        const blank = createBlankTab()
        return { tabs: [{ ...blank, id: prev.tabs[0].id }], activeTabId: prev.tabs[0].id }
      }
      const idx = prev.tabs.findIndex(t => t.id === id)
      const remaining = prev.tabs.filter(t => t.id !== id)
      const newActiveId = prev.activeTabId === id
        ? (remaining[Math.min(idx, remaining.length - 1)]?.id ?? remaining[0].id)
        : prev.activeTabId
      return { tabs: remaining, activeTabId: newActiveId }
    })
  }

  // ---- Workspace operations ----

  async function handleOpenWorkspace() {
    try {
      const path = await openWorkspace()
      if (path) {
        setWorkspacePath(path)
        await refreshWorkspace()
      }
    } catch (err) {
      console.error('open workspace:', err)
    }
  }

  async function handleNewWorkspace() {
    try {
      const path = await newWorkspace()
      if (path) {
        setWorkspacePath(path)
        await refreshWorkspace()
      }
    } catch (err) {
      console.error('new workspace:', err)
    }
  }

  async function handleSelectFile(path: string) {
    // Reuse existing tab if the file is already open
    const existing = tabsStateRef.current.tabs.find(t => t.filePath === path)
    if (existing) {
      switchTo(existing.id)
      return
    }
    try {
      const r = await loadRequest(path)
      if (!r) return
      openFileAsTab(r, path)
    } catch (err) {
      console.error('load request:', err)
    }
  }

  async function handleOpenRequest() {
    try {
      const r = await openRequest()
      if (!r || !r.filePath) return
      const existing = tabsStateRef.current.tabs.find(t => t.filePath === r.filePath)
      if (existing) { switchTo(existing.id); return }
      openFileAsTab(r, r.filePath)
    } catch (err) {
      console.error('open request:', err)
    }
  }

  function handleNewRequest() {
    handleNewTab()
  }

  async function saveTabById(tabId: string): Promise<boolean> {
    const tab = tabsStateRef.current.tabs.find(t => t.id === tabId)
    if (!tab) return false
    updateTab(tabId, { saveError: null })
    const name = tab.name?.trim() || tab.url.trim().split('/').filter(Boolean).pop() || 'request'
    try {
      const req: RequestFileIO = {
        schemaVersion: '1',
        name,
        method: tab.method,
        url: tab.url.trim(),
        headers: tab.headers.filter(h => h.enabled && h.name.trim()).map(h => ({ name: h.name, value: h.value })),
        queryParams: tab.queryParams.filter(p => p.enabled && p.name.trim()).map(p => ({ name: p.name, value: p.value })),
        body: tab.body.type === 'none' ? undefined : { type: tab.body.type, content: tab.body.content },
        auth: authStateToPayload(tab.auth) ?? undefined,
        filePath: tab.filePath ?? '',
        targetDir: tab.targetDir ?? '',
      }
      const savedPath = await saveRequest(req)
      if (savedPath) {
        const savedKey = computeStateKey(tab.method, tab.url, tab.queryParams, tab.headers, tab.body, tab.auth, name)
        updateTab(tabId, { name, filePath: savedPath, savedKey })
        await refreshWorkspace()
        return true
      }
      return false
    } catch (err) {
      updateTab(tabId, { saveError: String(err).replace(/^Error: /, '') })
      return false
    }
  }

  // Keep ref current so the keydown handler always calls the latest version
  saveTabByIdRef.current = saveTabById

  async function handleSaveRequest(): Promise<boolean> {
    return saveTabById(activeTabId)
  }

  // ---- Environment operations ----

  async function handleEnvChange(path: string) {
    try {
      await selectEnvironment(path)
      const envs = await listEnvironments()
      setEnvironments(envs ?? [])
      const active = path ? envs.find(e => e.path === path) ?? null : null
      setActiveEnv(active)
    } catch (err) {
      console.error('select environment:', err)
    }
  }

  // ---- History ----

  function handleSelectHistory(entry: HistoryEntry) {
    const method = (HTTP_METHODS as readonly string[]).includes(entry.method)
      ? entry.method as HttpMethod : 'GET'
    const savedKey = computeStateKey(method, entry.url, [], [], { type: 'none', content: '' }, { type: 'none' }, null)
    const newTab: TabState = {
      id: crypto.randomUUID(),
      method,
      url: entry.url,
      queryParams: [],
      headers: [],
      body: { type: 'none', content: '' },
      auth: { type: 'none' },
      name: null,
      filePath: null,
      targetDir: null,
      savedKey,
      response: null,
      loading: false,
      saveError: null,
      editingName: false,
    }
    setTabsState(prev => {
      const active = prev.tabs.find(t => t.id === prev.activeTabId)
      if (active && isBlankTab(active)) {
        return {
          tabs: prev.tabs.map(t => t.id === prev.activeTabId ? { ...newTab, id: prev.activeTabId } : t),
          activeTabId: prev.activeTabId,
        }
      }
      return { tabs: [...prev.tabs, newTab], activeTabId: newTab.id }
    })
  }

  // ---- Settings ----

  function applySettings(s: AppSettings) {
    const root = document.documentElement
    root.style.setProperty('--font-size-code', `${s.fontSize}px`)
    if (s.theme === 'light') {
      root.classList.add('light')
      root.classList.remove('dark')
    } else {
      root.classList.add('dark')
      root.classList.remove('light')
    }
  }

  async function handleToggleTheme() {
    const next = { ...settings, theme: settings.theme === 'dark' ? 'light' : 'dark' }
    await handleSaveSettings(next as AppSettings)
  }

  async function handleSaveSettings(s: AppSettings) {
    try {
      const saved = await saveSettings(s)
      setSettings(saved)
      applySettings(saved)
    } catch (err) {
      console.error('save settings:', err)
    }
  }

  // ---- OpenAPI import ----

  async function handleImportOpenAPI() {
    setImporting(true)
    try {
      const result = await importOpenAPISpec()
      if (result.error) {
        console.error('import error:', result.error)
      } else {
        await refreshWorkspace()
      }
    } catch (err) {
      console.error('import failed:', err)
    } finally {
      setImporting(false)
    }
  }

  // ---- Delete request file ----

  function handleDeleteFile(path: string) {
    setDeletingFilePath(path)
  }

  async function confirmDeleteFile() {
    if (!deletingFilePath) return
    const path = deletingFilePath
    setDeletingFilePath(null)
    try {
      await deleteRequest(path)
      // Close any open tab with this file path (no dirty-save prompt — file is gone)
      setTabsState(prev => {
        const remaining = prev.tabs.filter(t => t.filePath !== path)
        if (remaining.length === prev.tabs.length) {
          // No tab had this file open, just refresh workspace below
          return prev
        }
        if (remaining.length === 0) {
          const blank = createBlankTab()
          return { tabs: [blank], activeTabId: blank.id }
        }
        const activeStillExists = remaining.some(t => t.id === prev.activeTabId)
        return {
          tabs: remaining,
          activeTabId: activeStillExists ? prev.activeTabId : remaining[remaining.length - 1].id,
        }
      })
      await refreshWorkspace()
    } catch (err) {
      console.error('delete request:', err)
    }
  }

  // ---- Folder operations ----

  function handleCreateFolder(parentPath: string) {
    const parentLabel = parentPath
      ? parentPath.replace(/\\/g, '/').split('/').pop() ?? 'requests'
      : 'requests'
    setNewFolderDialog({ parentPath, parentLabel })
    setNewFolderName('')
  }

  async function confirmCreateFolder() {
    if (!newFolderDialog || !newFolderName.trim()) return
    try {
      await createFolder(newFolderDialog.parentPath, newFolderName.trim())
      setNewFolderDialog(null)
      setNewFolderName('')
      await refreshWorkspace()
    } catch (err) {
      // Show error inline in the dialog by keeping it open
      console.error('create folder:', err)
    }
  }

  function handleDeleteFolder(path: string) {
    setDeletingFolderPath(path)
  }

  async function confirmDeleteFolder() {
    if (!deletingFolderPath) return
    const path = deletingFolderPath
    setDeletingFolderPath(null)
    try {
      await deleteFolder(path)
      await refreshWorkspace()
    } catch (err) {
      console.error('delete folder:', err)
    }
  }

  function handleNewRequestInFolder(folderPath: string) {
    const tab = createBlankTab(folderPath)
    setTabsState(prev => ({ tabs: [...prev.tabs, tab], activeTabId: tab.id }))
  }

  // ---- Rename operations ----

  function rebasePath(oldBase: string, newBase: string, p: string | null): string | null {
    if (!p) return p
    if (p === oldBase) return newBase
    if (p.startsWith(oldBase + '\\') || p.startsWith(oldBase + '/')) return newBase + p.slice(oldBase.length)
    return p
  }

  async function handleRenameFile(path: string, newName: string) {
    try {
      const newPath = await renameRequest(path, newName)
      if (!newPath) return
      setTabsState(prev => ({
        ...prev,
        tabs: prev.tabs.map(t => {
          if (t.filePath !== path) return t
          const savedKey = computeStateKey(t.method, t.url, t.queryParams, t.headers, t.body, t.auth, newName)
          return { ...t, name: newName, filePath: newPath, savedKey }
        }),
      }))
      await refreshWorkspace()
    } catch (err) {
      console.error('rename request:', err)
    }
  }

  async function handleRenameFolder(path: string, newName: string) {
    try {
      const newPath = await renameFolder(path, newName)
      if (!newPath) return
      setTabsState(prev => ({
        ...prev,
        tabs: prev.tabs.map(t => {
          const fp = rebasePath(path, newPath, t.filePath)
          const td = rebasePath(path, newPath, t.targetDir)
          if (fp === t.filePath && td === t.targetDir) return t
          return { ...t, filePath: fp, targetDir: td }
        }),
      }))
      await refreshWorkspace()
    } catch (err) {
      console.error('rename folder:', err)
    }
  }

  // ---- URL paste param extraction ----

  function handleUrlPaste(raw: string) {
    try {
      const qIdx = raw.indexOf('?')
      const baseUrl = raw.slice(0, qIdx)
      const search = raw.slice(qIdx + 1)
      const parsed = new URLSearchParams(search)
      const extracted: KVPair[] = []
      parsed.forEach((value, name) => {
        extracted.push({ id: crypto.randomUUID(), name, value, enabled: true })
      })
      updateTab(activeTabId, { url: baseUrl })
      if (extracted.length > 0) {
        const names = new Set(extracted.map(p => p.name.toLowerCase()))
        const kept = activeTab.queryParams.filter(p => !names.has(p.name.toLowerCase()))
        updateTab(activeTabId, { queryParams: [...kept, ...extracted] })
      }
    } catch {
      updateTab(activeTabId, { url: raw })
    }
  }

  // ---- Request execution ----

  async function handleSend() {
    if (!activeTab.url.trim() || activeTab.loading) return
    const tabId = activeTabId // capture at call time
    updateTab(tabId, { loading: true, response: null })
    try {
      const result = await executeRequest({
        method: activeTab.method,
        url: activeTab.url.trim(),
        headers: activeTab.headers.filter(h => h.enabled && h.name.trim()).map(h => ({ name: h.name, value: h.value })),
        queryParams: activeTab.queryParams.filter(p => p.enabled && p.name.trim()).map(p => ({ name: p.name, value: p.value })),
        body: activeTab.body.type === 'none' ? undefined : { type: activeTab.body.type, content: activeTab.body.content },
        auth: authStateToPayload(activeTab.auth),
        timeoutMs: 0,
      })
      updateTab(tabId, { response: result })
    } catch (err) {
      updateTab(tabId, { response: { statusCode: 0, status: '', headers: [], body: '', durationMs: 0, size: 0, timing: { dnsMs: 0, connectMs: 0, tlsMs: 0, ttfbMs: 0, transferMs: 0 }, error: String(err) } })
    } finally {
      updateTab(tabId, { loading: false })
      refreshHistory()
    }
  }

  // ---- Menus ----

  const menus: MenuDef[] = [
    {
      label: 'File',
      items: [
        { kind: 'action', label: 'New Tab',           shortcut: 'Ctrl+N', onClick: handleNewTab },
        { kind: 'action', label: 'Open Request…',     shortcut: 'Ctrl+O', onClick: handleOpenRequest },
        { kind: 'action', label: 'Save Request',      shortcut: 'Ctrl+S', onClick: handleSaveRequest },
        { kind: 'separator' },
        { kind: 'action', label: 'Import OpenAPI Spec…', onClick: handleImportOpenAPI, disabled: !workspacePath },
        { kind: 'separator' },
        { kind: 'action', label: 'Open Workspace…',   onClick: handleOpenWorkspace },
        { kind: 'action', label: 'New Workspace…',    onClick: handleNewWorkspace },
      ],
    },
    {
      label: 'View',
      items: [
        { kind: 'action', label: settings.theme === 'dark' ? 'Switch to Light Theme' : 'Switch to Dark Theme', onClick: handleToggleTheme },
        { kind: 'separator' },
        { kind: 'action', label: 'Settings…', onClick: () => setShowSettings(true) },
      ],
    },
    {
      label: 'Help',
      items: [
        { kind: 'action', label: 'About EndpointDesk', onClick: () => setShowAbout(true) },
      ],
    },
  ]

  // ---- Tab info for TabBar ----

  const tabInfos = tabs.map(t => ({
    id: t.id,
    name: t.name,
    method: t.method,
    isDirty: isTabDirty(t),
  }))

  // ---- Dirty-close confirmation tab ----
  const closingTab = closingDirtyTabId ? tabs.find(t => t.id === closingDirtyTabId) : null

  // ---- Render ----

  return (
    <div className="flex flex-col h-screen w-screen overflow-hidden">
      <MenuBar menus={menus} />

      {/* Delete request confirmation */}
      {deletingFilePath && (
        <div
          className="fixed inset-0 z-[200] flex items-center justify-center bg-black/60"
          onClick={() => setDeletingFilePath(null)}
        >
          <div
            className="bg-[var(--color-card)] border border-[var(--color-border)] rounded-lg p-5 w-80 shadow-2xl flex flex-col gap-4"
            onClick={e => e.stopPropagation()}
          >
            <div>
              <h3 className="text-sm font-semibold text-[var(--color-foreground)]">Delete request</h3>
              <p className="text-xs text-[var(--color-muted-foreground)] mt-1">
                This will permanently delete{' '}
                <span className="font-medium text-[var(--color-foreground)] font-mono">
                  {deletingFilePath.replace(/\\/g, '/').split('/').pop()}
                </span>{' '}
                from disk. This cannot be undone.
              </p>
            </div>
            <div className="flex gap-2 justify-end">
              <button
                type="button"
                onClick={() => setDeletingFilePath(null)}
                className="px-3 py-1 text-xs rounded bg-[var(--color-muted)] text-[var(--color-foreground)] hover:bg-[var(--color-border)] transition-colors"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={confirmDeleteFile}
                className="px-3 py-1 text-xs rounded bg-red-600 text-white hover:bg-red-700 transition-colors"
              >
                Delete
              </button>
            </div>
          </div>
        </div>
      )}

      {/* About dialog */}
      {showAbout && (
        <div
          className="fixed inset-0 z-[200] flex items-center justify-center bg-black/60"
          onClick={() => setShowAbout(false)}
        >
          <div
            className="bg-[var(--color-card)] border border-[var(--color-border)] rounded-xl shadow-2xl flex flex-col w-[480px]"
            onClick={e => e.stopPropagation()}
          >
            {/* Header band with icon */}
            <div className="flex flex-col items-center gap-4 px-8 pt-8 pb-6 border-b border-[var(--color-border)]">
              <img
                src="/appicon.png"
                alt="EndpointDesk icon"
                className="w-24 h-24 rounded-2xl shadow-lg select-none"
                draggable={false}
              />
              <div className="flex flex-col items-center gap-1">
                <h2 className="text-xl font-bold text-[var(--color-foreground)] tracking-tight">EndpointDesk</h2>
                <p className="text-xs text-[var(--color-muted-foreground)]">Version 0.1.0 — Public MVP</p>
              </div>
              <p className="text-xs text-[var(--color-muted-foreground)] leading-relaxed text-center max-w-xs">
                API Contract Validation and Spec Conformance Testing — with a full API development workbench.
                Local-first, open-source, and git-native.
              </p>
            </div>

            {/* Details */}
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

            {/* Footer */}
            <div className="flex justify-between items-center px-8 py-4 border-t border-[var(--color-border)]">
              <span className="text-[10px] text-[var(--color-muted-foreground)]">
                © {new Date().getFullYear()} Fuat Yazar
              </span>
              <button
                type="button"
                onClick={() => setShowAbout(false)}
                className="px-4 py-1.5 text-xs rounded bg-[var(--color-muted)] text-[var(--color-foreground)] hover:bg-[var(--color-border)] transition-colors"
              >
                Close
              </button>
            </div>
          </div>
        </div>
      )}

      {/* New folder dialog */}
      {newFolderDialog && (
        <div
          className="fixed inset-0 z-[200] flex items-center justify-center bg-black/60"
          onClick={() => setNewFolderDialog(null)}
        >
          <div
            className="bg-[var(--color-card)] border border-[var(--color-border)] rounded-lg p-5 w-80 shadow-2xl flex flex-col gap-4"
            onClick={e => e.stopPropagation()}
          >
            <div>
              <h3 className="text-sm font-semibold text-[var(--color-foreground)]">New folder</h3>
              <p className="text-xs text-[var(--color-muted-foreground)] mt-0.5">
                Inside <span className="font-medium text-[var(--color-foreground)] font-mono">{newFolderDialog.parentLabel}</span>
              </p>
            </div>
            <input
              autoFocus
              type="text"
              value={newFolderName}
              onChange={e => setNewFolderName(e.target.value)}
              onKeyDown={e => {
                if (e.key === 'Enter') confirmCreateFolder()
                if (e.key === 'Escape') setNewFolderDialog(null)
              }}
              placeholder="folder-name"
              className={cn(
                'w-full text-xs rounded px-3 py-1.5 bg-[var(--color-muted)]',
                'border border-[var(--color-border)] outline-none focus:border-[var(--color-accent)]',
                'text-[var(--color-foreground)] placeholder:text-[var(--color-muted-foreground)]',
              )}
            />
            <div className="flex gap-2 justify-end">
              <button
                type="button"
                onClick={() => setNewFolderDialog(null)}
                className="px-3 py-1 text-xs rounded bg-[var(--color-muted)] text-[var(--color-foreground)] hover:bg-[var(--color-border)] transition-colors"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={confirmCreateFolder}
                disabled={!newFolderName.trim()}
                className="px-3 py-1 text-xs rounded bg-[var(--color-accent)] text-white hover:opacity-90 transition-opacity disabled:opacity-40"
              >
                Create
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Delete folder confirmation */}
      {deletingFolderPath && (
        <div
          className="fixed inset-0 z-[200] flex items-center justify-center bg-black/60"
          onClick={() => setDeletingFolderPath(null)}
        >
          <div
            className="bg-[var(--color-card)] border border-[var(--color-border)] rounded-lg p-5 w-80 shadow-2xl flex flex-col gap-4"
            onClick={e => e.stopPropagation()}
          >
            <div>
              <h3 className="text-sm font-semibold text-[var(--color-foreground)]">Delete folder</h3>
              <p className="text-xs text-[var(--color-muted-foreground)] mt-1">
                Delete empty folder{' '}
                <span className="font-medium text-[var(--color-foreground)] font-mono">
                  {deletingFolderPath.replace(/\\/g, '/').split('/').pop()}
                </span>
                ? This cannot be undone.
              </p>
            </div>
            <div className="flex gap-2 justify-end">
              <button
                type="button"
                onClick={() => setDeletingFolderPath(null)}
                className="px-3 py-1 text-xs rounded bg-[var(--color-muted)] text-[var(--color-foreground)] hover:bg-[var(--color-border)] transition-colors"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={confirmDeleteFolder}
                className="px-3 py-1 text-xs rounded bg-red-600 text-white hover:bg-red-700 transition-colors"
              >
                Delete
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Close-dirty-tab confirmation */}
      {closingTab && (
        <div
          className="fixed inset-0 z-[200] flex items-center justify-center bg-black/60"
          onClick={() => setClosingDirtyTabId(null)}
        >
          <div
            className="bg-[var(--color-card)] border border-[var(--color-border)] rounded-lg p-5 w-80 shadow-2xl flex flex-col gap-4"
            onClick={e => e.stopPropagation()}
          >
            <div>
              <h3 className="text-sm font-semibold text-[var(--color-foreground)]">Unsaved changes</h3>
              <p className="text-xs text-[var(--color-muted-foreground)] mt-1">
                <span className="font-medium text-[var(--color-foreground)]">
                  {closingTab.name ?? 'This request'}
                </span>{' '}
                has unsaved changes. Save before closing?
              </p>
            </div>
            <div className="flex gap-2 justify-end">
              <button
                type="button"
                onClick={() => setClosingDirtyTabId(null)}
                className="px-3 py-1 text-xs rounded bg-[var(--color-muted)] text-[var(--color-foreground)] hover:bg-[var(--color-border)] transition-colors"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={() => { setClosingDirtyTabId(null); removeTab(closingTab.id) }}
                className="px-3 py-1 text-xs rounded bg-[var(--color-muted)] text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] hover:bg-[var(--color-border)] transition-colors"
              >
                Discard
              </button>
              <button
                type="button"
                onClick={async () => {
                  const ok = await saveTabById(closingTab.id)
                  if (ok) { setClosingDirtyTabId(null); removeTab(closingTab.id) }
                }}
                className="px-3 py-1 text-xs rounded bg-[var(--color-accent)] text-white hover:opacity-90 transition-opacity"
              >
                Save & Close
              </button>
            </div>
          </div>
        </div>
      )}

      <div className="flex flex-1 overflow-hidden">
        {showSettings && (
          <SettingsPanel
            settings={settings}
            onSave={handleSaveSettings}
            onPreview={applySettings}
            onClose={() => { applySettings(settings); setShowSettings(false) }}
          />
        )}
        {showEnvEditor && (
          <EnvironmentEditor
            envPath={editingEnvPath}
            workspacePath={workspacePath}
            onClose={() => setShowEnvEditor(false)}
            onSaved={async saved => {
              setShowEnvEditor(false)
              await refreshWorkspace()
              try { await selectEnvironment(saved.path) } catch { /* best-effort */ }
              await refreshWorkspace()
            }}
            onDeleted={async () => {
              setShowEnvEditor(false)
              await refreshWorkspace()
            }}
          />
        )}

        <Sidebar
          width={sidebarWidth}
          tab={sidebarTab}
          onTabChange={setSidebarTab}
          workspacePath={workspacePath}
          fileTree={fileTree}
          history={history}
          onOpenWorkspace={handleOpenWorkspace}
          onNewWorkspace={handleNewWorkspace}
          onSelectFile={handleSelectFile}
          onDeleteFile={handleDeleteFile}
          onDeleteFolder={handleDeleteFolder}
          onNewRequestInFolder={handleNewRequestInFolder}
          onCreateFolder={handleCreateFolder}
          onRenameFile={handleRenameFile}
          onRenameFolder={handleRenameFolder}
          onSelectHistory={handleSelectHistory}
          onImportOpenAPI={handleImportOpenAPI}
          importing={importing}
          activeFilePath={activeTab.filePath ?? undefined}
          isDirty={isTabDirty(activeTab)}
        />

        {/* Sidebar resize handle */}
        <div
          className="w-1 shrink-0 cursor-col-resize hover:bg-[var(--color-accent)] transition-colors"
          onMouseDown={e => {
            e.preventDefault()
            draggingRef.current = 'sidebar'
            document.body.style.cursor = 'col-resize'
            document.body.style.userSelect = 'none'
          }}
        />

        <main className="flex-1 flex flex-col overflow-hidden min-w-0">
          <TabBar
            tabs={tabInfos}
            activeTabId={activeTabId}
            onSelect={handleSelectTab}
            onClose={handleCloseTab}
            onNew={handleNewTab}
          />

          {/* Request header bar */}
          <header className="flex items-center gap-2 px-3 py-1.5 border-b border-[var(--color-border)] bg-[var(--color-card)] shrink-0">
            <div className="flex items-center gap-1.5 min-w-0 flex-1">
              {isTabDirty(activeTab) && (
                <span className="shrink-0 w-2 h-2 rounded-full bg-amber-400" title="Unsaved changes" />
              )}
              {activeTab.name !== null && (
                <span className={cn('text-xs font-mono font-bold shrink-0', METHOD_COLOR[activeTab.method])}>
                  {activeTab.method}
                </span>
              )}
              {activeTab.editingName ? (
                <input
                  autoFocus
                  type="text"
                  value={activeTab.name ?? ''}
                  onChange={e => updateTab(activeTabId, { name: e.target.value })}
                  onBlur={() => {
                    if (!activeTab.name?.trim()) updateTab(activeTabId, { name: null, editingName: false })
                    else updateTab(activeTabId, { editingName: false })
                  }}
                  onKeyDown={e => {
                    if (e.key === 'Enter' || e.key === 'Escape') {
                      if (!activeTab.name?.trim()) updateTab(activeTabId, { name: null, editingName: false })
                      else updateTab(activeTabId, { editingName: false })
                    }
                  }}
                  className={cn(
                    'text-xs font-medium bg-transparent outline-none',
                    'border-b border-[var(--color-accent)]',
                    'text-[var(--color-foreground)] w-40',
                  )}
                />
              ) : activeTab.name !== null ? (
                <button
                  type="button"
                  onClick={() => updateTab(activeTabId, { editingName: true, saveError: null })}
                  title="Click to rename"
                  className="text-xs font-medium text-[var(--color-foreground)] hover:text-[var(--color-accent)] truncate transition-colors text-left"
                >
                  {activeTab.name}
                </button>
              ) : (
                <button
                  type="button"
                  onClick={() => updateTab(activeTabId, { name: '', editingName: true })}
                  title="Click to name this request"
                  className="text-xs text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors"
                >
                  New Request
                </button>
              )}
              {activeTab.saveError && (
                <span className="text-label text-[var(--color-destructive)] truncate ml-1">
                  {activeTab.saveError}
                </span>
              )}
            </div>

            <div className="ml-auto flex items-center gap-2 shrink-0">
              <button
                type="button"
                onClick={handleOpenRequest}
                title="Open request file"
                className="flex items-center gap-1 text-xs text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors"
              >
                <FolderOpen size={13} />
              </button>
              <button
                type="button"
                onClick={handleSaveRequest}
                title="Save request (Ctrl+S)"
                className="flex items-center gap-1 text-xs text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors"
              >
                <Save size={13} />
              </button>
              <button
                type="button"
                onClick={handleNewRequest}
                title="New request tab (Ctrl+N)"
                className="flex items-center gap-1 text-xs text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors"
              >
                <FilePlus size={13} />
              </button>
              <button
                type="button"
                onClick={() => setShowSettings(true)}
                title="Settings"
                className="flex items-center gap-1 text-xs text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors"
              >
                <Settings size={13} />
              </button>

              {environments.length > 0 && (
                <div className="w-px h-4 bg-[var(--color-border)]" />
              )}

              {environments.length > 0 && (
                <select
                  value={activeEnv?.path ?? ''}
                  onChange={e => handleEnvChange(e.target.value)}
                  className={cn(
                    'bg-[var(--color-muted)] text-xs rounded px-2 py-0.5',
                    'border border-[var(--color-border)] outline-none cursor-pointer',
                    activeEnv ? 'text-[var(--color-accent)]' : 'text-[var(--color-muted-foreground)]',
                  )}
                >
                  <option value="">No environment</option>
                  {environments.map(env => (
                    <option key={env.path} value={env.path}>{env.name}</option>
                  ))}
                </select>
              )}

              {workspacePath && (
                <button
                  type="button"
                  onClick={() => { setEditingEnvPath(activeEnv?.path ?? null); setShowEnvEditor(true) }}
                  title={activeEnv ? `Edit environment: ${activeEnv.name}` : 'New environment'}
                  className="flex items-center gap-1 text-xs text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors"
                >
                  <Pencil size={13} />
                </button>
              )}
            </div>
          </header>

          {/* Request + Response split */}
          <div className="flex-1 flex overflow-hidden">
            <div className="flex-1 flex flex-col overflow-hidden min-w-0">
              <RequestEditor
                method={activeTab.method}
                url={activeTab.url}
                loading={activeTab.loading}
                queryParams={activeTab.queryParams}
                headers={activeTab.headers}
                body={activeTab.body}
                auth={activeTab.auth}
                onMethodChange={m => updateTab(activeTabId, { method: m })}
                onUrlChange={u => updateTab(activeTabId, { url: u })}
                onUrlPaste={handleUrlPaste}
                onQueryParamsChange={p => updateTab(activeTabId, { queryParams: p })}
                onHeadersChange={h => updateTab(activeTabId, { headers: h })}
                onBodyChange={b => updateTab(activeTabId, { body: b })}
                onAuthChange={a => updateTab(activeTabId, { auth: a })}
                onSend={handleSend}
              />
            </div>

            {/* Response resize handle */}
            <div
              className="w-1 shrink-0 cursor-col-resize hover:bg-[var(--color-accent)] transition-colors"
              onMouseDown={e => {
                e.preventDefault()
                draggingRef.current = 'response'
                document.body.style.cursor = 'col-resize'
                document.body.style.userSelect = 'none'
              }}
            />

            <div
              className={cn('flex flex-col overflow-hidden min-w-0 shrink-0', responseWidth === 0 && 'w-[45%]')}
              style={responseWidth > 0 ? { width: responseWidth } : {}}
            >
              <ResponseViewer response={activeTab.response} loading={activeTab.loading} />
            </div>
          </div>
        </main>
      </div>
    </div>
  )
}
