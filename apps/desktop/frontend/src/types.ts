import type { FolderRunItem } from '@/lib/bridge'

export type FileClipboard = {
  op: 'copy' | 'cut'
  kind: 'request' | 'scenario'
  srcPath: string
}

export interface RunAllEntry {
  path: string
  name: string
  passed: boolean
  durationMs: number
  error?: string
}

export interface RunAllState {
  running: boolean
  done: number
  total: number
  passed: number
  results: RunAllEntry[]
}

export interface FolderRunState {
  folderPath: string
  folderName: string
  running: boolean
  results: FolderRunItem[]
}

export type KVPair = {
  id: string
  name: string
  value: string
  enabled: boolean
}

export type BodyType = 'none' | 'json' | 'raw' | 'form-urlencoded' | 'multipart'

export type BodyState = {
  type: BodyType
  content: string
}

export type AuthType = 'none' | 'bearer' | 'basic' | 'apikey'

export type AuthState =
  | { type: 'none' }
  | { type: 'bearer'; token: string }
  | { type: 'basic'; username: string; password: string }
  | { type: 'apikey'; key: string; value: string; location: 'header' | 'query' }
