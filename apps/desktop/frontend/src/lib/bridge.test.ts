import { describe, it, expect, vi, beforeEach } from 'vitest'

vi.mock('@wailsjs/go/main/App')
vi.mock('@wailsjs/go/main/RequestService')
vi.mock('@wailsjs/go/main/WorkspaceService')
vi.mock('@wailsjs/go/main/OpenAPIService')
vi.mock('@wailsjs/go/main/ImportService')
vi.mock('@wailsjs/go/main/BundleService')
vi.mock('@wailsjs/go/main/SettingsService')
vi.mock('@wailsjs/go/main/DemoService')
vi.mock('@wailsjs/go/main/ScenarioService')
vi.mock('@wailsjs/go/main/HelpService')

import * as bridge from './bridge'
import * as App from '@wailsjs/go/main/App'
import * as RequestService from '@wailsjs/go/main/RequestService'
import * as WorkspaceService from '@wailsjs/go/main/WorkspaceService'
import * as OpenAPIService from '@wailsjs/go/main/OpenAPIService'
import * as ImportService from '@wailsjs/go/main/ImportService'
import * as BundleService from '@wailsjs/go/main/BundleService'
import * as SettingsService from '@wailsjs/go/main/SettingsService'
import * as DemoService from '@wailsjs/go/main/DemoService'
import * as ScenarioService from '@wailsjs/go/main/ScenarioService'
import * as HelpService from '@wailsjs/go/main/HelpService'

type Fn = (...args: unknown[]) => unknown
const mocked = (m: unknown, name: string) => (m as Record<string, ReturnType<typeof vi.fn>>)[name]
const exported = bridge as unknown as Record<string, Fn>

// [bridge function, wails module, wails function, arguments passed through]
const passthrough: Array<[string, unknown, string, unknown[]]> = [
  ['setHasDirtyTabs', App, 'SetHasDirtyTabs', [true]],
  ['quitApp', App, 'Exit', []],
  ['executeRequest', RequestService, 'Execute', [{ method: 'GET', url: 'u' }]],
  ['cancelRequest', RequestService, 'CancelRequest', []],
  ['clearHistory', RequestService, 'ClearHistory', []],
  ['openWorkspace', WorkspaceService, 'OpenWorkspace', []],
  ['newWorkspace', WorkspaceService, 'NewWorkspace', []],
  ['closeWorkspace', WorkspaceService, 'CloseWorkspace', []],
  ['getRecentWorkspaces', WorkspaceService, 'GetRecentWorkspaces', []],
  ['openWorkspaceByPath', WorkspaceService, 'OpenWorkspaceByPath', ['/p']],
  ['saveContentToFile', WorkspaceService, 'SaveContentToFile', ['c', 'n.txt']],
  ['getWorkspacePath', WorkspaceService, 'GetWorkspacePath', []],
  ['loadRequest', WorkspaceService, 'LoadRequest', ['/r']],
  ['saveRequest', WorkspaceService, 'SaveRequest', [{ title: 't' }]],
  ['openRequest', WorkspaceService, 'OpenRequest', []],
  ['selectEnvironment', WorkspaceService, 'SelectEnvironment', ['/e']],
  ['getActiveEnvironment', WorkspaceService, 'GetActiveEnvironment', []],
  ['getEnvironment', WorkspaceService, 'GetEnvironment', ['/e']],
  ['saveEnvironment', WorkspaceService, 'SaveEnvironment', [{ path: '', name: 'n', variables: [] }]],
  ['deleteEnvironment', WorkspaceService, 'DeleteEnvironment', ['/e']],
  ['getEnvironmentRaw', WorkspaceService, 'GetEnvironmentRaw', ['/e']],
  ['saveEnvironmentRaw', WorkspaceService, 'SaveEnvironmentRaw', ['/e', 'yaml']],
  ['validateEnvironmentYAML', WorkspaceService, 'ValidateEnvironmentYAML', ['yaml']],
  ['deleteRequest', WorkspaceService, 'DeleteRequest', ['/r']],
  ['createFolder', WorkspaceService, 'CreateFolder', ['/p', 'n']],
  ['deleteFolder', WorkspaceService, 'DeleteFolder', ['/f']],
  ['renameFolder', WorkspaceService, 'RenameFolder', ['/f', 'n']],
  ['renameRequest', WorkspaceService, 'RenameRequest', ['/r', 'n']],
  ['saveResponseCache', WorkspaceService, 'SaveResponseCache', ['/r', { statusCode: 200 }]],
  ['loadResponseCache', WorkspaceService, 'LoadResponseCache', ['/r']],
  ['clearResponseCache', WorkspaceService, 'ClearResponseCache', ['/r']],
  ['openInSystemViewer', WorkspaceService, 'OpenInSystemViewer', ['body', 'json']],
  ['openInExplorer', WorkspaceService, 'OpenInExplorer', ['/p']],
  ['moveRequest', WorkspaceService, 'MoveRequest', ['/a', '/d']],
  ['copyRequest', WorkspaceService, 'CopyRequest', ['/a', '/d']],
  ['exportWorkspace', WorkspaceService, 'ExportWorkspace', []],
  ['workspaceHasSecrets', WorkspaceService, 'WorkspaceHasSecrets', []],
  ['scanWorkspaceImport', WorkspaceService, 'ScanWorkspaceImport', []],
  ['applyWorkspaceImport', WorkspaceService, 'ApplyWorkspaceImport', ['/z.zip', 'skip']],
  ['importOpenAPISpec', OpenAPIService, 'ImportOpenAPISpec', ['http://base']],
  ['importOpenAPISpecFromPath', OpenAPIService, 'ImportOpenAPISpecFromPath', ['/spec', 'http://base']],
  ['importOpenAPISpecFromURL', OpenAPIService, 'ImportOpenAPISpecFromURL', ['http://spec', 'http://base']],
  ['pickBundleFile', BundleService, 'PickBundleFile', []],
  ['exportBundle', BundleService, 'ExportBundle', []],
  ['previewBundle', BundleService, 'PreviewBundle', ['text', '/spec.yaml', 'rename']],
  ['importBundle', BundleService, 'ImportBundle', ['text', '', 'skip']],
  ['importPostmanCollection', ImportService, 'ImportPostmanCollection', []],
  ['importInsomniaExport', ImportService, 'ImportInsomniaExport', []],
  ['getSettings', SettingsService, 'GetSettings', []],
  ['saveSettings', SettingsService, 'SaveSettings', [{ theme: 'dark' }]],
  ['resetSettings', SettingsService, 'ResetSettings', []],
  ['startDemo', DemoService, 'StartDemo', []],
  ['stopDemo', DemoService, 'StopDemo', []],
  ['isDemoRunning', DemoService, 'IsRunning', []],
  ['getDemoPort', DemoService, 'GetPort', []],
  ['openDemoSpec', DemoService, 'OpenSpecInBrowser', []],
  ['loadScenario', ScenarioService, 'LoadScenario', ['/s']],
  ['runScenario', ScenarioService, 'RunScenario', ['/s', '/e', 2]],
  ['cancelScenario', ScenarioService, 'CancelScenario', []],
  ['loadScenarioRaw', ScenarioService, 'LoadScenarioRaw', ['/s']],
  ['saveScenarioRaw', ScenarioService, 'SaveScenarioRaw', ['/s', 'yaml']],
  ['validateScenarioYAML', ScenarioService, 'ValidateScenarioYAML', ['yaml']],
  ['createScenario', ScenarioService, 'CreateScenario', ['n', 'auth']],
  ['createScenarioFolder', ScenarioService, 'CreateScenarioFolder', ['auth']],
  ['renameScenarioFolder', ScenarioService, 'RenameScenarioFolder', ['auth', 'n']],
  ['deleteScenarioFolder', ScenarioService, 'DeleteScenarioFolder', ['auth']],
  ['saveScenarioResult', ScenarioService, 'SaveScenarioResult', ['/s', { title: 't' }]],
  ['loadScenarioResult', ScenarioService, 'LoadScenarioResult', ['/s']],
  ['duplicateScenario', ScenarioService, 'DuplicateScenario', ['/s']],
  ['deleteScenario', ScenarioService, 'DeleteScenario', ['/s']],
  ['renameScenario', ScenarioService, 'RenameScenario', ['/s', 'n']],
  ['moveScenario', ScenarioService, 'MoveScenario', ['/s', 'auth']],
  ['copyScenario', ScenarioService, 'CopyScenario', ['/s', 'auth']],
  ['createFromTemplate', ScenarioService, 'CreateFromTemplate', ['tpl', 'n']],
  ['getAccumulatedCoverage', ScenarioService, 'GetAccumulatedCoverage', ['spec.yaml']],
  ['resetAccumulatedCoverage', ScenarioService, 'ResetAccumulatedCoverage', ['spec.yaml']],
  ['getUserGuide', HelpService, 'GetUserGuide', []],
  ['getFormatSpec', HelpService, 'GetFormatSpec', []],
  ['getExampleScenario', HelpService, 'GetExampleScenario', []],
  ['getAIPromptInstructions', HelpService, 'GetAIPromptInstructions', []],
  ['getAIPrompt', HelpService, 'GetAIPrompt', []],
]

// Functions whose result is normalised to [] when the backend returns null (Go nil slices).
const listFns: Array<[string, unknown, string, unknown[]]> = [
  ['getHistory', RequestService, 'GetHistory', []],
  ['listRequests', WorkspaceService, 'ListRequests', []],
  ['listEnvironments', WorkspaceService, 'ListEnvironments', []],
  ['listScenarios', ScenarioService, 'ListScenarios', []],
  ['listTemplates', ScenarioService, 'ListTemplates', []],
  ['listAccumulatedCoverage', ScenarioService, 'ListAccumulatedCoverage', []],
  ['getScenarioRunHistory', ScenarioService, 'GetScenarioRunHistory', [100]],
  ['runFolder', ScenarioService, 'RunFolder', ['/f']],
]

beforeEach(() => vi.resetAllMocks())

describe('bridge passthrough', () => {
  it.each(passthrough)('%s calls the matching binding and returns its result', async (name, mod, wailsName, args) => {
    const sentinel = { marker: name }
    mocked(mod, wailsName).mockResolvedValue(sentinel)

    const result = await exported[name](...args)

    expect(mocked(mod, wailsName)).toHaveBeenCalledTimes(1)
    expect(mocked(mod, wailsName)).toHaveBeenCalledWith(...args)
    const returnsVoid = ['setHasDirtyTabs', 'quitApp', 'cancelRequest', 'clearHistory', 'closeWorkspace', 'selectEnvironment',
      'deleteEnvironment', 'deleteRequest', 'deleteFolder', 'saveResponseCache', 'clearResponseCache', 'stopDemo',
      'cancelScenario', 'saveScenarioResult', 'resetAccumulatedCoverage']
    if (returnsVoid.includes(name)) expect(result).toBeUndefined()
    else expect(result).toBe(sentinel)
  })

  it.each(passthrough)('%s propagates backend rejections', async (name, mod, wailsName, args) => {
    mocked(mod, wailsName).mockRejectedValue(new Error(`${name} failed`))
    await expect(exported[name](...args)).rejects.toThrow(`${name} failed`)
  })
})

describe('bridge list normalisation', () => {
  it.each(listFns)('%s returns [] when the backend returns null', async (name, mod, wailsName, args) => {
    mocked(mod, wailsName).mockResolvedValue(null)
    expect(await exported[name](...args)).toEqual([])
  })

  it.each(listFns)('%s returns the backend list unchanged', async (name, mod, wailsName, args) => {
    const list = [{ id: 1 }]
    mocked(mod, wailsName).mockResolvedValue(list)
    expect(await exported[name](...args)).toBe(list)
  })
})

describe('bridge defaults', () => {
  it('pickFile defaults its dialog title', async () => {
    mocked(WorkspaceService, 'PickFile').mockResolvedValue('/f')
    await bridge.pickFile()
    expect(WorkspaceService.PickFile).toHaveBeenCalledWith('Select file')
    await bridge.pickFile('Custom')
    expect(WorkspaceService.PickFile).toHaveBeenLastCalledWith('Custom')
  })

  it('OpenAPI imports default the base URL to empty', async () => {
    await bridge.importOpenAPISpec()
    expect(OpenAPIService.ImportOpenAPISpec).toHaveBeenCalledWith('')
    await bridge.importOpenAPISpecFromPath('/p')
    expect(OpenAPIService.ImportOpenAPISpecFromPath).toHaveBeenCalledWith('/p', '')
    await bridge.importOpenAPISpecFromURL('http://x')
    expect(OpenAPIService.ImportOpenAPISpecFromURL).toHaveBeenCalledWith('http://x', '')
  })

  it('scenario helpers default their optional arguments', async () => {
    await bridge.runScenario('/s')
    expect(ScenarioService.RunScenario).toHaveBeenCalledWith('/s', '', 0)
    await bridge.createScenario('n')
    expect(ScenarioService.CreateScenario).toHaveBeenCalledWith('n', '')
    await bridge.getScenarioRunHistory()
    expect(ScenarioService.GetScenarioRunHistory).toHaveBeenCalledWith(100)
  })
})
