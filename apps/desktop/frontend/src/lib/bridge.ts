// Typed wrapper around Wails-generated bindings.
// Uses plain interfaces for ergonomic object creation at call sites.
// The generated wailsjs classes use 'namespace' (not erasable syntax in TS7),
// so all Wails imports are isolated here.
import { Execute as WailsExecute, GetHistory as WailsGetHistory } from '@wailsjs/go/main/RequestService'
import {
  OpenWorkspace as WailsOpenWorkspace,
  NewWorkspace as WailsNewWorkspace,
  GetWorkspacePath as WailsGetWorkspacePath,
  ListRequests as WailsListRequests,
  LoadRequest as WailsLoadRequest,
  SaveRequest as WailsSaveRequest,
  OpenRequest as WailsOpenRequest,
  ListEnvironments as WailsListEnvironments,
  SelectEnvironment as WailsSelectEnvironment,
  GetActiveEnvironment as WailsGetActiveEnvironment,
} from '@wailsjs/go/main/WorkspaceService'
import {
  ImportOpenAPISpec as WailsImportOpenAPISpec,
  ImportOpenAPISpecFromPath as WailsImportOpenAPISpecFromPath,
} from '@wailsjs/go/main/OpenAPIService'
import {
  GetSettings as WailsGetSettings,
  SaveSettings as WailsSaveSettings,
  ResetSettings as WailsResetSettings,
} from '@wailsjs/go/main/SettingsService'

// ---- Shared I/O types ----

export interface AuthPayload {
  type: string
  token: string
  username: string
  password: string
  key: string
  value: string
  location: string
}

export interface RequestPayload {
  method: string
  url: string
  headers: Array<{ name: string; value: string }>
  queryParams: Array<{ name: string; value: string }>
  body?: { type: string; content: string }
  auth?: AuthPayload
  timeoutMs: number
}

export interface TimingData {
  dnsMs: number
  connectMs: number
  tlsMs: number
  ttfbMs: number
  transferMs: number
}

export interface ResponseData {
  statusCode: number
  status: string
  headers: Array<{ name: string; value: string }>
  body: string
  durationMs: number
  size: number
  timing: TimingData
  error?: string
}

export interface AppSettings {
  theme: string
  fontSize: number
  defaultTimeoutMs: number
  proxyUrl: string
  tlsSkipVerify: boolean
}

export interface HistoryEntry {
  id: number
  executedAt: string
  method: string
  url: string
  statusCode: number
  statusText: string
  durationMs: number
  sizeBytes: number
}

export interface FileNode {
  name: string
  path: string
  isDir: boolean
  children?: FileNode[]
}

export interface EnvInfo {
  name: string
  path: string
}

export interface RequestFileIO {
  schemaVersion: string
  name: string
  description?: string
  method: string
  url: string
  headers?: Array<{ name: string; value: string }>
  queryParams?: Array<{ name: string; value: string }>
  body?: { type: string; content: string }
  auth?: AuthPayload
  timeoutMs?: number
}

// ---- Request execution ----

export async function executeRequest(payload: RequestPayload): Promise<ResponseData> {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const result = await WailsExecute(payload as any)
  return result as unknown as ResponseData
}

// ---- History ----

export async function getHistory(): Promise<HistoryEntry[]> {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const result = await WailsGetHistory()
  return (result ?? []) as unknown as HistoryEntry[]
}

// ---- Workspace ----

export async function openWorkspace(): Promise<string> {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const result = await WailsOpenWorkspace()
  return result as unknown as string
}

export async function newWorkspace(): Promise<string> {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const result = await WailsNewWorkspace()
  return result as unknown as string
}

export async function getWorkspacePath(): Promise<string> {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const result = await WailsGetWorkspacePath()
  return result as unknown as string
}

export async function listRequests(): Promise<FileNode[]> {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const result = await WailsListRequests()
  return (result ?? []) as unknown as FileNode[]
}

export async function loadRequest(path: string): Promise<RequestFileIO> {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const result = await WailsLoadRequest(path)
  return result as unknown as RequestFileIO
}

export async function saveRequest(req: RequestFileIO): Promise<string> {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const result = await WailsSaveRequest(req as any)
  return result as unknown as string
}

export async function openRequest(): Promise<RequestFileIO | null> {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const result = await WailsOpenRequest()
  return result as unknown as RequestFileIO | null
}

// ---- Environments ----

export async function listEnvironments(): Promise<EnvInfo[]> {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const result = await WailsListEnvironments()
  return (result ?? []) as unknown as EnvInfo[]
}

export async function selectEnvironment(path: string): Promise<void> {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  await WailsSelectEnvironment(path)
}

export async function getActiveEnvironment(): Promise<EnvInfo | null> {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const result = await WailsGetActiveEnvironment()
  return result as unknown as EnvInfo | null
}

// ---- OpenAPI import ----

export interface ImportedRequest {
  operationId: string
  summary: string
  method: string
  url: string
  savedPath: string
  tags?: string[]
}

export interface ImportResult {
  specTitle: string
  specVersion: string
  imported: ImportedRequest[]
  error?: string
}

export async function importOpenAPISpec(baseURL = ''): Promise<ImportResult> {
  const result = await WailsImportOpenAPISpec(baseURL)
  return result as unknown as ImportResult
}

export async function importOpenAPISpecFromPath(path: string, baseURL = ''): Promise<ImportResult> {
  const result = await WailsImportOpenAPISpecFromPath(path, baseURL)
  return result as unknown as ImportResult
}

// ---- Settings ----

export async function getSettings(): Promise<AppSettings> {
  const result = await WailsGetSettings()
  return result as unknown as AppSettings
}

export async function saveSettings(s: AppSettings): Promise<AppSettings> {
  const result = await WailsSaveSettings(s as unknown as any) // eslint-disable-line @typescript-eslint/no-explicit-any
  return result as unknown as AppSettings
}

export async function resetSettings(): Promise<AppSettings> {
  const result = await WailsResetSettings()
  return result as unknown as AppSettings
}
