// Typed wrapper around Wails-generated bindings.
// Uses plain interfaces for ergonomic object creation at call sites.
// The generated wailsjs classes use 'namespace' (not erasable syntax in TS7),
// so all Wails imports are isolated here.
import { SetHasDirtyTabs as WailsSetHasDirtyTabs } from '@wailsjs/go/main/App'
import {
  Execute as WailsExecute,
  GetHistory as WailsGetHistory,
  CancelRequest as WailsCancelRequest,
  ClearHistory as WailsClearHistory,
} from '@wailsjs/go/main/RequestService'
import {
  OpenWorkspace as WailsOpenWorkspace,
  NewWorkspace as WailsNewWorkspace,
  CloseWorkspace as WailsCloseWorkspace,
  GetWorkspacePath as WailsGetWorkspacePath,
  ListRequests as WailsListRequests,
  LoadRequest as WailsLoadRequest,
  SaveRequest as WailsSaveRequest,
  OpenRequest as WailsOpenRequest,
  ListEnvironments as WailsListEnvironments,
  SelectEnvironment as WailsSelectEnvironment,
  GetActiveEnvironment as WailsGetActiveEnvironment,
  GetEnvironment as WailsGetEnvironment,
  SaveEnvironment as WailsSaveEnvironment,
  DeleteEnvironment as WailsDeleteEnvironment,
  DeleteRequest as WailsDeleteRequest,
  CreateFolder as WailsCreateFolder,
  DeleteFolder as WailsDeleteFolder,
  RenameFolder as WailsRenameFolder,
  RenameRequest as WailsRenameRequest,
  SaveResponseCache as WailsSaveResponseCache,
  LoadResponseCache as WailsLoadResponseCache,
  OpenInSystemViewer as WailsOpenInSystemViewer,
  OpenInExplorer as WailsOpenInExplorer,
  MoveRequest as WailsMoveRequest,
} from '@wailsjs/go/main/WorkspaceService'
import {
  ImportOpenAPISpec as WailsImportOpenAPISpec,
  ImportOpenAPISpecFromPath as WailsImportOpenAPISpecFromPath,
  ImportOpenAPISpecFromURL as WailsImportOpenAPISpecFromURL,
} from '@wailsjs/go/main/OpenAPIService'
import {
  GetSettings as WailsGetSettings,
  SaveSettings as WailsSaveSettings,
  ResetSettings as WailsResetSettings,
} from '@wailsjs/go/main/SettingsService'
import {
  StartDemo as WailsStartDemo,
  StopDemo as WailsStopDemo,
  IsRunning as WailsIsDemoRunning,
  GetPort as WailsGetDemoPort,
} from '@wailsjs/go/main/DemoService'
import {
  ListScenarios as WailsListScenarios,
  LoadScenario as WailsLoadScenario,
  RunScenario as WailsRunScenario,
  LoadScenarioRaw as WailsLoadScenarioRaw,
  SaveScenarioRaw as WailsSaveScenarioRaw,
  CreateScenario as WailsCreateScenario,
  DuplicateScenario as WailsDuplicateScenario,
  DeleteScenario as WailsDeleteScenario,
  RenameScenario as WailsRenameScenario,
  SaveScenarioResult as WailsSaveScenarioResult,
  LoadScenarioResult as WailsLoadScenarioResult,
} from '@wailsjs/go/main/ScenarioService'

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
  filePath?: string
  targetDir?: string
}

// ---- App lifecycle ----

export async function setHasDirtyTabs(dirty: boolean): Promise<void> {
  await WailsSetHasDirtyTabs(dirty)
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

export async function cancelRequest(): Promise<void> {
  await WailsCancelRequest()
}

export async function clearHistory(): Promise<void> {
  await WailsClearHistory()
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

export async function closeWorkspace(): Promise<void> {
  await WailsCloseWorkspace()
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

export interface EnvironmentVariable {
  name: string
  value: string
  secret: boolean
}

export interface EnvironmentIO {
  path: string
  name: string
  variables: EnvironmentVariable[]
}

export async function getEnvironment(path: string): Promise<EnvironmentIO> {
  const result = await WailsGetEnvironment(path)
  return result as unknown as EnvironmentIO
}

export async function saveEnvironment(env: EnvironmentIO): Promise<EnvInfo> {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const result = await WailsSaveEnvironment(env as any)
  return result as unknown as EnvInfo
}

export async function deleteEnvironment(path: string): Promise<void> {
  await WailsDeleteEnvironment(path)
}

export async function deleteRequest(path: string): Promise<void> {
  await WailsDeleteRequest(path)
}

export async function createFolder(parentPath: string, name: string): Promise<string> {
  const result = await WailsCreateFolder(parentPath, name)
  return result as unknown as string
}

export async function deleteFolder(path: string): Promise<void> {
  await WailsDeleteFolder(path)
}

export async function renameFolder(path: string, newName: string): Promise<string> {
  const result = await WailsRenameFolder(path, newName)
  return result as unknown as string
}

export async function renameRequest(path: string, newName: string): Promise<string> {
  const result = await WailsRenameRequest(path, newName)
  return result as unknown as string
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

export async function importOpenAPISpecFromURL(url: string, baseURL = ''): Promise<ImportResult> {
  const result = await WailsImportOpenAPISpecFromURL(url, baseURL)
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

// ---- Demo server ----

export interface DemoResult {
  workspacePath: string
  port: number
  error?: string
}

export async function startDemo(): Promise<DemoResult> {
  const result = await WailsStartDemo()
  return result as unknown as DemoResult
}

export async function stopDemo(): Promise<void> {
  await WailsStopDemo()
}

export async function isDemoRunning(): Promise<boolean> {
  const result = await WailsIsDemoRunning()
  return result as unknown as boolean
}

export async function getDemoPort(): Promise<number> {
  const result = await WailsGetDemoPort()
  return result as unknown as number
}

// ---- Response cache ----

export interface ResponseCacheData extends ResponseData {
  cachedAt: string
}

export async function saveResponseCache(reqPath: string, response: ResponseData): Promise<void> {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  await WailsSaveResponseCache(reqPath, response as any)
}

export async function openInSystemViewer(content: string, format: string): Promise<string> {
  const result = await WailsOpenInSystemViewer(content, format)
  return result as unknown as string
}

export async function openInExplorer(path: string): Promise<string> {
  const result = await WailsOpenInExplorer(path)
  return result as unknown as string
}

export async function moveRequest(srcPath: string, destDir: string): Promise<string> {
  const result = await WailsMoveRequest(srcPath, destDir)
  return result as unknown as string
}

export async function loadResponseCache(reqPath: string): Promise<ResponseCacheData | null> {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const result = await WailsLoadResponseCache(reqPath)
  return result as unknown as ResponseCacheData | null
}

// ---- Scenarios ----

export interface ScenarioInfo {
  name: string
  path: string
  stepCount?: number
}

export interface AssertionDetail {
  type: string
  path?: string
  operator?: string
  expected?: unknown
  schema?: string
}

export interface ExtractionDetail {
  name: string
  source: string
  path?: string
}

export interface StepDetail {
  name: string
  description?: string
  method?: string
  url?: string
  headers?: Array<{ name: string; value: string }>
  queryParams?: Array<{ name: string; value: string }>
  bodyType?: string
  bodyContent?: string
  requestRef?: string
  assertions: AssertionDetail[]
  extractions: ExtractionDetail[]
  failPolicy?: string
  timeoutMs?: number
  retryMaxAttempts?: number
  retryDelayMs?: number
}

export interface ScenarioDetail {
  path: string
  name: string
  description: string
  steps: StepDetail[]
  error?: string
}

export interface AssertionResult {
  type: string
  passed: boolean
  message: string
}

export interface StepResult {
  name: string
  passed: boolean
  skipped: boolean
  statusCode: number
  durationMs: number
  error?: string
  extracted: Record<string, string>
  assertions: AssertionResult[]
}

export interface ScenarioResult {
  name: string
  passed: boolean
  durationMs: number
  steps: StepResult[]
  error?: string
}

export async function listScenarios(): Promise<ScenarioInfo[]> {
  const result = await WailsListScenarios()
  return (result ?? []) as unknown as ScenarioInfo[]
}

export async function loadScenario(path: string): Promise<ScenarioDetail> {
  const result = await WailsLoadScenario(path)
  return result as unknown as ScenarioDetail
}

export async function runScenario(path: string, envPath = ''): Promise<ScenarioResult> {
  const result = await WailsRunScenario(path, envPath)
  return result as unknown as ScenarioResult
}

export async function loadScenarioRaw(path: string): Promise<string> {
  return (await WailsLoadScenarioRaw(path)) as unknown as string
}

export async function saveScenarioRaw(path: string, content: string): Promise<string> {
  return (await WailsSaveScenarioRaw(path, content)) as unknown as string
}

export interface CreateScenarioResult {
  path: string
  error?: string
}

export async function createScenario(name: string): Promise<CreateScenarioResult> {
  const result = await WailsCreateScenario(name)
  return result as unknown as CreateScenarioResult
}

// Returns "" on success or an error message.
export async function saveScenarioResult(scenarioPath: string, result: ScenarioResult): Promise<void> {
  await WailsSaveScenarioResult(scenarioPath, result)
}

// Returns null if no cached result exists.
export async function loadScenarioResult(scenarioPath: string): Promise<ScenarioResult | null> {
  const r = await WailsLoadScenarioResult(scenarioPath)
  return r as unknown as ScenarioResult | null
}

export async function duplicateScenario(path: string): Promise<CreateScenarioResult> {
  const result = await WailsDuplicateScenario(path)
  return result as unknown as CreateScenarioResult
}

// Returns "" on success, or an error message.
export async function deleteScenario(path: string): Promise<string> {
  return (await WailsDeleteScenario(path)) as unknown as string
}

// Returns the new file path on success, or a string starting with "error:" on failure.
export async function renameScenario(path: string, newName: string): Promise<string> {
  return (await WailsRenameScenario(path, newName)) as unknown as string
}
