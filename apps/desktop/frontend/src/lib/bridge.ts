// Typed wrapper around Wails-generated bindings.
// Uses plain interfaces for ergonomic object creation at call sites.
// The generated wailsjs classes use 'namespace' (not erasable syntax in TS7),
// so all Wails imports are isolated here.
import type { DynamicFunction } from '@/lib/dynamicFunctions'
import { SetHasDirtyTabs as WailsSetHasDirtyTabs, Exit as WailsExit } from '@wailsjs/go/main/App'
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
  GetEnvironmentRaw as WailsGetEnvironmentRaw,
  SaveEnvironmentRaw as WailsSaveEnvironmentRaw,
  ValidateEnvironmentYAML as WailsValidateEnvironmentYAML,
  DeleteRequest as WailsDeleteRequest,
  CreateFolder as WailsCreateFolder,
  DeleteFolder as WailsDeleteFolder,
  RenameFolder as WailsRenameFolder,
  RenameRequest as WailsRenameRequest,
  SaveResponseCache as WailsSaveResponseCache,
  LoadResponseCache as WailsLoadResponseCache,
  ClearResponseCache as WailsClearResponseCache,
  OpenInSystemViewer as WailsOpenInSystemViewer,
  OpenInExplorer as WailsOpenInExplorer,
  MoveRequest as WailsMoveRequest,
  CopyRequest as WailsCopyRequest,
  GetRecentWorkspaces as WailsGetRecentWorkspaces,
  OpenWorkspaceByPath as WailsOpenWorkspaceByPath,
  SaveContentToFile as WailsSaveContentToFile,
  PickFile as WailsPickFile,
  ExportWorkspace as WailsExportWorkspace,
  WorkspaceHasSecrets as WailsWorkspaceHasSecrets,
  ScanWorkspaceImport as WailsScanWorkspaceImport,
  ApplyWorkspaceImport as WailsApplyWorkspaceImport,
} from '@wailsjs/go/main/WorkspaceService'
import {
  ImportOpenAPISpec as WailsImportOpenAPISpec,
  ImportOpenAPISpecFromPath as WailsImportOpenAPISpecFromPath,
  ImportOpenAPISpecFromURL as WailsImportOpenAPISpecFromURL,
} from '@wailsjs/go/main/OpenAPIService'
import {
  ImportPostmanCollection as WailsImportPostmanCollection,
  ImportInsomniaExport as WailsImportInsomniaExport,
} from '@wailsjs/go/main/ImportService'
import {
  PickBundleFile as WailsPickBundleFile,
  PreviewBundle as WailsPreviewBundle,
  ImportBundle as WailsImportBundle,
  ExportBundle as WailsExportBundle,
} from '@wailsjs/go/main/BundleService'
import {
  GetCliCatalog as WailsGetCliCatalog,
  GetCliContext as WailsGetCliContext,
  BuildCliCommand as WailsBuildCliCommand,
  PickCliFile as WailsPickCliFile,
  PickCliFolder as WailsPickCliFolder,
} from '@wailsjs/go/main/CliBuilderService'
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
  OpenSpecInBrowser as WailsOpenDemoSpec,
} from '@wailsjs/go/main/DemoService'
import {
  GetConformContext as WailsGetConformContext,
  InspectConformSpec as WailsInspectConformSpec,
  RunConformance as WailsRunConformance,
  CancelConformance as WailsCancelConformance,
} from '@wailsjs/go/main/ConformanceService'
import {
  Start as WailsStartConformanceServer,
  Stop as WailsStopConformanceServer,
  OpenSpecInBrowser as WailsOpenConformanceSpec,
} from '@wailsjs/go/main/ConformanceServerService'
import {
  ListScenarios as WailsListScenarios,
  LoadScenario as WailsLoadScenario,
  RunScenario as WailsRunScenario,
  RunFolder as WailsRunFolder,
  LoadScenarioRaw as WailsLoadScenarioRaw,
  SaveScenarioRaw as WailsSaveScenarioRaw,
  ValidateScenarioYAML as WailsValidateScenarioYAML,
  CreateScenario as WailsCreateScenario,
  DuplicateScenario as WailsDuplicateScenario,
  DeleteScenario as WailsDeleteScenario,
  RenameScenario as WailsRenameScenario,
  CreateScenarioFolder as WailsCreateScenarioFolder,
  RenameScenarioFolder as WailsRenameScenarioFolder,
  DeleteScenarioFolder as WailsDeleteScenarioFolder,
  MoveScenario as WailsMoveScenario,
  CopyScenario as WailsCopyScenario,
  SaveScenarioResult as WailsSaveScenarioResult,
  LoadScenarioResult as WailsLoadScenarioResult,
  ListTemplates as WailsListTemplates,
  CreateFromTemplate as WailsCreateFromTemplate,
  GetAccumulatedCoverage as WailsGetAccumulatedCoverage,
  ResetAccumulatedCoverage as WailsResetAccumulatedCoverage,
  ListAccumulatedCoverage as WailsListAccumulatedCoverage,
  ExportCoverageReport as WailsExportCoverageReport,
  ExportRunReport as WailsExportRunReport,
  ListDynamicFunctions as WailsListDynamicFunctions,
  GetScenarioRunHistory as WailsGetScenarioRunHistory,
  CancelScenario as WailsCancelScenario,
} from '@wailsjs/go/main/ScenarioService'
import {
  GetUserGuide as WailsGetUserGuide,
  GetFormatSpec as WailsGetFormatSpec,
  GetExampleScenario as WailsGetExampleScenario,
  GetAIPrompt as WailsGetAIPrompt,
  GetAIPromptInstructions as WailsGetAIPromptInstructions,
} from '@wailsjs/go/main/HelpService'

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
  noFollowRedirects?: boolean
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
  // the request as it was sent, credentials masked by the Go core; not stored in the response cache
  requestMethod?: string
  requestUrl?: string
  requestHeaders?: Array<{ name: string; value: string }>
  requestBody?: string
}

export interface AppSettings {
  theme: 'dark' | 'light'
  fontSize: number
  defaultTimeoutMs: number
  proxyUrl: string
  tlsSkipVerify: boolean
  telemetryEnabled: boolean
  // mutual TLS: file paths only (may contain ${NAME}); never the certificate contents
  clientCertFile?: string
  clientKeyFile?: string
  caFile?: string
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
  headers: { name: string; value: string }[]
  bodyType: string
  bodyContent: string
  auth?: AuthPayload
}

export interface FileNode {
  name: string
  path: string
  isDir: boolean
  method?: string
  children?: FileNode[]
}

export interface EnvInfo {
  name: string
  path: string
}

export interface RequestFileIO {
  schemaVersion: string
  title: string
  method: string
  url: string
  headers?: Array<{ name: string; value: string }>
  queryParams?: Array<{ name: string; value: string }>
  body?: { type: string; content: string }
  auth?: AuthPayload
  timeoutMs?: number
  noFollowRedirects?: boolean
  filePath?: string
  targetDir?: string
}

// ---- App lifecycle ----

export async function setHasDirtyTabs(dirty: boolean): Promise<void> {
  await WailsSetHasDirtyTabs(dirty)
}

export async function quitApp(): Promise<void> {
  await WailsExit()
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

export async function getRecentWorkspaces(): Promise<string[]> {
  const result = await WailsGetRecentWorkspaces()
  return result as unknown as string[]
}

export async function openWorkspaceByPath(path: string): Promise<string> {
  const result = await WailsOpenWorkspaceByPath(path)
  return result as unknown as string
}

export async function saveContentToFile(content: string, defaultName: string): Promise<string> {
  const result = await WailsSaveContentToFile(content, defaultName)
  return result as unknown as string
}

export async function pickFile(title = 'Select file'): Promise<string> {
  const result = await WailsPickFile(title)
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

export async function getEnvironmentRaw(path: string): Promise<string> {
  const result = await WailsGetEnvironmentRaw(path)
  return result as unknown as string
}

export async function saveEnvironmentRaw(path: string, content: string): Promise<EnvInfo> {
  const result = await WailsSaveEnvironmentRaw(path, content)
  return result as unknown as EnvInfo
}

export async function validateEnvironmentYAML(content: string): Promise<string> {
  return (await WailsValidateEnvironmentYAML(content)) as unknown as string
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

// ---- Collection import (Postman / Insomnia) ----

export interface CollectionImportResult {
  collectionName: string
  imported: ImportedRequest[]
  skipped: number
  error?: string
}

export async function importPostmanCollection(): Promise<CollectionImportResult> {
  const result = await WailsImportPostmanCollection()
  return result as unknown as CollectionImportResult
}

export async function importInsomniaExport(): Promise<CollectionImportResult> {
  const result = await WailsImportInsomniaExport()
  return result as unknown as CollectionImportResult
}

// ---- Bundle import (environments + requests + scenarios in one document) ----

export type BundleStrategy = 'skip' | 'overwrite' | 'rename'

export interface BundleFile {
  path: string   // empty = the user cancelled
  text: string
  error?: string
}

export interface BundleIssue {
  severity: 'error' | 'warning'
  where: string
  message: string
}

export interface BundleEntry {
  kind: 'environment' | 'request' | 'scenario'
  name: string
  path: string           // where it will be written, relative to the workspace
  requestedPath: string  // where the bundle asked for it
  action: 'create' | 'overwrite' | 'rename' | 'skip'
}

export interface BundlePreview {
  title: string
  specFile: string       // the bundle's own default spec
  issues: BundleIssue[]
  entries: BundleEntry[]
  hasErrors: boolean
  parseError?: string
}

export interface BundleImportResult {
  created: number
  overwritten: number
  renamed: number
  skipped: number
  written: string[]
  error?: string
}

export interface BundleExportResult {
  path: string            // empty with no error = the user cancelled
  environments: number
  requests: number
  scenarios: number
  redactedSecrets: number
  skipped: string[]       // files that could not be exported, with the reason
  warnings: string[]
  error?: string
}

// Saves the open workspace as one bundle file chosen in a save dialog. Secret values are redacted.
export async function exportBundle(): Promise<BundleExportResult> {
  return (await WailsExportBundle()) as unknown as BundleExportResult
}

export async function pickBundleFile(): Promise<BundleFile> {
  return (await WailsPickBundleFile()) as unknown as BundleFile
}

export async function previewBundle(text: string, specFile: string, strategy: BundleStrategy): Promise<BundlePreview> {
  return (await WailsPreviewBundle(text, specFile, strategy)) as unknown as BundlePreview
}

export async function importBundle(text: string, specFile: string, strategy: BundleStrategy): Promise<BundleImportResult> {
  return (await WailsImportBundle(text, specFile, strategy)) as unknown as BundleImportResult
}

// ---- CLI Command Builder ----

export type CliInputKind = 'file' | 'fileOrURL' | 'dir' | 'path' | 'outputFile' | 'url' | 'enum' | 'int' | 'bool' | 'string'

export interface CliInput {
  name: string
  short: string
  positional: boolean
  required: boolean
  kind: CliInputKind
  role: '' | 'scenario' | 'environment' | 'spec' | 'workspace'
  accepts: string[]
  enum: string[]
  default: string
  label: string
  help: string
  example: string
  group: 'essential' | 'advanced'
}

export interface CliExitCode {
  code: number
  meaning: string
  forCI: string
}

export interface CliCommand {
  name: string
  goal: string
  summary: string
  when: string
  inputs: CliInput[]
  outputs: string[]
  exitCodes: CliExitCode[]
}

export interface CliFile {
  path: string
  label: string
}

export interface CliContext {
  workspace: string
  runFrom: string
  scenarios: CliFile[]
  scenarioFolders: CliFile[]
  environments: CliFile[]
  specs: CliFile[]
}

export interface CliBuildInput {
  command: string
  values: Record<string, string>
  runFrom: string
}

export interface CliMessage {
  field: string
  severity: 'error' | 'warning' | 'info'
  text: string
}

export interface CliNeed {
  name: string
  secret: boolean
  from: string
}

export interface CliSuggestion {
  field: string
  value: string
  why: string
}

export type CliShell = 'bash' | 'powershell' | 'cmd'

export interface CliRendering {
  shell: CliShell
  multiline: string
  oneline: string
}

export interface CliBuildOutput {
  command: string
  valid: boolean
  argv: string[]
  renderings: CliRendering[]
  messages: CliMessage[]
  needs: CliNeed[]
  suggestions: CliSuggestion[]
  explanation: {
    goal: string
    summary: string
    when: string
    parts: { token: string; meaning: string }[]
    outputs: string[]
    exitCodes: CliExitCode[]
  }
}

export async function getCliCatalog(): Promise<CliCommand[]> {
  return (await WailsGetCliCatalog()) as unknown as CliCommand[]
}

export async function getCliContext(): Promise<CliContext> {
  return (await WailsGetCliContext()) as unknown as CliContext
}

export async function buildCliCommand(input: CliBuildInput): Promise<CliBuildOutput> {
  return (await WailsBuildCliCommand(input as never)) as unknown as CliBuildOutput
}

export async function pickCliFile(title: string): Promise<string> {
  return WailsPickCliFile(title)
}

export async function pickCliFolder(title: string): Promise<string> {
  return WailsPickCliFolder(title)
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

export async function openDemoSpec(): Promise<string> {
  const result = await WailsOpenDemoSpec()
  return result as unknown as string
}

// ---- Check conformance ----

export interface ConformContext {
  workspace: string
  specs: CliFile[]
  environments: CliFile[]
}

export interface ConformSpecInfo {
  title: string
  version: string
  serverUrl: string
  reads: number
  writes: number
  error?: string
}

export interface ConformRunInput {
  specPath: string
  baseUrl: string
  envPath: string
  allowWrites: boolean
  confirmedRemote: boolean
}

export interface ConformOperation {
  operationId: string
  method: string
  path: string
  status: 'passed' | 'failed' | 'error' | 'skipped'
  statusCode: number
  durationMs: number
  violations: string[]
  error?: string
}

export interface ConformRunOutput {
  error?: string
  specTitle: string
  baseUrl: string
  writes: boolean
  durationMs: number
  operations: ConformOperation[]
  total: number
  covered: number
  notCovered: number
  passing: number
  failing: number
}

export async function getConformContext(): Promise<ConformContext> {
  return (await WailsGetConformContext()) as unknown as ConformContext
}

export async function inspectConformSpec(specPath: string): Promise<ConformSpecInfo> {
  return (await WailsInspectConformSpec(specPath)) as unknown as ConformSpecInfo
}

export async function runConformance(input: ConformRunInput): Promise<ConformRunOutput> {
  return (await WailsRunConformance(input as never)) as unknown as ConformRunOutput
}

export async function cancelConformance(): Promise<void> {
  await WailsCancelConformance()
}

// ---- Conformance server ----

export interface ConformanceServerResult {
  port: number
  error?: string
}

export async function startConformanceServer(): Promise<ConformanceServerResult> {
  const result = await WailsStartConformanceServer()
  return result as unknown as ConformanceServerResult
}

export async function stopConformanceServer(): Promise<void> {
  await WailsStopConformanceServer()
}

export async function openConformanceSpec(): Promise<string> {
  const result = await WailsOpenConformanceSpec()
  return result as unknown as string
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

export async function copyRequest(srcPath: string, destDir: string): Promise<string> {
  const result = await WailsCopyRequest(srcPath, destDir)
  return result as unknown as string
}

export async function moveScenario(srcPath: string, destFolderRelPath: string): Promise<string> {
  const result = await WailsMoveScenario(srcPath, destFolderRelPath)
  return result as unknown as string
}

export async function copyScenario(srcPath: string, destFolderRelPath: string): Promise<string> {
  const result = await WailsCopyScenario(srcPath, destFolderRelPath)
  return result as unknown as string
}

export async function loadResponseCache(reqPath: string): Promise<ResponseCacheData | null> {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const result = await WailsLoadResponseCache(reqPath)
  return result as unknown as ResponseCacheData | null
}

export async function clearResponseCache(reqPath: string): Promise<void> {
  await WailsClearResponseCache(reqPath)
}

// ---- Scenarios ----

export interface ScenarioInfo {
  title: string
  path: string
  folder?: string  // relative path from {workspace}/scenarios/, "/" separated, "" or absent for root
  stepCount?: number
  isFolder?: boolean  // sentinel: represents a directory entry, not a scenario file
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
  label: string
  notes?: string
  disabled?: boolean
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
  noFollowRedirects?: boolean
  auth?: AuthPayload
}

export interface ScenarioDetail {
  path: string
  title: string
  description: string
  specFile?: string
  variables?: ScenarioVariable[]
  steps: StepDetail[]
  error?: string
}

export interface ScenarioVariable {
  name: string
  value: string
  secret?: boolean
}

export interface AssertionResult {
  type: string
  passed: boolean
  message: string
}

export interface StepResult {
  label: string
  passed: boolean
  skipped: boolean
  statusCode: number
  durationMs: number
  error?: string
  extracted: Record<string, string>
  assertions: AssertionResult[]
  responseBody?: string
  responseHeaders?: Array<{ name: string; value: string }>
  attempts?: number
  // the request as it was sent (variables and dynamic values filled in, credentials masked)
  requestMethod?: string
  requestUrl?: string
  requestHeaders?: Array<{ name: string; value: string }>
  requestBody?: string
  // spec conformance (absent when the scenario has no spec)
  specState?: 'matched' | 'violations' | 'not-checked'
  specReason?: string
  specViolations?: string[]
}

export interface ScenarioCoverage {
  covered: number
  total: number
  percent: number
  exercised: string[]
  missed: string[]
}

export interface QualityGap {
  operation: string
  missing: number[]
}

export interface QualityReport {
  neverExercised: string[]
  happyPathOnly: string[]
  securedNeverAuthed: string[]
  errorsNeverTriggered: QualityGap[]
}

export interface AccumulatedCoverage {
  specFile: string
  covered: number
  total: number
  percent: number
  totalRuns: number
  passedRuns: number
  failedRuns: number
  lastRunAt?: string
  exercised: string[]
  missed: string[]
  operationResults?: Record<string, { lastPassed: boolean; lastRunAt: string }>
  quality?: QualityReport
}

export interface ScenarioResult {
  title: string
  passed: boolean
  durationMs: number
  steps: StepResult[]
  error?: string
  specError?: string
  coverage?: ScenarioCoverage
  accumulatedCoverage?: AccumulatedCoverage
  seed?: number
  variables?: ScenarioVariable[]
}

export async function listDynamicFunctions(): Promise<DynamicFunction[]> {
  return ((await WailsListDynamicFunctions()) ?? []) as unknown as DynamicFunction[]
}

export async function listScenarios(): Promise<ScenarioInfo[]> {
  const result = await WailsListScenarios()
  return (result ?? []) as unknown as ScenarioInfo[]
}

export async function loadScenario(path: string): Promise<ScenarioDetail> {
  const result = await WailsLoadScenario(path)
  return result as unknown as ScenarioDetail
}

export async function runScenario(path: string, envPath = '', fromStep = 0): Promise<ScenarioResult> {
  const result = await WailsRunScenario(path, envPath, fromStep)
  return result as unknown as ScenarioResult
}

export async function cancelScenario(): Promise<void> {
  await WailsCancelScenario()
}

export async function loadScenarioRaw(path: string): Promise<string> {
  return (await WailsLoadScenarioRaw(path)) as unknown as string
}

export async function saveScenarioRaw(path: string, content: string): Promise<string> {
  return (await WailsSaveScenarioRaw(path, content)) as unknown as string
}

export async function validateScenarioYAML(content: string): Promise<string> {
  return (await WailsValidateScenarioYAML(content)) as unknown as string
}

export interface CreateScenarioResult {
  path: string
  error?: string
}

export async function createScenario(name: string, folderPath = ''): Promise<CreateScenarioResult> {
  const result = await WailsCreateScenario(name, folderPath)
  return result as unknown as CreateScenarioResult
}

// Returns "" on success or an error message.
export async function createScenarioFolder(folderPath: string): Promise<string> {
  return (await WailsCreateScenarioFolder(folderPath)) as unknown as string
}

// Returns the new relative folder path on success, or "error: ..." on failure.
export async function renameScenarioFolder(folderPath: string, newName: string): Promise<string> {
  return (await WailsRenameScenarioFolder(folderPath, newName)) as unknown as string
}

// Returns "" on success or an error message.
export async function deleteScenarioFolder(folderPath: string): Promise<string> {
  return (await WailsDeleteScenarioFolder(folderPath)) as unknown as string
}

// Returns "" on success or an error message.
export async function saveScenarioResult(scenarioPath: string, result: ScenarioResult): Promise<void> {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  await WailsSaveScenarioResult(scenarioPath, result as any)
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

// ---- Scenario templates ----

export interface TemplateInfo {
  id: string
  title: string
  description: string
  category: string
  tags: string[]
  stepCount: number
  note?: string
}

export async function listTemplates(): Promise<TemplateInfo[]> {
  const result = await WailsListTemplates()
  return (result ?? []) as unknown as TemplateInfo[]
}

export async function createFromTemplate(templateID: string, name: string): Promise<CreateScenarioResult> {
  const result = await WailsCreateFromTemplate(templateID, name)
  return result as unknown as CreateScenarioResult
}

export type RunReportFormat = 'html' | 'junit' | 'json' | 'markdown' | 'text'

export interface RunReport {
  content: string
  error?: string
}

// Renders the latest results of the given scenarios with the Go reporting package, the same code
// (and the same redaction of secrets) the CLI uses.
export async function exportRunReport(format: RunReportFormat, paths: string[]): Promise<RunReport> {
  return (await WailsExportRunReport(format, paths)) as unknown as RunReport
}

export type CoverageReportFormat = 'csv' | 'markdown' | 'html'

export interface CoverageReport {
  content: string
  error?: string
}

// The report is rendered by the Go core, with the same code the CLI's coverage command uses.
export async function exportCoverageReport(specFile: string, format: CoverageReportFormat): Promise<CoverageReport> {
  return (await WailsExportCoverageReport(specFile, format)) as unknown as CoverageReport
}

export async function getAccumulatedCoverage(specFile: string): Promise<AccumulatedCoverage | null> {
  const result = await WailsGetAccumulatedCoverage(specFile)
  return result as unknown as AccumulatedCoverage | null
}

export async function resetAccumulatedCoverage(specFile: string): Promise<void> {
  await WailsResetAccumulatedCoverage(specFile)
}

export interface StepRunEntry {
  label: string
  passed: boolean
  skipped: boolean
  statusCode?: number
  durationMs?: number
  error?: string
}

export interface ScenarioRunEntry {
  id: number
  executedAt: string
  scenarioPath: string
  title: string
  passed: boolean
  durationMs: number
  stepTotal: number
  stepPassed: number
  stepFailed: number
  stepSkipped: number
  steps: StepRunEntry[]
}

export async function getScenarioRunHistory(limit = 100): Promise<ScenarioRunEntry[]> {
  const result = await WailsGetScenarioRunHistory(limit)
  return (result ?? []) as unknown as ScenarioRunEntry[]
}

export async function listAccumulatedCoverage(): Promise<AccumulatedCoverage[]> {
  const result = await WailsListAccumulatedCoverage()
  return (result ?? []) as unknown as AccumulatedCoverage[]
}

// ---- Folder run ----

export interface FolderRunItem {
  name: string
  path: string
  method: string
  statusCode: number
  durationMs: number
  error?: string
  passed: boolean
}

export async function runFolder(folderPath: string): Promise<FolderRunItem[]> {
  const result = await WailsRunFolder(folderPath)
  return (result ?? []) as unknown as FolderRunItem[]
}

// ---- Workspace export / import ----

export interface ScanImportResult {
  zipPath: string       // empty = user cancelled
  totalFiles: number
  conflicts: string[]   // relative paths that already exist
  error?: string
}

export interface ApplyImportResult {
  imported: number
  skipped: number
  renamed: number
  error?: string
}

// Returns "" on success or an error message. Empty string also returned on cancel.
export async function exportWorkspace(): Promise<string> {
  return (await WailsExportWorkspace()) as unknown as string
}

export async function workspaceHasSecrets(): Promise<boolean> {
  return (await WailsWorkspaceHasSecrets()) as unknown as boolean
}

export async function scanWorkspaceImport(): Promise<ScanImportResult> {
  const result = await WailsScanWorkspaceImport()
  return result as unknown as ScanImportResult
}

export async function applyWorkspaceImport(zipPath: string, strategy: string): Promise<ApplyImportResult> {
  const result = await WailsApplyWorkspaceImport(zipPath, strategy)
  return result as unknown as ApplyImportResult
}

// ---- Help ----

export async function getUserGuide(): Promise<string> {
  return (await WailsGetUserGuide()) as unknown as string
}

export async function getFormatSpec(): Promise<string> {
  return (await WailsGetFormatSpec()) as unknown as string
}

export async function getExampleScenario(): Promise<string> {
  return (await WailsGetExampleScenario()) as unknown as string
}

// The AI prompt document as shown in the Help panel (instructions only).
export async function getAIPromptInstructions(): Promise<string> {
  return (await WailsGetAIPromptInstructions()) as unknown as string
}

// The complete prompt to paste into an AI assistant: instructions + format spec + example.
export async function getAIPrompt(): Promise<string> {
  return (await WailsGetAIPrompt()) as unknown as string
}
