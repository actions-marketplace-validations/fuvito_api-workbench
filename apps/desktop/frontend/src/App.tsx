import { useState, useEffect, useCallback } from 'react'
import { FolderOpen, Save, FilePlus } from 'lucide-react'
import {
  executeRequest, getHistory, openWorkspace, newWorkspace, getWorkspacePath, listRequests,
  loadRequest, saveRequest, openRequest,
  listEnvironments, selectEnvironment, getActiveEnvironment,
  type ResponseData, type AuthPayload, type HistoryEntry, type FileNode,
  type EnvInfo, type RequestFileIO,
} from '@/lib/bridge'
import { cn } from '@/lib/utils'
import { KeyValueEditor } from '@/components/KeyValueEditor'
import { BodyEditor } from '@/components/BodyEditor'
import { AuthEditor } from '@/components/AuthEditor'
import { WorkspacePanel } from '@/components/WorkspacePanel'
import type { KVPair, BodyState, AuthState } from '@/types'

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
  tab: SidebarTab
  onTabChange: (t: SidebarTab) => void
  workspacePath: string
  fileTree: FileNode[]
  history: HistoryEntry[]
  onOpenWorkspace: () => void
  onNewWorkspace: () => void
  onSelectFile: (path: string) => void
  onSelectHistory: (entry: HistoryEntry) => void
}

function Sidebar({
  tab, onTabChange,
  workspacePath, fileTree, history,
  onOpenWorkspace, onNewWorkspace,
  onSelectFile, onSelectHistory,
}: SidebarProps) {
  return (
    <aside className="w-56 shrink-0 flex flex-col border-r border-[var(--color-border)] bg-[var(--color-card)]">
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
  onQueryParamsChange: (p: KVPair[]) => void
  onHeadersChange: (h: KVPair[]) => void
  onBodyChange: (b: BodyState) => void
  onAuthChange: (a: AuthState) => void
  onSend: () => void
}

function RequestEditor({
  method, url, loading,
  queryParams, headers, body, auth,
  onMethodChange, onUrlChange,
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
      <div className="flex items-center gap-2 px-3 py-2 border-b border-[var(--color-border)]">
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
          <pre className="text-xs font-mono text-[var(--color-foreground)] whitespace-pre-wrap break-all leading-relaxed">
            {prettyBody(response.body)}
          </pre>
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

// ---- App ----

export default function App() {
  // Request state
  const [method, setMethod] = useState<HttpMethod>('GET')
  const [url, setUrl] = useState<string>('')
  const [loading, setLoading] = useState<boolean>(false)
  const [response, setResponse] = useState<ResponseData | null>(null)
  const [queryParams, setQueryParams] = useState<KVPair[]>([])
  const [headers, setHeaders] = useState<KVPair[]>([])
  const [body, setBody] = useState<BodyState>({ type: 'none', content: '' })
  const [auth, setAuth] = useState<AuthState>({ type: 'none' })

  // Workspace state
  const [workspacePath, setWorkspacePath] = useState<string>('')
  const [fileTree, setFileTree] = useState<FileNode[]>([])
  const [environments, setEnvironments] = useState<EnvInfo[]>([])
  const [activeEnv, setActiveEnv] = useState<EnvInfo | null>(null)
  const [sidebarTab, setSidebarTab] = useState<SidebarTab>('workspace')

  // History state
  const [history, setHistory] = useState<HistoryEntry[]>([])

  // ---- init ----

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
  }, [refreshWorkspace, refreshHistory])

  // ---- workspace actions ----

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
    try {
      const r = await loadRequest(path)
      if (!r) return
      const state = requestFileIOToState(r)
      setMethod(state.method)
      setUrl(state.url)
      setHeaders(state.headers)
      setQueryParams(state.queryParams)
      setBody(state.body)
      setAuth(state.auth)
      setResponse(null)
    } catch (err) {
      console.error('load request:', err)
    }
  }

  async function handleOpenRequest() {
    try {
      const r = await openRequest()
      if (!r) return
      const state = requestFileIOToState(r)
      setMethod(state.method)
      setUrl(state.url)
      setHeaders(state.headers)
      setQueryParams(state.queryParams)
      setBody(state.body)
      setAuth(state.auth)
      setResponse(null)
    } catch (err) {
      console.error('open request:', err)
    }
  }

  async function handleSaveRequest() {
    try {
      const req: RequestFileIO = {
        schemaVersion: '1',
        name: url.trim().split('/').pop() ?? 'request',
        method,
        url: url.trim(),
        headers: headers.filter(h => h.enabled && h.name.trim()).map(h => ({ name: h.name, value: h.value })),
        queryParams: queryParams.filter(p => p.enabled && p.name.trim()).map(p => ({ name: p.name, value: p.value })),
        body: body.type === 'none' ? undefined : { type: body.type, content: body.content },
        auth: authStateToPayload(auth) ?? undefined,
      }
      const path = await saveRequest(req)
      if (path) await refreshWorkspace()
    } catch (err) {
      console.error('save request:', err)
    }
  }

  // ---- environment actions ----

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

  // ---- history actions ----

  function handleSelectHistory(entry: HistoryEntry) {
    if ((HTTP_METHODS as readonly string[]).includes(entry.method)) {
      setMethod(entry.method as HttpMethod)
    }
    setUrl(entry.url)
  }

  // ---- request execution ----

  async function handleSend() {
    if (!url.trim() || loading) return
    setLoading(true)
    setResponse(null)
    try {
      const result = await executeRequest({
        method,
        url: url.trim(),
        headers: headers.filter(h => h.enabled && h.name.trim()).map(h => ({ name: h.name, value: h.value })),
        queryParams: queryParams.filter(p => p.enabled && p.name.trim()).map(p => ({ name: p.name, value: p.value })),
        body: body.type === 'none' ? undefined : { type: body.type, content: body.content },
        auth: authStateToPayload(auth),
        timeoutMs: 0,
      })
      setResponse(result)
    } catch (err) {
      setResponse({ statusCode: 0, status: '', headers: [], body: '', durationMs: 0, size: 0, error: String(err) })
    } finally {
      setLoading(false)
      refreshHistory()
    }
  }

  // ---- render ----

  return (
    <div className="flex h-screen w-screen overflow-hidden">
      <Sidebar
        tab={sidebarTab}
        onTabChange={setSidebarTab}
        workspacePath={workspacePath}
        fileTree={fileTree}
        history={history}
        onOpenWorkspace={handleOpenWorkspace}
        onNewWorkspace={handleNewWorkspace}
        onSelectFile={handleSelectFile}
        onSelectHistory={handleSelectHistory}
      />

      <main className="flex-1 flex flex-col overflow-hidden">
        {/* Top bar */}
        <header className="flex items-center gap-2 px-3 py-1.5 border-b border-[var(--color-border)] bg-[var(--color-card)] shrink-0">
          <span className="text-xs font-semibold text-[var(--color-muted-foreground)] tracking-wide">
            EndpointDesk
          </span>

          <div className="ml-auto flex items-center gap-2">
            {/* Toolbar buttons */}
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
              title="Save request"
              className="flex items-center gap-1 text-xs text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors"
            >
              <Save size={13} />
            </button>
            <button
              type="button"
              onClick={() => { setMethod('GET'); setUrl(''); setQueryParams([]); setHeaders([]); setBody({ type: 'none', content: '' }); setAuth({ type: 'none' }); setResponse(null) }}
              title="New request"
              className="flex items-center gap-1 text-xs text-[var(--color-muted-foreground)] hover:text-[var(--color-foreground)] transition-colors"
            >
              <FilePlus size={13} />
            </button>

            {/* Separator */}
            {environments.length > 0 && (
              <div className="w-px h-4 bg-[var(--color-border)]" />
            )}

            {/* Environment selector */}
            {environments.length > 0 && (
              <select
                value={activeEnv?.path ?? ''}
                onChange={e => handleEnvChange(e.target.value)}
                className={cn(
                  'bg-[var(--color-muted)] text-xs rounded px-2 py-0.5',
                  'border border-[var(--color-border)] outline-none cursor-pointer',
                  'text-[var(--color-foreground)]',
                  activeEnv ? 'text-[var(--color-accent)]' : 'text-[var(--color-muted-foreground)]',
                )}
              >
                <option value="">No environment</option>
                {environments.map(env => (
                  <option key={env.path} value={env.path}>{env.name}</option>
                ))}
              </select>
            )}
          </div>
        </header>

        {/* Request + Response split */}
        <div className="flex-1 flex overflow-hidden">
          <div className="flex-1 flex flex-col overflow-hidden min-w-0">
            <RequestEditor
              method={method}
              url={url}
              loading={loading}
              queryParams={queryParams}
              headers={headers}
              body={body}
              auth={auth}
              onMethodChange={setMethod}
              onUrlChange={setUrl}
              onQueryParamsChange={setQueryParams}
              onHeadersChange={setHeaders}
              onBodyChange={setBody}
              onAuthChange={setAuth}
              onSend={handleSend}
            />
          </div>
          <div className="w-[45%] flex flex-col overflow-hidden min-w-0">
            <ResponseViewer response={response} loading={loading} />
          </div>
        </div>
      </main>
    </div>
  )
}
