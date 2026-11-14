/**
 * Injected into the page via page.addInitScript before React loads.
 * Sets up window.go (Wails IPC) and window.runtime stubs so the app
 * renders without a live Go backend.
 *
 * Tests can override specific behaviors after page.goto() via page.evaluate:
 *   await page.evaluate(() => {
 *     (window as any).__bridge.executeResult = { statusCode: 404, ... }
 *   })
 */
export function setupMockBridge() {
  const noop = () => Promise.resolve(undefined)
  const noopNull = () => Promise.resolve(null)

  const DEFAULT_SETTINGS = {
    theme: 'dark',
    fontSize: 13,
    defaultTimeoutMs: 30000,
    proxyUrl: '',
    tlsSkipVerify: false,
  }

  const DEFAULT_RESPONSE = {
    statusCode: 200,
    status: 'OK',
    body: '{"ok":true}',
    headers: [{ name: 'Content-Type', value: 'application/json' }],
    durationMs: 123,
    size: 11,
    error: '',
    timing: { dnsMs: 1, connectMs: 5, tlsMs: 0, ttfbMs: 100, transferMs: 17 },
  }

  // Mutable bridge state: tests can overwrite fields after goto()
  ;(window as unknown as Record<string, unknown>)['__bridge'] = {
    settings: { ...DEFAULT_SETTINGS },
    executeResult: { ...DEFAULT_RESPONSE },
    history: [] as unknown[],
    workspacePath: '/workspace',
    fileTree: [] as unknown[],
    environments: [] as unknown[],
    activeEnv: null,
    scenarios: [] as unknown[],
    scenarioDetail: null as unknown,
    scenarioResult: null as unknown,
    scenarioRaw: '' as unknown,
    scenarioResultCache: {} as Record<string, unknown>,
    recentWorkspaces: [] as unknown[],
  }

  const bridge = (window as unknown as Record<string, Record<string, unknown>>)['__bridge']

  ;(window as unknown as Record<string, unknown>)['go'] = {
    main: {
      App: {
        SetHasDirtyTabs: noop,
      },
      RequestService: {
        Execute: () => Promise.resolve(bridge['executeResult']),
        CancelRequest: noop,
        GetHistory: () => Promise.resolve(bridge['history']),
        ClearHistory: () => { bridge['history'] = []; return Promise.resolve() },
        RebuildExecutor: noop,
      },
      WorkspaceService: {
        GetWorkspacePath: () => Promise.resolve(bridge['workspacePath']),
        ListRequests: () => Promise.resolve(bridge['fileTree']),
        ListEnvironments: () => Promise.resolve(bridge['environments']),
        GetActiveEnvironment: () => Promise.resolve(bridge['activeEnv']),
        GetEnvironment: () => Promise.resolve({ name: '', variables: [] }),
        ActiveEnvironment: noopNull,
        OpenWorkspace: () => { bridge['workspacePath'] = '/workspace'; return Promise.resolve('/workspace') },
        NewWorkspace: () => { bridge['workspacePath'] = '/workspace'; return Promise.resolve('/workspace') },
        CloseWorkspace: () => { bridge['workspacePath'] = ''; return Promise.resolve() },
        OpenWorkspaceByPath: (path: string) => { bridge['workspacePath'] = path; return Promise.resolve(path) },
        GetRecentWorkspaces: () => Promise.resolve(bridge['recentWorkspaces'] ?? []),
        SaveRequest: (r: Record<string, string>) =>
          Promise.resolve('/workspace/' + (r?.name || 'req') + '.yaml'),
        LoadRequest: noopNull,
        OpenRequest: noopNull,
        DeleteRequest: noop,
        CreateFolder: noop,
        DeleteFolder: noop,
        RenameRequest: (p: string) => Promise.resolve(p),
        RenameFolder: (p: string) => Promise.resolve(p),
        MoveRequest: (src: string) => Promise.resolve(src),
        SaveResponseCache: noop,
        LoadResponseCache: noopNull,
        ClearResponseCache: noop,
        PickFile: noopNull,
        SaveContentToFile: () => Promise.resolve(''),
        OpenInExplorer: () => Promise.resolve(''),
        OpenInSystemViewer: noop,
        SaveEnvironment: (e: Record<string, string>) =>
          Promise.resolve('/workspace/environments/' + (e?.name || 'env') + '.yaml'),
        DeleteEnvironment: noop,
        SelectEnvironment: noop,
      },
      SettingsService: {
        GetSettings: () => Promise.resolve({ ...(bridge['settings'] as object) }),
        SaveSettings: (s: unknown) => { bridge['settings'] = s as Record<string, unknown>; return Promise.resolve(s) },
        ResetSettings: () => {
          bridge['settings'] = { ...DEFAULT_SETTINGS }
          return Promise.resolve({ ...DEFAULT_SETTINGS })
        },
      },
      DemoService: {
        StartDemo: () => Promise.resolve({ port: 8080, error: '' }),
        StopDemo: noop,
        IsRunning: () => Promise.resolve(false),
        GetPort: () => Promise.resolve(0),
      },
      OpenAPIService: {
        ImportOpenAPISpec: () => Promise.resolve({ count: 0, error: '' }),
        ImportOpenAPISpecFromPath: () => Promise.resolve({ count: 0, error: '' }),
        ImportOpenAPISpecFromURL: () => Promise.resolve({ count: 0, error: '' }),
      },
      ImportService: {
        ImportPostmanCollection: () => Promise.resolve({ count: 0, error: '' }),
        ImportInsomniaExport: () => Promise.resolve({ count: 0, error: '' }),
      },
      ScenarioService: {
        ListScenarios: () => Promise.resolve(bridge['scenarios']),
        ListTemplates: () => Promise.resolve([]),
        RunFolder: () => Promise.resolve([]),
        CreateFromTemplate: (templateId: string) => {
          const path = `/workspace/scenarios/${templateId}.yaml`
          return Promise.resolve({ path, error: '' })
        },
        LoadScenario: () => Promise.resolve(bridge['scenarioDetail'] ?? { path: '', title: '', description: '', error: '', steps: [] }),
        RunScenario: (_path: string, _envPath: string) => Promise.resolve(bridge['scenarioResult'] ?? { title: '', error: '', passed: true, durationMs: 0, steps: [] }),
        LoadScenarioRaw: () => Promise.resolve(bridge['scenarioRaw'] ?? ''),
        SaveScenarioRaw: (_path: string, content: string) => { bridge['scenarioRaw'] = content; return Promise.resolve('') },
        SaveScenarioResult: (path: string, result: unknown) => {
          (bridge['scenarioResultCache'] as Record<string, unknown>)[path] = result
          return Promise.resolve('')
        },
        LoadScenarioResult: (path: string) => {
          const cached = (bridge['scenarioResultCache'] as Record<string, unknown>)[path]
          return Promise.resolve(cached ?? null)
        },
        CreateScenario: (name: string) => {
          const slug = name.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '')
          const path = `/workspace/scenarios/${slug || 'scenario'}.yaml`
          const newScenario = { title: name, path, stepCount: 1 }
          ;(bridge['scenarios'] as unknown[]).push(newScenario)
          bridge['scenarioDetail'] = { path, title: name, description: '', steps: [], error: '' }
          bridge['scenarioRaw'] = `schemaVersion: "1"\ntitle: ${name}\nsteps:\n  - title: Step 1\n`
          return Promise.resolve({ path, error: '' })
        },
        DuplicateScenario: (path: string) => {
          const scenarios = bridge['scenarios'] as Array<{path: string; name: string}>
          const src = scenarios.find(s => s.path === path)
          if (!src) return Promise.resolve({ path: '', error: 'not found' })
          const dir = path.substring(0, path.lastIndexOf('/'))
          const base = path.substring(path.lastIndexOf('/') + 1).replace(/\.(yaml|yml|json)$/i, '')
          const newPath = `${dir}/${base}-copy.yaml`
          const newName = src.name + ' Copy'
          scenarios.push({ path: newPath, name: newName, stepCount: src.stepCount ?? 0 })
          return Promise.resolve({ path: newPath, error: '' })
        },
        DeleteScenario: (path: string) => {
          bridge['scenarios'] = (bridge['scenarios'] as Array<{path: string}>).filter(s => s.path !== path)
          return Promise.resolve('')
        },
        RenameScenario: (path: string, newName: string) => {
          const slug = newName.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '')
          const dir = path.substring(0, path.lastIndexOf('/'))
          const newPath = `${dir}/${slug || 'scenario'}.yaml`
          bridge['scenarios'] = (bridge['scenarios'] as Array<{path: string; name: string}>).map(
            s => s.path === path ? { ...s, path: newPath, name: newName } : s
          )
          return Promise.resolve(newPath)
        },
      },
    },
  }

  // Wails runtime stubs (window.runtime is used by wailsjs/runtime/runtime.js)
  ;(window as unknown as Record<string, unknown>)['runtime'] = {
    LogPrint: noop, LogTrace: noop, LogDebug: noop,
    LogInfo: noop, LogWarning: noop, LogError: noop,
    EventsOn: () => {}, EventsOff: () => {}, EventsOnce: () => {}, EventsEmit: () => {},
    WindowMinimise: () => {}, WindowMaximise: () => {}, WindowUnmaximise: () => {},
    WindowIsMaximised: () => Promise.resolve(false),
    WindowClose: () => {}, WindowSetTitle: () => {},
    BrowserOpenURL: () => {},
  }
}
