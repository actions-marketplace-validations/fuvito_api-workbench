import type { KVPair, BodyState, AuthState } from '@/types'
import type { ResponseData, ScenarioResult } from '@/lib/bridge'
import type { HttpMethod } from './httpUtils'

export type TabState = {
  id: string
  kind?: 'request' | 'scenario' | 'coverage' | 'runner' | 'cli' | 'conformance' | 'workspace'
  // request tab fields
  method: HttpMethod
  url: string
  queryParams: KVPair[]
  headers: KVPair[]
  body: BodyState
  auth: AuthState
  timeoutMs: number
  noFollowRedirects?: boolean // true returns a 3xx response instead of following it
  name: string | null
  filePath: string | null
  targetDir: string | null
  savedKey: string
  response: ResponseData | null
  previousResponse: ResponseData | null
  responseCachedAt: string | null
  loading: boolean
  saveError: string | null
  editingName: boolean
  // scenario tab fields
  scenarioPath?: string | null
  scenarioName?: string | null
  scenarioResult?: ScenarioResult | null
  scenarioRunning?: boolean
  scenarioDirty?: boolean
  scenarioSaveSignal?: number
  // coverage tab fields
  coverageSpecFile?: string | null
}

export function computeStateKey(
  method: string,
  url: string,
  queryParams: KVPair[],
  headers: KVPair[],
  body: BodyState,
  auth: AuthState,
  name: string | null,
  timeoutMs = 0,
  noFollowRedirects = false,
): string {
  return JSON.stringify({
    method,
    url: url.trim(),
    name,
    timeoutMs,
    // only written when set, so keys of ordinary requests stay as they were
    ...(noFollowRedirects ? { noFollowRedirects } : {}),
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

export function createBlankTab(targetDir: string | null = null): TabState {
  const savedKey = computeStateKey('GET', '', [], [], { type: 'none', content: '' }, { type: 'none' }, null, 0)
  return {
    id: crypto.randomUUID(),
    method: 'GET',
    url: '',
    queryParams: [],
    headers: [],
    body: { type: 'none', content: '' },
    auth: { type: 'none' },
    timeoutMs: 0,
    name: null,
    filePath: null,
    targetDir,
    savedKey,
    response: null,
    previousResponse: null,
    responseCachedAt: null,
    loading: false,
    saveError: null,
    editingName: false,
  }
}

// Page tabs show an app screen (not a request or scenario file), so they are never dirty or blank.
export function isPageTab(kind: TabState['kind']): boolean {
  return kind === 'coverage' || kind === 'runner' || kind === 'cli' || kind === 'conformance' || kind === 'workspace'
}

type TabsStateTuple = { tabs: TabState[]; activeTabId: string }

// Drops the tabs that belong to a temporary workspace or its local server, for when that workspace
// is closed. Keeps one blank tab when nothing is left, and the active tab when it survives.
export function withoutTabsIn(prev: TabsStateTuple, urlPrefix: string | null, workspacePath: string): TabsStateTuple {
  const normWs = workspacePath ? workspacePath.replace(/\\/g, '/') : null
  const inside = (t: TabState): boolean => {
    if (urlPrefix && t.url.startsWith(urlPrefix)) return true
    const fp = (t.filePath ?? '').replace(/\\/g, '/')
    if (normWs && fp.startsWith(normWs)) return true
    const sp = (t.scenarioPath ?? '').replace(/\\/g, '/')
    return !!normWs && sp.startsWith(normWs)
  }
  const remaining = prev.tabs.filter(t => !inside(t))
  if (remaining.length === prev.tabs.length) return prev
  if (remaining.length === 0) {
    const blank = createBlankTab()
    return { tabs: [blank], activeTabId: blank.id }
  }
  const activeStillOpen = remaining.some(t => t.id === prev.activeTabId)
  return { tabs: remaining, activeTabId: activeStillOpen ? prev.activeTabId : remaining[remaining.length - 1].id }
}

export function isTabDirty(tab: TabState): boolean {
  if (tab.kind === 'scenario') return tab.scenarioDirty === true
  if (isPageTab(tab.kind)) return false
  return computeStateKey(tab.method, tab.url, tab.queryParams, tab.headers, tab.body, tab.auth, tab.name, tab.timeoutMs, tab.noFollowRedirects) !== tab.savedKey
}

export function isBlankTab(tab: TabState): boolean {
  if (tab.kind === 'scenario' || isPageTab(tab.kind)) return false
  return tab.url === '' && tab.name === null && tab.filePath === null && !isTabDirty(tab)
}
