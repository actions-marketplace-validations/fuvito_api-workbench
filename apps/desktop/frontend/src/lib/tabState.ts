import type { KVPair, BodyState, AuthState } from '@/types'
import type { ResponseData } from '@/lib/bridge'
import type { HttpMethod } from './httpUtils'

export type TabState = {
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
  responseCachedAt: string | null
  loading: boolean
  saveError: string | null
  editingName: boolean
}

export function computeStateKey(
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

export function createBlankTab(targetDir: string | null = null): TabState {
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
    responseCachedAt: null,
    loading: false,
    saveError: null,
    editingName: false,
  }
}

export function isTabDirty(tab: TabState): boolean {
  return computeStateKey(tab.method, tab.url, tab.queryParams, tab.headers, tab.body, tab.auth, tab.name) !== tab.savedKey
}

export function isBlankTab(tab: TabState): boolean {
  return tab.url === '' && tab.name === null && tab.filePath === null && !isTabDirty(tab)
}
